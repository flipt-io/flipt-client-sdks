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

  private final String namespace;

  private final ObjectMapper objectMapper;

  public interface CLibrary extends Library {
    CLibrary INSTANCE = (CLibrary) Native.load("fliptengine", CLibrary.class);

    Pointer initialize_engine(String[] namespaces, String opts);

    String evaluate_boolean(Pointer engine, String evaluation_request);

    String evaluate_variant(Pointer engine, String evaluation_request);

    String evaluate_batch(Pointer engine, String batch_evaluation_request);

    void destroy_engine(Pointer engine);
  }

  private FliptEvaluationClient(String namespace, EngineOpts engineOpts)
      throws JsonProcessingException {

    String[] namespaces = {namespace};

    ObjectMapper objectMapper = new ObjectMapper();
    objectMapper.registerModule(new Jdk8Module());

    String engineOptions = objectMapper.writeValueAsString(engineOpts);

    Pointer engine = CLibrary.INSTANCE.initialize_engine(namespaces, engineOptions);

    this.objectMapper = objectMapper;
    this.engine = engine;
    this.namespace = namespace;
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

  private static class EvalRequest {
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

  public Result<VariantEvaluationResponse> evaluateVariant(
      String flagKey, String entityId, Map<String, String> context) throws JsonProcessingException {
    EvalRequest evaluationRequest = new EvalRequest(this.namespace, flagKey, entityId, context);

    String evaluationRequestSerialized = this.objectMapper.writeValueAsString(evaluationRequest);
    String value = CLibrary.INSTANCE.evaluate_variant(this.engine, evaluationRequestSerialized);

    TypeReference<Result<VariantEvaluationResponse>> typeRef =
        new TypeReference<Result<VariantEvaluationResponse>>() {};

    return this.objectMapper.readValue(value, typeRef);
  }

  public Result<BooleanEvaluationResponse> evaluateBoolean(
      String flagKey, String entityId, Map<String, String> context) throws JsonProcessingException {
    EvalRequest evaluationRequest = new EvalRequest(this.namespace, flagKey, entityId, context);

    String evaluationRequestSerialized = this.objectMapper.writeValueAsString(evaluationRequest);
    String value = CLibrary.INSTANCE.evaluate_boolean(this.engine, evaluationRequestSerialized);

    TypeReference<Result<BooleanEvaluationResponse>> typeRef =
        new TypeReference<Result<BooleanEvaluationResponse>>() {};
    return this.objectMapper.readValue(value, typeRef);
  }

  public Result<BatchEvaluationResponse> evaluateBatch(EvaluationRequest[] batchEvaluationRequests)
      throws JsonProcessingException {
    ArrayList<EvalRequest> evaluationRequests = new ArrayList<>(batchEvaluationRequests.length);

    for (int i = 0; i < batchEvaluationRequests.length; i++) {
      evaluationRequests.add(
          i,
          new EvalRequest(
              this.namespace,
              batchEvaluationRequests[i].getFlagKey(),
              batchEvaluationRequests[i].getEntityId(),
              batchEvaluationRequests[i].getContext()));
    }

    String batchEvaluationRequestSerialized =
        this.objectMapper.writeValueAsString(evaluationRequests.toArray());
    String value = CLibrary.INSTANCE.evaluate_batch(this.engine, batchEvaluationRequestSerialized);

    TypeReference<Result<BatchEvaluationResponse>> typeRef =
        new TypeReference<Result<BatchEvaluationResponse>>() {};

    return this.objectMapper.readValue(value, typeRef);
  }

  public void shutdown() {
    CLibrary.INSTANCE.destroy_engine(this.engine);
  }
}
