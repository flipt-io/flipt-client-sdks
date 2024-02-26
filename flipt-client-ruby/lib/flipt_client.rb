# frozen_string_literal: true

require 'flipt_client/version'
require 'flipt_client/models'
require 'ffi'
require 'json'

module Flipt
  class Error < StandardError; end

  class EvaluationClient
    extend FFI::Library

    FLIPTENGINE = 'libfliptengine'

    def self.libfile
      case RbConfig::CONFIG['arch']
      when /arm64-darwin/
        "ext/darwin_arm64/#{FLIPTENGINE}.dylib"
      when /arm64-linux|aarch64-linux/
        "ext/linux_arm64/#{FLIPTENGINE}.so"
      when /x86_64-linux/
        "ext/linux_x86_64/#{FLIPTENGINE}.so"
      else
        raise "unsupported platform #{RbConfig::CONFIG['arch']}"
      end
    end

    ffi_lib File.expand_path(libfile, __dir__)

    # void *initialize_engine(const char *namespace);
    attach_function :initialize_engine, [:pointer], :pointer
    # void destroy_engine(void *engine_ptr);
    attach_function :destroy_engine, [:pointer], :void
    # const char *evaluate_variant(void *engine_ptr, const char *evaluation_request);
    attach_function :evaluate_variant, %i[pointer string], :string
    # const char *evaluate_boolean(void *engine_ptr, const char *evaluation_request);
    attach_function :evaluate_boolean, %i[pointer string], :string
    # const char *evaluate_batch(void *engine_ptr, const char *batch_evaluation_request);
    attach_function :evaluate_batch, %i[pointer string], :string
    # const char *list_flags(void *engine_ptr);
    attach_function :list_flags, [:pointer], :string

    # Create a new Flipt client
    #
    # @param namespace [String] namespace
    # @param opts [Hash] options
    # @option opts [String] :url Flipt server url
    # @option opts [AuthenticationStrategy] :authentication strategy to authenticate with Flipt
    # @option opts [Integer] :update_interval interval in seconds to update the cache
    # @option opts [String] :reference reference to use for namespace data
    def initialize(namespace = 'default', opts = {})
      @namespace = namespace

      # set default no auth if not provided
      authentication = opts.fetch(:authentication, Flipt::NoAuthentication.new)
      raise ArgumentError, "invalid authentication strategy" unless authentication && authentication.is_a?(Flipt::AuthenticationStrategy)

      opts[:authentication] = authentication.strategy

      ns = FFI::MemoryPointer.from_string(namespace)
      @engine = self.class.initialize_engine(ns, opts.to_json)
      ObjectSpace.define_finalizer(self, self.class.finalize(@engine))
    end

    def self.finalize(engine)
      proc { destroy_engine(engine) }
    end

    # Evaluate a variant flag for a given request
    #
    # @param evaluation_request [Hash] evaluation request
    # @option evaluation_request [String] :entity_id entity id
    # @option evaluation_request [String] :flag_key flag key
    def evaluate_variant(evaluation_request = {})
      validate_evaluation_request(evaluation_request)
      resp = self.class.evaluate_variant(@engine, evaluation_request.to_json)
      JSON.parse(resp)
    end

    # Evaluate a boolean flag for a given request
    #
    # @param evaluation_request [Hash] evaluation request
    # @option evaluation_request [String] :entity_id entity id
    # @option evaluation_request [String] :flag_key flag key
    def evaluate_boolean(evaluation_request = {})
      validate_evaluation_request(evaluation_request)
      resp = self.class.evaluate_boolean(@engine, evaluation_request.to_json)
      JSON.parse(resp)
    end
  
    # Evaluate a batch of flags for a given request
    #
    # @param batch_evaluation_request [Array<Hash>] batch evaluation request
    #   - :entity_id [String] entity id
    #   - :flag_key [String] flag key
    def evaluate_batch(batch_evaluation_request = [])
      for request in batch_evaluation_request do
        validate_evaluation_request(request)
      end

      resp = self.class.evaluate_batch(@engine, batch_evaluation_request.to_json)
      JSON.parse(resp)
    end

    # List all flags in the namespace
    def list_flags
      resp = self.class.list_flags(@engine)
      JSON.parse(resp)
    end

    private
    def validate_evaluation_request(evaluation_request)
      if evaluation_request[:entity_id].nil? || evaluation_request[:entity_id].empty?
        raise ArgumentError, 'entity_id is required'
      elsif evaluation_request[:flag_key].nil? || evaluation_request[:flag_key].empty?
        raise ArgumentError, 'flag_key is required'
      end
    end
  end
end
