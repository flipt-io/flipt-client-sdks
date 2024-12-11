package io.flipt.client.models;

import com.fasterxml.jackson.annotation.JsonInclude;
import java.util.Map;

@JsonInclude(JsonInclude.Include.NON_EMPTY)
public class EvaluationRequest {
  private final String flagKey;

  private final String entityId;

  private final Map<String, String> context;

  public EvaluationRequest(String flagKey, String entityId, Map<String, String> context) {
    if (flagKey == null || flagKey.isEmpty()) {
      throw new IllegalArgumentException("flagKey is required");
    }
    if (entityId == null || entityId.isEmpty()) {
      throw new IllegalArgumentException("entityId is required");
    }
    if (context == null) {
      context = new HashMap<>();
    }
    this.flagKey = flagKey;
    this.entityId = entityId;
    this.context = context;
  }

  public String getFlagKey() {
    return flagKey;
  }

  public String getEntityId() {
    return entityId;
  }

  public Map<String, String> getContext() {
    return context;
  }
}
