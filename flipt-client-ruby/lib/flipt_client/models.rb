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
      super
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
      super
      @token = token
    end

    def strategy
      {
        jwt_token: @token
      }
    end
  end
end
