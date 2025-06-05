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
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import lombok.Builder;
import lombok.Builder.Default;
import lombok.Nullable;

/**
 * A client for interacting with Flipt feature flags.
 *
 * <p>This client is thread-safe and should be reused across your application. Use the builder to
 * create a new instance:
 *
 * <pre>{@code
 * var client = FliptClient.builder()
 *     .url("http://localhost:8080")
 *     .build();
 * }</pre>
 *
 * @see FliptClientBuilder
 * @since 1.0.0
 */
public class FliptClient implements AutoCloseable {
  private static final String STATUS_SUCCESS = "success";
  private final Pointer engine;

  private final ObjectMapper objectMapper;

  public interface CLibrary extends Library {
    CLibrary INSTANCE = Native.load("fliptengine", CLibrary.class);

    Pointer initialize_engine(String opts);

    Pointer evaluate_boolean(Pointer engine, String evaluation_request);

    Pointer evaluate_variant(Pointer engine, String evaluation_request);

    Pointer evaluate_batch(Pointer engine, String batch_evaluation_request);

    Pointer list_flags(Pointer engine);

    Pointer get_snapshot(Pointer engine);

    void destroy_engine(Pointer engine);

    void destroy_string(Pointer str);
  }

  private FliptClient(ClientOptions clientOptions) {
    // Validate client options
    String environment = clientOptions.getEnvironment().orElse("default");
    if (environment.isEmpty()) {
      throw new FliptException.ConfigurationException("environment must not be empty");
    }
    String namespace = clientOptions.getNamespace().orElse("default");
    if (namespace.isEmpty()) {
      throw new FliptException.ConfigurationException("namespace must not be empty");
    }
    clientOptions
        .getRequestTimeout()
        .ifPresent(
            timeout -> {
              if (timeout < 0) {
                throw new FliptException.ConfigurationException(
                    "requestTimeout must not be negative");
              }
            });
    clientOptions
        .getUpdateInterval()
        .ifPresent(
            interval -> {
              if (interval < 0) {
                throw new FliptException.ConfigurationException(
                    "updateInterval must not be negative");
              }
            });

    ObjectMapper objectMapper = new ObjectMapper();
    objectMapper.registerModule(new Jdk8Module());

    String clientOptionsSerialized;
    try {
      clientOptionsSerialized = objectMapper.writeValueAsString(clientOptions);
    } catch (JsonProcessingException e) {
      throw new FliptException.EvaluationException(e);
    }

    Pointer engine = CLibrary.INSTANCE.initialize_engine(clientOptionsSerialized);

    this.objectMapper = objectMapper;
    this.engine = engine;
  }

  @Builder
  public static final class FliptClientBuilder {
    @Default private String environment = "default";
    @Default private String namespace = "default";
    @Default private String url = "http://localhost:8080";
    @Nullable private AuthenticationStrategy authentication;
    @Nullable private String reference;
    @Nullable private Duration requestTimeout;
    @Nullable private Duration updateInterval;
    @Default private FetchMode fetchMode = FetchMode.POLLING;
    @Default private ErrorStrategy errorStrategy = ErrorStrategy.FAIL;
    @Nullable private String snapshot;

    public FliptClient build() {
      return new FliptClient(
          new ClientOptions(
              Optional.of(environment),
              Optional.of(namespace),
              Optional.of(url),
              Optional.ofNullable(requestTimeout),
              Optional.ofNullable(updateInterval),
              Optional.ofNullable(authentication),
              Optional.ofNullable(reference),
              Optional.ofNullable(fetchMode),
              Optional.ofNullable(errorStrategy),
              Optional.ofNullable(snapshot)));
    }
  }

  private static class InternalEvaluationRequest {

    private final String flagKey;

    private final String entityId;

    private final Map<String, String> context;

