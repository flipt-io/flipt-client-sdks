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

  # TlsConfig provides configuration for TLS connections to Flipt servers
  class TlsConfig
    attr_reader :ca_cert_file, :ca_cert_data, :insecure_skip_verify,
                :client_cert_file, :client_key_file, :client_cert_data, :client_key_data

    # Initialize TLS configuration
    #
    # @param ca_cert_file [String, nil] Path to CA certificate file (PEM format)
    # @param ca_cert_data [String, nil] Raw CA certificate content (PEM format)
    # @param insecure_skip_verify [Boolean, nil] Skip certificate verification (development only)
    # @param client_cert_file [String, nil] Path to client certificate file (PEM format)
    # @param client_key_file [String, nil] Path to client key file (PEM format)
    # @param client_cert_data [String, nil] Raw client certificate content (PEM format)
    # @param client_key_data [String, nil] Raw client key content (PEM format)
    def initialize(ca_cert_file: nil, ca_cert_data: nil, insecure_skip_verify: nil,
                   client_cert_file: nil, client_key_file: nil, 
                   client_cert_data: nil, client_key_data: nil)
      @ca_cert_file = ca_cert_file
      @ca_cert_data = ca_cert_data
      @insecure_skip_verify = insecure_skip_verify
      @client_cert_file = client_cert_file
      @client_key_file = client_key_file
      @client_cert_data = client_cert_data
      @client_key_data = client_key_data

      validate_files!
    end

    # Create TLS config for insecure connections (development only)
    # WARNING: Only use this in development environments
    #
    # @return [TlsConfig] TLS config with certificate verification disabled
    def self.insecure
      new(insecure_skip_verify: true)
    end

    # Create TLS config with CA certificate file
    #
    # @param ca_cert_file [String] Path to CA certificate file
    # @return [TlsConfig] TLS config with custom CA certificate
    def self.with_ca_cert_file(ca_cert_file)
      new(ca_cert_file: ca_cert_file)
    end

    # Create TLS config with CA certificate data
    #
    # @param ca_cert_data [String] CA certificate content in PEM format
    # @return [TlsConfig] TLS config with custom CA certificate
    def self.with_ca_cert_data(ca_cert_data)
      new(ca_cert_data: ca_cert_data)
    end

    # Create TLS config for mutual TLS with certificate files
    #
    # @param client_cert_file [String] Path to client certificate file
    # @param client_key_file [String] Path to client key file
    # @return [TlsConfig] TLS config with mutual TLS
    def self.with_mutual_tls(client_cert_file, client_key_file)
      new(client_cert_file: client_cert_file, client_key_file: client_key_file)
    end

    # Create TLS config for mutual TLS with certificate data
    #
    # @param client_cert_data [String] Client certificate content in PEM format
    # @param client_key_data [String] Client key content in PEM format
    # @return [TlsConfig] TLS config with mutual TLS
    def self.with_mutual_tls_data(client_cert_data, client_key_data)
      new(client_cert_data: client_cert_data, client_key_data: client_key_data)
    end

    # Convert to hash for JSON serialization
    # @return [Hash] TLS configuration as hash
    def to_h
      hash = {}
      hash[:ca_cert_file] = @ca_cert_file if @ca_cert_file
      hash[:ca_cert_data] = @ca_cert_data if @ca_cert_data
      hash[:insecure_skip_verify] = @insecure_skip_verify unless @insecure_skip_verify.nil?
      hash[:client_cert_file] = @client_cert_file if @client_cert_file
      hash[:client_key_file] = @client_key_file if @client_key_file
      hash[:client_cert_data] = @client_cert_data if @client_cert_data
      hash[:client_key_data] = @client_key_data if @client_key_data
      hash
    end

    private

    def validate_files!
      validate_file_exists(@ca_cert_file, 'CA certificate file') if @ca_cert_file
      validate_file_exists(@client_cert_file, 'Client certificate file') if @client_cert_file
      validate_file_exists(@client_key_file, 'Client key file') if @client_key_file
    end

    def validate_file_exists(file_path, description)
      return if file_path.nil? || file_path.strip.empty?
      
      unless File.exist?(file_path)
        raise ValidationError, "#{description} does not exist: #{file_path}"
      end
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
