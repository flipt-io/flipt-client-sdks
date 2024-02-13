package io.flipt.client.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.util.Map;

public class EvalRequest {
  private final String namespaceKey;

  private final String flagKey;

  private final String entityId;

  private final Map<String, String> context;

  public EvalRequest(
      String namespaceKey, String flagKey, String entityId, Map<String, String> context) {
    this.namespaceKey = namespaceKey;
    this.flagKey = flagKey;
    this.entityId = entityId;
    this.context = context;
  }

  @JsonProperty("namespace_key")
  public String getNamespaceKey() {
    return namespaceKey;
  }

  @JsonProperty("flag_key")
  public String getFlagKey() {
    return flagKey;
  }

  @JsonProperty("entity_id")
  public String getEntityId() {
    return entityId;
  }

  @JsonProperty("context")
  public Map<String, String> getContext() {
    return context;
  }
}