    public InternalEvaluationRequest(String flagKey, String entityId, Map<String, String> context) {
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

  /**
   * evaluateVariant evaluates a variant flag.
   *
   * @param flagKey the key for the flag to evaluate
   * @param entityId the ID for the entity to evaluate
   * @param context the context for the evaluation
   * @return the evaluation response
   */
  public VariantEvaluationResponse evaluateVariant(
      String flagKey, String entityId, Map<String, String> context) {
    InternalEvaluationRequest evaluationRequest =
        new InternalEvaluationRequest(flagKey, entityId, context);

    String evaluationRequestSerialized;
    try {
      evaluationRequestSerialized = this.objectMapper.writeValueAsString(evaluationRequest);
    } catch (JsonProcessingException e) {
      throw new FliptException.EvaluationException(e);
    }

    Pointer value = CLibrary.INSTANCE.evaluate_variant(this.engine, evaluationRequestSerialized);

    TypeReference<Result<VariantEvaluationResponse>> typeRef =
        new TypeReference<Result<VariantEvaluationResponse>>() {};

    Result<VariantEvaluationResponse> resp = this.readValue(value, typeRef);
    if (!FliptClient.STATUS_SUCCESS.equals(resp.getStatus())) {
      throw new FliptException.EvaluationException(resp.getErrorMessage().get());
    }
    return resp.getResult().get();
  }

  /**
   * evaluateBoolean evaluates a boolean flag.
   *
   * @param flagKey the key for the flag to evaluate
   * @param entityId the ID for the entity to evaluate
   * @param context the context for the evaluation
   * @return the evaluation response
   */
  public BooleanEvaluationResponse evaluateBoolean(
      String flagKey, String entityId, Map<String, String> context) {
    InternalEvaluationRequest evaluationRequest =
        new InternalEvaluationRequest(flagKey, entityId, context);

    String evaluationRequestSerialized;
    try {
      evaluationRequestSerialized = this.objectMapper.writeValueAsString(evaluationRequest);
    } catch (JsonProcessingException e) {
      throw new FliptException.EvaluationException(e);
    }
    Pointer value = CLibrary.INSTANCE.evaluate_boolean(this.engine, evaluationRequestSerialized);

    TypeReference<Result<BooleanEvaluationResponse>> typeRef =
        new TypeReference<Result<BooleanEvaluationResponse>>() {};
    Result<BooleanEvaluationResponse> resp = this.readValue(value, typeRef);

    if (!FliptClient.STATUS_SUCCESS.equals(resp.getStatus())) {
      throw new FliptException.EvaluationException(resp.getErrorMessage().get());
    }
    return resp.getResult().get();
  }

  /**
   * evaluateBatch evaluates a batch of flags.
   *
   * @param batchEvaluationRequests the batch of flags to evaluate
   * @return the evaluation response
   */
  public BatchEvaluationResponse evaluateBatch(List<EvaluationRequest> batchEvaluationRequests) {
    List<InternalEvaluationRequest> evaluationRequests =
        new ArrayList<>(batchEvaluationRequests.size());

    for (EvaluationRequest req : batchEvaluationRequests) {
      evaluationRequests.add(
          new InternalEvaluationRequest(req.getFlagKey(), req.getEntityId(), req.getContext()));
    }

    String batchEvaluationRequestSerialized;
    try {
      batchEvaluationRequestSerialized = this.objectMapper.writeValueAsString(evaluationRequests);
    } catch (JsonProcessingException e) {
      throw new FliptException.EvaluationException(e);
    }
    Pointer value = CLibrary.INSTANCE.evaluate_batch(this.engine, batchEvaluationRequestSerialized);

    TypeReference<Result<BatchEvaluationResponse>> typeRef =
        new TypeReference<Result<BatchEvaluationResponse>>() {};

    Result<BatchEvaluationResponse> resp = this.readValue(value, typeRef);
    if (!FliptClient.STATUS_SUCCESS.equals(resp.getStatus())) {
      throw new FliptException.EvaluationException(resp.getErrorMessage().get());
    }
    return resp.getResult().get();
  }

  /**
   * listFlags lists all flags in the namespace.
   *
   * @return the list of flags
   */
  public ArrayList<Flag> listFlags() {
    Pointer value = CLibrary.INSTANCE.list_flags(this.engine);

    TypeReference<Result<ArrayList<Flag>>> typeRef =
        new TypeReference<Result<ArrayList<Flag>>>() {};

    Result<ArrayList<Flag>> resp = this.readValue(value, typeRef);
    if (!FliptClient.STATUS_SUCCESS.equals(resp.getStatus())) {
      throw new FliptException.EvaluationException(resp.getErrorMessage().get());
    }
    return resp.getResult().get();
  }

  /**
   * getSnapshot gets the snapshot from the Flipt client.
   *
   * @return the snapshot
   */
  public String getSnapshot() {
    Pointer value = CLibrary.INSTANCE.get_snapshot(this.engine);
    try {
      return value.getString(0, "UTF-8");
    } finally {
      CLibrary.INSTANCE.destroy_string(value);
    }
  }

  @Override
  public void close() {
    if (this.engine != null) {
      CLibrary.INSTANCE.destroy_engine(this.engine);
    }
  }

  private <T> T readValue(Pointer ptr, TypeReference<T> typeRef) {
    try {
      String value = ptr.getString(0, "UTF-8");
      return this.objectMapper.readValue(value, typeRef);
    } catch (JsonProcessingException e) {
      throw new FliptException.EvaluationException(e);
    } finally {
      CLibrary.INSTANCE.destroy_string(ptr);
    }
  }
}
