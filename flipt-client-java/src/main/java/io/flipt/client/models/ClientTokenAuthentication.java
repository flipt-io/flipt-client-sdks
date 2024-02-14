package io.flipt.client.models;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonProperty;

@JsonInclude(JsonInclude.Include.NON_EMPTY)
public class ClientTokenAuthentication implements AuthenticationStrategy {
  private final String clientToken;

  public ClientTokenAuthentication(String clientToken) {
    this.clientToken = clientToken;
  }

  @JsonProperty("client_token")
  public String getClientToken() {
    return clientToken;
  }
}
