# frozen_string_literal: true

module Flipt
  # AuthenticationStrategy is a base class for different authentication strategies
  class AuthenticationStrategy
    def strategy
      raise NotImplementedError
    end
  end

  # NoAuthentication is a strategy that does not require authentication
  class NoAuthentication < AuthenticationStrategy
    def strategy
      nil
    end
  end

  # ClientTokenAuthentication is a strategy that uses a client token for authentication
  class ClientTokenAuthentication < AuthenticationStrategy
    def initialize(token)
      @token = token
    end

    def strategy
      {
        client_token: @token
      }
    end
  end

  # JWTAuthentication is a strategy that uses a JWT token for authentication
  class JWTAuthentication < AuthenticationStrategy
    def initialize(token)
      @token = token
    end

    def strategy
      {
        jwt_token: @token
      }
    end
  end

  # VariantEvaluationResponse
  # @attr_reader [String] flag_key
  # @attr_reader [Boolean] match
  # @attr_reader [String] reason
  # @attr_reader [String] variant_key
  # @attr_reader [String, nil] variant_attachment
  # @attr_reader [Array<String>] segment_keys
  class VariantEvaluationResponse
    attr_reader :flag_key, :match, :reason, :variant_key, :variant_attachment, :segment_keys
    def initialize(flag_key:, match:, reason:, variant_key:, variant_attachment: nil, segment_keys: [])
      @flag_key = flag_key
      @match = match
      @reason = reason
      @variant_key = variant_key
      @variant_attachment = variant_attachment
      @segment_keys = segment_keys
    end
  end

  # BooleanEvaluationResponse
  # @attr_reader [String] flag_key
  # @attr_reader [Boolean] enabled
  # @attr_reader [String] reason
  # @attr_reader [Array<String>] segment_keys
  class BooleanEvaluationResponse
    attr_reader :flag_key, :enabled, :reason, :segment_keys
    def initialize(flag_key:, enabled:, reason:, segment_keys: [])
      @flag_key = flag_key
      @enabled = enabled
      @reason = reason
      @segment_keys = segment_keys
    end
  end

  # ErrorEvaluationResponse
  # @attr_reader [String] flag_key
  # @attr_reader [String] namespace_key
  # @attr_reader [String] reason
  # @attr_reader [String] error_message
  class ErrorEvaluationResponse
    attr_reader :flag_key, :namespace_key, :reason, :error_message
    def initialize(flag_key:, namespace_key:, reason:, error_message:)
      @flag_key = flag_key
      @namespace_key = namespace_key
      @reason = reason
      @error_message = error_message
    end
  end

  # BatchEvaluationResponse
  # @attr_reader [Array] responses
  class BatchEvaluationResponse
    attr_reader :responses
    def initialize(responses: [])
      @responses = responses
    end
  end
end
