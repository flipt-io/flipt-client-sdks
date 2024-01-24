# frozen_string_literal: true

module Flipt
  class AuthenticationStrategy
    def strategy
        raise NotImplementedError
    end
  end

  class NoAuthentication < AuthenticationStrategy
    def strategy
      nil
    end
  end

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
end