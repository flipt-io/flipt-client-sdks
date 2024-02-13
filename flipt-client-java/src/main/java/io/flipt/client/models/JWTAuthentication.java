package io.flipt.client.models;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonProperty;

@JsonInclude(JsonInclude.Include.NON_EMPTY)
public class JWTAuthentication implements AuthenticationStrategy {
  private final String jwtToken;

  public JWTAuthentication(String jwtToken) {
    this.jwtToken = jwtToken;
  }

  @JsonProperty("jwt_token")
  public String getJwtToken() {
    return jwtToken;
  }
}
