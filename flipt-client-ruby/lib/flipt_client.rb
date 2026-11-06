# frozen_string_literal: true

require 'flipt_client/version'
require 'flipt_client/models'
require 'flipt_client/state_updates'
require 'ffi'
require 'json'

module Flipt
  class Error < StandardError; end
  class ValidationError < Error; end
  class EvaluationError < Error; end

  # Client is a Ruby Client Side Evaluation Library for Flipt
  class Client
    extend FFI::Library

    FLIPTENGINE = 'fliptengine'

    LIB_FILES = {
      /arm64-darwin/ => "ext/darwin_aarch64/lib#{FLIPTENGINE}.dylib",
      /x86_64-darwin/ => "ext/darwin_x86_64/lib#{FLIPTENGINE}.dylib",
      /arm64-linux|aarch64-linux/ => "ext/linux_aarch64/lib#{FLIPTENGINE}.so",
      /x86_64-linux/ => "ext/linux_x86_64/lib#{FLIPTENGINE}.so",
      /x86_64-mingw32/ => "ext/windows_x86_64/#{FLIPTENGINE}.dll"
    }.freeze

    def self.libfile
      arch = RbConfig::CONFIG['arch']
      LIB_FILES.each do |pattern, path|
        return path if arch.match?(pattern)
      end
      raise "unsupported platform #{arch}"
    end

    ffi_lib File.expand_path(libfile, __dir__)

    # void *initialize_engine(const char *opts);
    # Does the initial fetch, and destroy_engine stops the fetch thread, so both release the GVL
    # to let other Ruby threads run while they wait on the network.
    attach_function :initialize_engine, [:string], :pointer, blocking: true
    # void destroy_engine(void *engine_ptr);
    attach_function :destroy_engine, [:pointer], :void, blocking: true
    # const char *evaluate_variant(void *engine_ptr, const char *evaluation_request);
    attach_function :evaluate_variant, %i[pointer string], :strptr
    # const char *evaluate_boolean(void *engine_ptr, const char *evaluation_request);
    attach_function :evaluate_boolean, %i[pointer string], :strptr
    # const char *evaluate_batch(void *engine_ptr, const char *batch_evaluation_request);
    attach_function :evaluate_batch, %i[pointer string], :strptr
    # const char *list_flags(void *engine_ptr);
    attach_function :list_flags, [:pointer], :strptr
    # void destroy_string(const char *ptr);
    attach_function :destroy_string, [:pointer], :void
    # const char *get_snapshot(void *engine_ptr);
    attach_function :get_snapshot, [:pointer], :strptr
    # void *subscribe_state_updates(void *engine_ptr);
    attach_function :subscribe_state_updates, [:pointer], :pointer
    # const char *wait_for_state_update(void *subscription_ptr);
    # Blocks until flag state changes or the subscription closes, so it releases the GVL.
    attach_function :wait_for_state_update, [:pointer], :strptr, blocking: true
    # void close_state_subscription(void *subscription_ptr);
    attach_function :close_state_subscription, [:pointer], :void
    # void destroy_state_subscription(void *subscription_ptr);
    attach_function :destroy_state_subscription, [:pointer], :void

    # Create a new Flipt client
    #
    # @param opts [Hash] options
    # @option opts [String] :environment Flipt environment (default: 'default')
    # @option opts [String] :namespace Flipt namespace (default: 'default')
    # @option opts [String] :url Flipt server url
    # @option opts [AuthenticationStrategy] :authentication strategy to authenticate with Flipt
    # @option opts [Integer] :request_timeout timeout in seconds for the request
    # @option opts [Integer] :update_interval interval in seconds to update the cache
    # @option opts [String] :reference reference to use for namespace data
    # @option opts [Symbol] :fetch_mode fetch mode to use for the client (:polling or :streaming).
    #   Note: Streaming is currently only supported when using the SDK with Flipt Cloud or Flipt v2.
    # @option opts [Symbol] :error_strategy error strategy to use for the client (:fail or :fallback).
    # @option opts [String] :snapshot snapshot to use when initializing the client
    # @option opts [TlsConfig] :tls_config TLS configuration for connecting to servers with custom certificates
    # @option opts [#call] :on_state_update called with no arguments after the client applies changed flag state
    def initialize(**opts)
      @namespace = opts.fetch(:namespace, 'default')
      on_state_update = opts.delete(:on_state_update)

      opts[:authentication] = validate_authentication(opts.fetch(:authentication, NoAuthentication.new))
      opts[:fetch_mode] = validate_fetch_mode(opts.fetch(:fetch_mode, :polling))
      opts[:error_strategy] = validate_error_strategy(opts.fetch(:error_strategy, :fail))
      opts[:tls_config] = validate_tls_config(opts.fetch(:tls_config, nil))

      @engine = self.class.initialize_engine(opts.to_json)
      ObjectSpace.define_finalizer(self, self.class.finalize(@engine))
      @state_updates = StateUpdates.new(@engine)

      return if on_state_update.nil?

      begin
        add_state_update_listener(on_state_update)
      rescue StandardError
        # The caller never gets a client to close, so tear the engine down here.
        close
        raise
      end
    end

    # Close the client and release the engine. When called from a thread other than the state
    # update thread, waits for a running listener to return first. Closing twice is a no-op.
    def close
      return unless @state_updates.close

      # Clear the pointer before destroying: destroy_engine releases the GVL, so other threads
      # keep running and must see a closed client rather than a pointer being freed.
      engine = @engine
      @engine = nil
      ObjectSpace.undefine_finalizer(self)
      self.class.destroy_engine(engine)
    end

    # Add a listener that is called after each change to flag state. Read the flags you care
    # about once after adding a listener: it is called for changes applied after it was added,
    # and may occasionally see one extra call.
    #
    # Listeners run in the order they were added, on a single thread named flipt-state-update.
    # Updates that arrive while they run are coalesced into a single follow-up call. A listener
    # may evaluate flags, add or remove listeners, or call close. close from another thread waits
    # for a running listener to return. An error raised by one listener is logged and does not
    # stop the others. The client must be closed to stop its update thread.
    #
    # @param listener [#call] the listener to call, or pass a block
    # @return [Subscription] a subscription that removes the listener when closed
    def add_state_update_listener(listener = nil, &block)
      listener ||= block
      raise ValidationError, 'listener must respond to call' unless listener.respond_to?(:call)

      @state_updates.add(listener)
    end

    def self.finalize(engine)
      proc { destroy_engine(engine) }
    end

    # Evaluate a variant flag for a given request
    #
    # @param flag_key [String]
    # @param entity_id [String]
    # @param context [Hash]
    def evaluate_variant(flag_key:, entity_id:, context: {})
      validate_evaluation_request(flag_key, entity_id, context)
      req = { flag_key: flag_key, entity_id: entity_id, context: context }
      resp, ptr = self.class.evaluate_variant(open_engine, req.to_json)
      ptr = FFI::AutoPointer.new(ptr, Client.method(:destroy_string))
      data = JSON.parse(resp)
      raise EvaluationError, data['error_message'] if data['status'] != 'success'

      r = data['result']
      VariantEvaluationResponse.new(
        flag_key: r['flag_key'],
        match: r['match'],
        reason: r['reason'],
        variant_key: r['variant_key'],
        variant_attachment: r['variant_attachment'],
        segment_keys: r['segment_keys'] || []
      )
    end

    # Evaluate a boolean flag for a given request
    #
    # @param flag_key [String]
    # @param entity_id [String]
    # @param context [Hash]
    def evaluate_boolean(flag_key:, entity_id:, context: {})
      validate_evaluation_request(flag_key, entity_id, context)
      req = { flag_key: flag_key, entity_id: entity_id, context: context }
      resp, ptr = self.class.evaluate_boolean(open_engine, req.to_json)
      ptr = FFI::AutoPointer.new(ptr, Client.method(:destroy_string))
      data = JSON.parse(resp)
      raise EvaluationError, data['error_message'] if data['status'] != 'success'

      r = data['result']
      BooleanEvaluationResponse.new(
        flag_key: r['flag_key'],
        enabled: r['enabled'],
        reason: r['reason'],
        segment_keys: r['segment_keys'] || []
      )
    end

    # Evaluate a batch of flags for a given request
    #
    # @param requests [Array<Hash>] batch evaluation request
    #   - :entity_id [String] entity id
    #   - :flag_key [String] flag key
    def evaluate_batch(requests:)
      unless requests.is_a?(Array)
        raise ValidationError, 'requests must be an array of evaluation requests'
      end

      requests.each do |request|
        validate_evaluation_request(request[:flag_key], request[:entity_id], request[:context] || {})
      end
      resp, ptr = self.class.evaluate_batch(open_engine, requests.to_json)
      ptr = FFI::AutoPointer.new(ptr, Client.method(:destroy_string))
      data = JSON.parse(resp)
      raise EvaluationError, data['error_message'] if data['status'] != 'success'

      responses = (data['result']['responses'] || []).map do |r|
        case r['type']
        when 'VARIANT_EVALUATION_RESPONSE_TYPE'
          v = r['variant_evaluation_response']
          VariantEvaluationResponse.new(
            flag_key: v['flag_key'],
            match: v['match'],
            reason: v['reason'],
            variant_key: v['variant_key'],
            variant_attachment: v['variant_attachment'],
            segment_keys: v['segment_keys'] || []
          )
        when 'BOOLEAN_EVALUATION_RESPONSE_TYPE'
          b = r['boolean_evaluation_response']
          BooleanEvaluationResponse.new(
            flag_key: b['flag_key'],
            enabled: b['enabled'],
            reason: b['reason'],
            segment_keys: b['segment_keys'] || []
          )
        when 'ERROR_EVALUATION_RESPONSE_TYPE'
          e = r['error_evaluation_response']
          ErrorEvaluationResponse.new(
            flag_key: e['flag_key'],
            namespace_key: e['namespace_key'],
            reason: e['reason'],
            error_message: e['error_message']
          )
        else
          raise EvaluationError, "Unknown response type encountered: #{r['type']}"
        end
      end
      BatchEvaluationResponse.new(responses: responses)
    end

    # List all flags in the namespace
    def list_flags
      resp, ptr = self.class.list_flags(open_engine)
      ptr = FFI::AutoPointer.new(ptr, Client.method(:destroy_string))
      data = JSON.parse(resp)
      raise Error, data['error_message'] if data['status'] != 'success'

      data['result']
    end

    # Get the snapshot of the current flag state
    def snapshot
      resp, ptr = self.class.get_snapshot(open_engine)
      ptr = FFI::AutoPointer.new(ptr, Client.method(:destroy_string))
      resp
    end

    private

    def open_engine
      engine = @engine
      raise Error, 'client is closed' if engine.nil?

      engine
    end

    def validate_evaluation_request(flag_key, entity_id, context)
      raise ValidationError, 'flag_key is required' if flag_key.nil? || flag_key.empty?
      raise ValidationError, 'entity_id is required' if entity_id.nil? || entity_id.empty?
      return if context.is_a?(Hash)

      raise ValidationError, 'context must be a Hash<String, String>'
    end

    def validate_authentication(authentication)
      return authentication.strategy if authentication.is_a?(AuthenticationStrategy)

      raise ValidationError, 'invalid authentication strategy'
    end

    def validate_fetch_mode(fetch_mode)
      return fetch_mode if %i[polling streaming].include?(fetch_mode)

      raise ValidationError, 'invalid fetch mode'
    end

    def validate_error_strategy(error_strategy)
      return error_strategy if %i[fail fallback].include?(error_strategy)

      raise ValidationError, 'invalid error strategy'
    end

    def validate_tls_config(tls_config)
      return nil if tls_config.nil?
      return tls_config.to_h if tls_config.is_a?(TlsConfig)

      raise ValidationError, 'invalid tls_config: must be TlsConfig instance'
    end
  end

  # Deprecation shim for EvaluationClient
  class EvaluationClient < Client
    def initialize(*args, **kwargs)
      warn '[DEPRECATION] `EvaluationClient` is deprecated. Please use `Client` instead.'
      super
    end
  end
end
