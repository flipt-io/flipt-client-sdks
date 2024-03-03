package io.flipt.client;

import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.datatype.jdk8.Jdk8Module;
import com.sun.jna.Library;
import com.sun.jna.Native;
import com.sun.jna.Pointer;
import io.flipt.client.models.*;
import java.time.Duration;
import java.util.ArrayList;
import java.util.Map;
import java.util.Optional;

public class FliptEvaluationClient {
  private final Pointer engine;

  private final ObjectMapper objectMapper;

  public interface CLibrary extends Library {
    CLibrary INSTANCE = (CLibrary) Native.load("fliptengine", CLibrary.class);

    Pointer initialize_engine(String namespace, String opts);

    Pointer evaluate_boolean(Pointer engine, String evaluation_request);

    Pointer evaluate_variant(Pointer engine, String evaluation_request);

    Pointer evaluate_batch(Pointer engine, String batch_evaluation_request);

    Pointer list_flags(Pointer engine);

    void destroy_engine(Pointer engine);

    void destroy_string(Pointer str);
  }

  private FliptEvaluationClient(String namespace, EngineOpts engineOpts)
      throws JsonProcessingException {

    ObjectMapper objectMapper = new ObjectMapper();
    objectMapper.registerModule(new Jdk8Module());

    String engineOptions = objectMapper.writeValueAsString(engineOpts);

    Pointer engine = CLibrary.INSTANCE.initialize_engine(namespace, engineOptions);

    this.objectMapper = objectMapper;
    this.engine = engine;
  }

  public static FliptEvaluationClientBuilder builder() {
    return new FliptEvaluationClientBuilder();
  }

  public static final class FliptEvaluationClientBuilder {
    private String namespace = "default";
    private String url = "http://localhost:8080";
    private AuthenticationStrategy authentication;
    private String reference;
    private Duration updateInterval;

    public FliptEvaluationClientBuilder() {}

    public FliptEvaluationClientBuilder url(String url) {
      this.url = url;
      return this;
    }

    public FliptEvaluationClientBuilder namespace(String namespace) {
      this.namespace = namespace;
      return this;
    }

    public FliptEvaluationClientBuilder authentication(AuthenticationStrategy authentication) {
      this.authentication = authentication;
      return this;
    }

    public FliptEvaluationClientBuilder updateInterval(Duration updateInterval) {
      this.updateInterval = updateInterval;
      return this;
    }

    public FliptEvaluationClientBuilder reference(String reference) {
      this.reference = reference;
      return this;
    }

    public FliptEvaluationClient build() throws JsonProcessingException {
      return new FliptEvaluationClient(
          namespace,
          new EngineOpts(
              Optional.of(url),
              Optional.ofNullable(updateInterval),
              Optional.ofNullable(authentication),
              Optional.ofNullable(reference)));
    }
  }

  private static class InternalEvaluationRequest {

    private final String flagKey;

    private final String entityId;

    private final Map<String, String> context;

    public InternalEvaluationRequest(String flagKey, String entityId, Map<String, String> context) {
      this.flagKey = flagKey;
      this.entityId = entityId;
      this.context = context;
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

  public Result<VariantEvaluationResponse> evaluateVariant(
      String flagKey, String entityId, Map<String, String> context) throws JsonProcessingException {
    InternalEvaluationRequest evaluationRequest =
        new InternalEvaluationRequest(flagKey, entityId, context);

    String evaluationRequestSerialized = this.objectMapper.writeValueAsString(evaluationRequest);
    Pointer value = CLibrary.INSTANCE.evaluate_variant(this.engine, evaluationRequestSerialized);

    TypeReference<Result<VariantEvaluationResponse>> typeRef =
        new TypeReference<Result<VariantEvaluationResponse>>() {};

    return this.readValue(value, typeRef);
  }

  public Result<BooleanEvaluationResponse> evaluateBoolean(
      String flagKey, String entityId, Map<String, String> context) throws JsonProcessingException {
    InternalEvaluationRequest evaluationRequest =
        new InternalEvaluationRequest(flagKey, entityId, context);

    String evaluationRequestSerialized = this.objectMapper.writeValueAsString(evaluationRequest);
    Pointer value = CLibrary.INSTANCE.evaluate_boolean(this.engine, evaluationRequestSerialized);

    TypeReference<Result<BooleanEvaluationResponse>> typeRef =
        new TypeReference<Result<BooleanEvaluationResponse>>() {};
    return this.readValue(value, typeRef);
  }

  public Result<BatchEvaluationResponse> evaluateBatch(EvaluationRequest[] batchEvaluationRequests)
      throws JsonProcessingException {
    ArrayList<InternalEvaluationRequest> evaluationRequests =
        new ArrayList<>(batchEvaluationRequests.length);

    for (int i = 0; i < batchEvaluationRequests.length; i++) {
      evaluationRequests.add(
          i,
          new InternalEvaluationRequest(
              batchEvaluationRequests[i].getFlagKey(),
              batchEvaluationRequests[i].getEntityId(),
              batchEvaluationRequests[i].getContext()));
    }

    String batchEvaluationRequestSerialized =
        this.objectMapper.writeValueAsString(evaluationRequests.toArray());
    Pointer value = CLibrary.INSTANCE.evaluate_batch(this.engine, batchEvaluationRequestSerialized);

    TypeReference<Result<BatchEvaluationResponse>> typeRef =
        new TypeReference<Result<BatchEvaluationResponse>>() {};

    return this.readValue(value, typeRef);
  }

  public Result<ArrayList<Flag>> listFlags() throws JsonProcessingException {
    Pointer value = CLibrary.INSTANCE.list_flags(this.engine);

    TypeReference<Result<ArrayList<Flag>>> typeRef =
        new TypeReference<Result<ArrayList<Flag>>>() {};

    return this.readValue(value, typeRef);
  }

  public void close() {
    CLibrary.INSTANCE.destroy_engine(this.engine);
  }

  private <T> T readValue(Pointer ptr, TypeReference<T> typeRef) throws JsonProcessingException {
    try {
      String value = ptr.getString(0, "UTF-8");
      return this.objectMapper.readValue(value, typeRef);
    } finally {
      CLibrary.INSTANCE.destroy_string(ptr);
    }
  }
}
