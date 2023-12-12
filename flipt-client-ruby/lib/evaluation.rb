# frozen_string_literal: true

require 'client/version'
require 'ffi'
require 'json'

module Flipt
  class Error < StandardError; end

  class EvaluationClient
    extend FFI::Library

    FLIPTENGINE = 'ext/libfliptengine'

    def self.libfile
      case RbConfig::CONFIG['host_os']
      when /mswin|msys|mingw|cygwin|bccwin|wince|emc/
        "#{FLIPTENGINE}.dll"
      when /darwin|mac os/
        "#{FLIPTENGINE}.dylib"
      when /linux/
        "#{FLIPTENGINE}.so"
      else
        raise "unsupported platform #{RbConfig::CONFIG['host_os']}"
      end
    end

    ffi_lib File.expand_path(libfile, __dir__)

    # void *initialize_engine(const char *const *namespaces, const char *opts);
    attach_function :initialize_engine, %i[pointer string], :pointer
    # void destroy_engine(void *engine_ptr);
    attach_function :destroy_engine, [:pointer], :void
    # const char *evaluate_variant(void *engine_ptr, const char *evaluation_request);
    attach_function :evaluate_variant, %i[pointer string], :string
    # const char *evaluate_boolean(void *engine_ptr, const char *evaluation_request);
    attach_function :evaluate_boolean, %i[pointer string], :string

    def initialize(namespace = 'default', opts = {})
      @namespace = namespace
      namespace_list = [namespace]
      ns = FFI::MemoryPointer.new(:pointer, namespace_list.size)
      namespace_list.each_with_index do |namespace, i|
        ns[i].put_pointer(0, FFI::MemoryPointer.from_string(namespace))
      end

      @engine = self.class.initialize_engine(ns, opts.to_json)
      ObjectSpace.define_finalizer(self, self.class.finalize(@engine))
    end

    def self.finalize(engine)
      proc { destroy_engine(engine) }
    end

    def evaluate_variant(evaluation_request = {})
      evaluation_request[:namespace_key] = @namespace
      validate_evaluation_request(evaluation_request)
      resp = self.class.evaluate_variant(@engine, evaluation_request.to_json)
      JSON.parse(resp)
    end

    def evaluate_boolean(evaluation_request = {})
      evaluation_request[:namespace_key] = @namespace
      validate_evaluation_request(evaluation_request)
      resp = self.class.evaluate_boolean(@engine, evaluation_request.to_json)
      JSON.parse(resp)
    end

    private
    def validate_evaluation_request(evaluation_request)
      if evaluation_request[:entity_id].nil? || evaluation_request[:entity_id].empty?
        raise ArgumentError, 'entity_id is required'
      elsif evaluation_request[:namespace_key].nil? || evaluation_request[:namespace_key].empty?
        raise ArgumentError, 'namespace_key is required'
      elsif evaluation_request[:flag_key].nil? || evaluation_request[:flag_key].empty?
        raise ArgumentError, 'flag_key is required'
      end
    end
  end
end
