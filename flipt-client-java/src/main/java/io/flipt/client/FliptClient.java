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
 *
 * // With TLS configuration
 * var tlsClient = FliptClient.builder()
 *     .url("https://flipt.example.com")
 *     .tlsConfig(TlsConfig.builder().caCertFile("/path/to/ca.pem").build())
 *     .build();
 * }</pre>
 *
 * @since 1.0.0
 */
public class FliptClient implements AutoCloseable {
  private static final String STATUS_SUCCESS = "success";

  private String environment;
  private String namespace;
  private String url;
  private AuthenticationStrategy authentication;
  private String reference;
  private Duration requestTimeout;
  private Duration updateInterval;
  private FetchMode fetchMode;
  private ErrorStrategy errorStrategy;
  private String snapshot;
  private TlsConfig tlsConfig;

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

  @Builder
  private FliptClient(
      String environment,
      String namespace,
      String url,
      AuthenticationStrategy authentication,
      String reference,
      Duration requestTimeout,
      Duration updateInterval,
      FetchMode fetchMode,
      ErrorStrategy errorStrategy,
      String snapshot,
      TlsConfig tlsConfig) {
    this.environment = environment != null ? environment : "default";
    this.namespace = namespace != null ? namespace : "default";
    this.url = url != null ? url : "http://localhost:8080";
    this.authentication = authentication;
    this.reference = reference;
    this.requestTimeout = requestTimeout != null ? requestTimeout : Duration.ZERO;
    this.updateInterval = updateInterval != null ? updateInterval : Duration.ofSeconds(120);
    this.fetchMode = fetchMode != null ? fetchMode : FetchMode.POLLING;
    this.errorStrategy = errorStrategy != null ? errorStrategy : ErrorStrategy.FAIL;
    this.snapshot = snapshot;
    this.tlsConfig = tlsConfig;

    this.objectMapper = new ObjectMapper();
    this.objectMapper.registerModule(new Jdk8Module());

    ClientOptions clientOptions =
        new ClientOptions(
            Optional.ofNullable(this.environment),
            Optional.ofNullable(this.namespace),
            Optional.ofNullable(this.url),
            Optional.ofNullable(this.requestTimeout),
            Optional.ofNullable(this.updateInterval),
            Optional.ofNullable(this.authentication),
            Optional.ofNullable(this.reference),
            Optional.ofNullable(this.fetchMode),
            Optional.ofNullable(this.errorStrategy),
            Optional.ofNullable(this.snapshot),
            Optional.ofNullable(this.tlsConfig));

    String clientOptionsSerialized;

    try {
      clientOptionsSerialized = objectMapper.writeValueAsString(clientOptions);
    } catch (JsonProcessingException e) {
      throw new FliptException.EvaluationException(e);
    }

    this.engine = CLibrary.INSTANCE.initialize_engine(clientOptionsSerialized);
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
      throw new FliptException.EvaluationException(resp.getErrorMessage().orElse("Unknown error"));
    }
    return resp.getResult()
        .orElseThrow(
            () -> new FliptException.EvaluationException("No result returned from engine"));
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
      throw new FliptException.EvaluationException(resp.getErrorMessage().orElse("Unknown error"));
    }
    return resp.getResult()
        .orElseThrow(
            () -> new FliptException.EvaluationException("No result returned from engine"));
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
      throw new FliptException.EvaluationException(resp.getErrorMessage().orElse("Unknown error"));
    }
    return resp.getResult()
        .orElseThrow(
            () -> new FliptException.EvaluationException("No result returned from engine"));
  }

  /**
   * listFlags lists all flags in the namespace.
   *
   * @return the list of flags
   */
  public List<Flag> listFlags() {
    Pointer value = CLibrary.INSTANCE.list_flags(this.engine);

    TypeReference<Result<ArrayList<Flag>>> typeRef =
        new TypeReference<Result<ArrayList<Flag>>>() {};

    Result<ArrayList<Flag>> resp = this.readValue(value, typeRef);
    if (!FliptClient.STATUS_SUCCESS.equals(resp.getStatus())) {
      throw new FliptException.EvaluationException(resp.getErrorMessage().orElse("Unknown error"));
    }
    return resp.getResult()
        .orElseThrow(
            () -> new FliptException.EvaluationException("No result returned from engine"));
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
