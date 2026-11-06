# frozen_string_literal: true

module Flipt
  # Subscription is a registered state update listener. Closing it unsubscribes; closing twice is a no-op.
  class Subscription
    def initialize(&unsubscribe)
      @unsubscribe = unsubscribe
    end

    # Remove the listener
    def close
      @unsubscribe.call
      nil
    end
  end

  # StateUpdates owns the single thread that waits on the engine for flag state changes and fans
  # out to the client's listeners. It is internal to Client.
  class StateUpdates
    def initialize(engine)
      @engine = engine
      # Guards everything below. Never held while running listeners or joining the thread.
      @mutex = Mutex.new
      @closed = false
      # Replaced, never mutated, so the thread can iterate a snapshot without the mutex.
      @listeners = [].freeze
      # Started on the first listener. It owns the native subscription and frees it on exit.
      @thread = nil
      @subscription = nil
    end

    def add(listener)
      # A distinct wrapper per call, so adding the same callable twice yields two independent
      # subscriptions.
      entry = proc { listener.call }
      @mutex.synchronize do
        raise Error, 'client is closed' if @closed

        ensure_thread
        @listeners = (@listeners + [entry]).freeze
      end
      Subscription.new { remove(entry) }
    end

    # Marks the client closed, wakes the thread and waits for a running listener to return, unless
    # called from that listener. Returns false if already closed.
    def close
      thread = nil
      @mutex.synchronize do
        return false if @closed

        @closed = true
        thread = @thread
        # Wakes the thread if it is waiting. It still owns and frees the handle.
        Client.close_state_subscription(@subscription) if @subscription
      end
      # Let a running listener finish before the engine it may be evaluating against is freed.
      thread.join if thread && !thread.equal?(Thread.current)
      @listeners = [].freeze
      true
    end

    private

    def remove(entry)
      @mutex.synchronize do
        @listeners = @listeners.reject { |existing| existing.equal?(entry) }.freeze
      end
    end

    # Called with the mutex held. The thread runs until close, so adding or removing listeners
    # never calls into the engine, including from inside a listener.
    def ensure_thread
      return if @thread

      subscription = Client.subscribe_state_updates(@engine)
      raise EvaluationError, 'Failed to subscribe to state updates' if subscription.null?

      begin
        thread = Thread.new { run(subscription) }
      rescue StandardError
        Client.destroy_state_subscription(subscription)
        raise
      end
      thread.name = 'flipt-state-update'
      # The thread can't exit before these are set: its cleanup takes the mutex we hold.
      @subscription = subscription
      @thread = thread
    end

    def run(subscription)
      until @closed
        resp, ptr = Client.wait_for_state_update(subscription)
        Client.destroy_string(ptr)
        data = JSON.parse(resp)
        if data['status'] != 'success'
          warn "[flipt] Failed to wait for state update: #{data['error_message'] || 'Unknown error'}"
          return
        end
        return if data['result'] != 'changed'

        @listeners.each do |listener|
          # Leaves the loop too, since the until condition sees @closed.
          break if @closed

          begin
            listener.call
          # Exception, so a listener raising NotImplementedError or calling exit is logged and does
          # not silently stop notifications for the others.
          rescue Exception => e # rubocop:disable Lint/RescueException
            warn "[flipt] State update listener raised: #{e.class}: #{e.message}"
          end
        end
      end
    rescue StandardError => e
      warn "[flipt] State update thread failed: #{e.class}: #{e.message}"
    ensure
      # This thread is the only one that waits on the subscription, so it frees it. Under the
      # mutex, so close never wakes a handle that is already freed.
      @mutex.synchronize do
        @subscription = nil
        Client.destroy_state_subscription(subscription)
        # If this thread died early, the next listener starts a new one.
        @thread = nil if @thread.equal?(Thread.current)
      end
    end
  end
end
