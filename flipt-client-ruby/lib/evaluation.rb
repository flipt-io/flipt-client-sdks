require "client/version"
require "ffi"
require "json"

module Flipt
  class Error < StandardError; end

  class EvaluationClient
    FLIPTENGINE="ext/libfliptengine"

    def self.platform_specific_lib
      case RbConfig::CONFIG['host_os']
      when /darwin|mac os/
        "#{FLIPTENGINE}.dylib"
      when /linux/
        "#{FLIPTENGINE}.so"
      when /mswin|msys|mingw|cygwin|bccwin|wince|emc/
        "#{FLIPTENGINE}.dll"
      else
        raise "unsupported platform #{RbConfig::CONFIG['host_os']}"
      end
    end

    extend FFI::Library
    ffi_lib File.expand_path(platform_specific_lib, __dir__)

    # void *initialize_engine(const char *const *namespaces);
    attach_function :initialize_engine, [:pointer], :pointer
    # void *initialize_engine_with_state(const char *const *namespaces, const char *state);
    attach_function :initialize_engine_with_state, [:pointer, :string], :pointer
    # void destroy_engine(void *engine_ptr);
    attach_function :destroy_engine, [:pointer], :void
    # const char *variant(void *engine_ptr, const char *evaluation_request);
    attach_function :variant, [:pointer, :string], :string
    # const char *boolean(void *engine_ptr, const char *evaluation_request);
    attach_function :boolean, [:pointer, :string], :string

    def initialize(namespace = "default", opts = {})
      @namespace = namespace
      namespace_list = [namespace]
      ns = FFI::MemoryPointer.new(:pointer, namespace_list.size)
      namespace_list.each_with_index do |namespace, i|
        ns[i].put_pointer(0, FFI::MemoryPointer.from_string(namespace))
      end

      if opts[:state].nil?
        @engine = self.class.initialize_engine(ns)
      else
        @engine = self.class.initialize_engine_with_state(ns, opts[:state])
      end

      ObjectSpace.define_finalizer(self, self.class.finalize(@engine))
    end

    def self.finalize(engine)
      proc { self.destroy_engine(engine) }
    end

    def variant(evaluation_request = {})
      evaluation_request[:namespace_key] = @namespace
      validate_evaluation_request(evaluation_request)
      resp = self.class.variant(@engine, evaluation_request.to_json)
      JSON.parse(resp)
    end

    def boolean(evaluation_request = {})
      evaluation_request[:namespace_key] = @namespace
      validate_evaluation_request(evaluation_request)
      resp = self.class.boolean(@engine, evaluation_request.to_json)
      JSON.parse(resp)
    end

    private 
    def validate_evaluation_request(evaluation_request)
      if evaluation_request[:entity_id].nil? || evaluation_request[:entity_id].empty?
        raise ArgumentError, "entity_id is required"
      elsif evaluation_request[:namespace_key].nil? || evaluation_request[:namespace_key].empty?
        raise ArgumentError, "namespace_key is required"
      elsif evaluation_request[:flag_key].nil? || evaluation_request[:flag_key].empty?
        raise ArgumentError, "flag_key is required"
      end
    end
  end
end
