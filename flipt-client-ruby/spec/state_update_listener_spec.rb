# frozen_string_literal: true

require 'socket'
require 'timeout'
require_relative '../lib/flipt_client'

EMPTY = '{"namespace":{"key":"default"},"flags":[]}'
WITH_FLAG = '{"namespace":{"key":"default"},"flags":[{"key":"new_flag","name":"New",' \
            '"type":"BOOLEAN_FLAG_TYPE","enabled":true}]}'
BARRIER = '{"namespace":{"key":"default"},"flags":[{"key":"barrier","name":"Barrier",' \
          '"type":"BOOLEAN_FLAG_TYPE","enabled":true}]}'
END_STREAM = '<end>'

# Pops from the queue, or returns nil after seconds. Queue#pop(timeout:) needs Ruby 3.2, and on
# 3.1 the keyword is taken as non_block, so pop raises at once on an empty queue.
def pop_with_timeout(queue, seconds)
  Timeout.timeout(seconds) { queue.pop }
rescue Timeout::Error
  nil
end

# A minimal HTTP server for the snapshot and stream endpoints, so no Flipt instance is needed.
class FakeFlipt
  attr_accessor :body
  attr_reader :stream_docs

  def initialize
    @body = EMPTY
    # Documents the stream endpoint sends, one per line, as the test pushes them.
    @stream_docs = Queue.new
    @server = TCPServer.new('127.0.0.1', 0)
    @thread = Thread.new { accept_loop }
  end

  def url
    "http://127.0.0.1:#{@server.addr[1]}"
  end

  def stop
    @stream_docs << END_STREAM
    @thread.kill
    @server.close
  end

  private

  def accept_loop
    loop do
      socket = @server.accept
      Thread.new(socket) { |s| handle(s) }
    end
  rescue IOError, Errno::EBADF
    # The server was closed.
  end

  def handle(socket)
    path = socket.gets.to_s.split[1].to_s
    # Discard headers.
    while (line = socket.gets) && line != "\r\n"; end

    if path.start_with?('/internal/v1/evaluation/snapshot/namespace/default')
      data = @body
      socket.write("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\n" \
                   "Content-Length: #{data.bytesize}\r\nConnection: close\r\n\r\n#{data}")
    elsif path.start_with?('/client/v2/environments/default/namespaces/default/stream')
      socket.write("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nTransfer-Encoding: chunked\r\n\r\n")
      loop do
        doc = pop_with_timeout(@stream_docs, 30)
        break if doc.nil? || doc == END_STREAM

        chunk = "{\"result\":#{doc}}\n"
        socket.write("#{chunk.bytesize.to_s(16)}\r\n#{chunk}\r\n")
        socket.flush
      end
      socket.write("0\r\n\r\n")
    else
      socket.write("HTTP/1.1 404 Not Found\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
    end
  rescue IOError, SystemCallError
    # The client went away or the server is stopping.
  ensure
    socket.close
  end
end

RSpec.describe 'Flipt::Client state update listeners' do
  before { @flipt = FakeFlipt.new }
  after { @flipt.stop }

  def new_client(**opts)
    Flipt::Client.new(url: @flipt.url, update_interval: 1, error_strategy: :fallback, **opts)
  end

  # Waits up to 5s for the queue to receive a value.
  def await(queue)
    Timeout.timeout(5) { queue.pop }
  end

  it 'fires on change and can evaluate from the listener' do
    calls = 0
    evaluated = Queue.new
    client = nil
    client = new_client(on_state_update: lambda {
      calls += 1
      # Evaluating from inside the listener must not deadlock.
      evaluated << client.evaluate_boolean(flag_key: 'new_flag', entity_id: 'entity').enabled
    })
    begin
      # Unchanged polls must not fire the listener.
      sleep 1.5
      expect(calls).to eq(0)

      @flipt.body = WITH_FLAG
      expect(await(evaluated)).to eq(true)
    ensure
      client.close
    end
  end

  it 'fires on a streamed change' do
    fired = Queue.new
    @flipt.stream_docs << EMPTY
    client = new_client(fetch_mode: :streaming, on_state_update: -> { fired << true })
    begin
      @flipt.stream_docs << WITH_FLAG
      expect(await(fired)).to eq(true)
    ensure
      client.close
    end
  end

  it 'waits in close for a running listener' do
    started = Queue.new
    release = Queue.new
    finished = false
    client = new_client(on_state_update: lambda {
      started << true
      pop_with_timeout(release, 5)
      finished = true
    })
    begin
      @flipt.body = WITH_FLAG
      await(started)

      closer = Thread.new { client.close }
      sleep 0.3
      # close must still be waiting for the listener.
      expect(closer.alive?).to eq(true)
      release << true
      expect(closer.join(5)).not_to be_nil
      expect(finished).to eq(true)
    ensure
      release << true
      client.close
    end
  end

  it 'keeps calling other listeners when one raises' do
    fired = Queue.new
    client = new_client(on_state_update: -> { raise 'boom' })
    begin
      client.add_state_update_listener { fired << true }
      # Change state inside the block, so the listener's warning is captured.
      expect do
        @flipt.body = WITH_FLAG
        expect(await(fired)).to eq(true)
      end.to output(/boom/).to_stderr
      expect(client.evaluate_boolean(flag_key: 'new_flag', entity_id: 'entity').enabled).to eq(true)
    ensure
      client.close
    end
  end

  it 'can close the client from inside a listener' do
    done = Queue.new
    later_calls = 0
    client = nil
    client = new_client(on_state_update: lambda {
      client.close
      done << true
    })
    # Registered after the closing listener, so it must never run.
    client.add_state_update_listener { later_calls += 1 }
    @flipt.body = WITH_FLAG
    await(done)
    sleep 0.3
    expect(later_calls).to eq(0)
    expect { client.evaluate_boolean(flag_key: 'new_flag', entity_id: 'entity') }
      .to raise_error(Flipt::Error, 'client is closed')
  end

  it 'keeps notifying when a listener raises a non-StandardError' do
    fired = Queue.new
    client = new_client(on_state_update: -> { raise NotImplementedError, 'nope' })
    begin
      client.add_state_update_listener { fired << true }
      expect do
        @flipt.body = WITH_FLAG
        await(fired)
        @flipt.body = BARRIER
        # The thread survived the first error, so it delivers the second change too.
        expect(await(fired)).to eq(true)
      end.to output(/NotImplementedError/).to_stderr
    ensure
      client.close
    end
  end

  it 'raises from every method after close' do
    client = new_client
    client.close
    expect { client.evaluate_variant(flag_key: 'f', entity_id: 'e') }.to raise_error(Flipt::Error, 'client is closed')
    expect { client.evaluate_batch(requests: []) }.to raise_error(Flipt::Error, 'client is closed')
    expect { client.list_flags }.to raise_error(Flipt::Error, 'client is closed')
    expect { client.snapshot }.to raise_error(Flipt::Error, 'client is closed')
  end

  it 'adds and removes listeners while running' do
    fired = Queue.new
    removed_calls = 0
    client = new_client
    begin
      sub = client.add_state_update_listener { removed_calls += 1 }
      sub.close
      sub.close # closing twice is a no-op
      client.add_state_update_listener { fired << true }
      @flipt.body = WITH_FLAG
      await(fired)
      expect(removed_calls).to eq(0)
    ensure
      client.close
    end
  end

  it 'can add a listener from inside a listener' do
    added = Queue.new
    added_fired = Queue.new
    client = nil
    once = false
    client = new_client(on_state_update: lambda {
      next if once

      once = true
      client.add_state_update_listener { added_fired << true }
      added << true
    })
    begin
      @flipt.body = WITH_FLAG
      await(added)
      @flipt.body = BARRIER
      expect(await(added_fired)).to eq(true)
    ensure
      client.close
    end
  end

  it 'raises when adding a listener after close' do
    # Start the update thread first, so the closed check is what raises.
    client = new_client(on_state_update: -> {})
    client.close
    expect { client.add_state_update_listener { nil } }.to raise_error(Flipt::Error, 'client is closed')
  end

  it 'rejects a listener that is not callable' do
    client = new_client
    begin
      expect { client.add_state_update_listener('nope') }.to raise_error(Flipt::ValidationError)
    ensure
      client.close
    end
  end

  it 'stops calling listeners after close' do
    calls = 0
    client = new_client(on_state_update: -> { calls += 1 })
    client.close
    client.close # closing twice is a no-op
    @flipt.body = WITH_FLAG
    sleep 1.5
    expect(calls).to eq(0)
    # close joined the update thread, so it is gone, not just idle.
    expect(Thread.list.map(&:name)).not_to include('flipt-state-update')
  end

  it 'does not send on_state_update to the engine' do
    # initialize_engine would reject or ignore it; make sure it never reaches the JSON.
    expect(Flipt::Client).to receive(:initialize_engine)
      .with(satisfy { |json| !json.include?('on_state_update') })
      .and_call_original
    new_client(on_state_update: -> {}).close
  end
end
