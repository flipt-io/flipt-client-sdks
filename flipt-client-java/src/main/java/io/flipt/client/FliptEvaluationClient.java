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
  private static final String STATUS_SUCCESS = "success";
  private final Pointer engine;

  private final ObjectMapper objectMapper;

  public interface CLibrary extends Library {
    CLibrary INSTANCE = Native.load("fliptengine", CLibrary.class);

    Pointer initialize_engine(String namespace, String opts);

    Pointer evaluate_boolean(Pointer engine, String evaluation_request);

    Pointer evaluate_variant(Pointer engine, String evaluation_request);

    Pointer evaluate_batch(Pointer engine, String batch_evaluation_request);

    Pointer list_flags(Pointer engine);

    void destroy_engine(Pointer engine);

    void destroy_string(Pointer str);
  }

  private FliptEvaluationClient(String namespace, ClientOptions clientOptions)
      throws EvaluationException {

    ObjectMapper objectMapper = new ObjectMapper();
    objectMapper.registerModule(new Jdk8Module());

    String clientOptionsSerialized;
    try {
      clientOptionsSerialized = objectMapper.writeValueAsString(clientOptions);
    } catch (JsonProcessingException e) {
      throw new EvaluationException(e);
    }

    Pointer engine = CLibrary.INSTANCE.initialize_engine(namespace, clientOptionsSerialized);

    this.objectMapper = objectMapper;
    this.engine = engine;
  }

  public static FliptEvaluationClientBuilder builder() {
    return new FliptEvaluationClientBuilder();
  }

  /** FliptEvaluationClientBuilder is a builder for creating a FliptEvaluationClient. */
  public static final class FliptEvaluationClientBuilder {
    private String namespace = "default";
    private String url = "http://localhost:8080";
    private AuthenticationStrategy authentication;
    private String reference;
    private Duration updateInterval;
    private FetchMode fetchMode = FetchMode.POLLING;

    public FliptEvaluationClientBuilder() {}

    /**
     * url sets the URL for the Flipt server.
     *
     * @param url the URL for the Flipt server
     * @return the FliptEvaluationClientBuilder
     */
    public FliptEvaluationClientBuilder url(String url) {
      this.url = url;
      return this;
    }

    /**
     * namespace sets the namespace for the Flipt server.
     *
     * @param namespace the namespace for the Flipt server
     * @return the FliptEvaluationClientBuilder
     */
    public FliptEvaluationClientBuilder namespace(String namespace) {
      this.namespace = namespace;
      return this;
    }

    /**
     * authentication sets the authentication strategy for the Flipt server.
     *
     * @param authentication the authentication strategy for the Flipt server
     * @return the FliptEvaluationClientBuilder
     */
    public FliptEvaluationClientBuilder authentication(AuthenticationStrategy authentication) {
      this.authentication = authentication;
      return this;
    }

    /**
     * updateInterval sets the update interval for the Flipt server.
     *
     * @param updateInterval the update interval for the Flipt server
     * @return the FliptEvaluationClientBuilder
     */
    public FliptEvaluationClientBuilder updateInterval(Duration updateInterval) {
      this.updateInterval = updateInterval;
      return this;
    }

    /**
     * reference sets the reference for the Flipt server.
     *
     * @param reference the reference for the Flipt server
     * @return the FliptEvaluationClientBuilder
     */
    public FliptEvaluationClientBuilder reference(String reference) {
      this.reference = reference;
      return this;
    }

    /**
     * fetchMode sets the fetch mode for the Flipt server. Note: Streaming is currently only
     * supported when using the SDK with Flipt Cloud (https://flipt.io/cloud).
     *
     * @param fetchMode the fetch mode for the Flipt server
     * @return the FliptEvaluationClientBuilder
     */
    public FliptEvaluationClientBuilder fetchMode(FetchMode fetchMode) {
      this.fetchMode = fetchMode;
      return this;
    }

    /**
     * build builds a new FliptEvaluationClient.
     *
     * @return the FliptEvaluationClient
     * @throws EvaluationException if the FliptEvaluationClient could not be built
     */
    public FliptEvaluationClient build() throws EvaluationException {
      return new FliptEvaluationClient(
          namespace,
          new ClientOptions(
              Optional.of(url),
              Optional.ofNullable(updateInterval),
              Optional.ofNullable(authentication),
              Optional.ofNullable(reference),
              Optional.ofNullable(fetchMode)));
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
   * @throws EvaluationException if the evaluation failed
   */
  public VariantEvaluationResponse evaluateVariant(
      String flagKey, String entityId, Map<String, String> context) throws EvaluationException {
    InternalEvaluationRequest evaluationRequest =
        new InternalEvaluationRequest(flagKey, entityId, context);

    String evaluationRequestSerialized;
    try {
      evaluationRequestSerialized = this.objectMapper.writeValueAsString(evaluationRequest);
    } catch (JsonProcessingException e) {
      throw new EvaluationException(e);
    }

    Pointer value = CLibrary.INSTANCE.evaluate_variant(this.engine, evaluationRequestSerialized);

    TypeReference<Result<VariantEvaluationResponse>> typeRef =
        new TypeReference<Result<VariantEvaluationResponse>>() {};

    Result<VariantEvaluationResponse> resp = this.readValue(value, typeRef);
    if (!FliptEvaluationClient.STATUS_SUCCESS.equals(resp.getStatus())) {
      throw new EvaluationException(resp.getErrorMessage().get());
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
   * @throws EvaluationException if the evaluation failed
   */
  public BooleanEvaluationResponse evaluateBoolean(
      String flagKey, String entityId, Map<String, String> context) throws EvaluationException {
    InternalEvaluationRequest evaluationRequest =
        new InternalEvaluationRequest(flagKey, entityId, context);

    String evaluationRequestSerialized;
    try {
      evaluationRequestSerialized = this.objectMapper.writeValueAsString(evaluationRequest);
    } catch (JsonProcessingException e) {
      throw new EvaluationException(e);
    }
    Pointer value = CLibrary.INSTANCE.evaluate_boolean(this.engine, evaluationRequestSerialized);

    TypeReference<Result<BooleanEvaluationResponse>> typeRef =
        new TypeReference<Result<BooleanEvaluationResponse>>() {};
    Result<BooleanEvaluationResponse> resp = this.readValue(value, typeRef);

    if (!FliptEvaluationClient.STATUS_SUCCESS.equals(resp.getStatus())) {
      throw new EvaluationException(resp.getErrorMessage().get());
    }
    return resp.getResult().get();
  }

  /**
   * evaluateBatch evaluates a batch of flags.
   *
   * @param batchEvaluationRequests the batch of flags to evaluate
   * @return the evaluation response
   * @throws EvaluationException if the evaluation failed
   */
  public BatchEvaluationResponse evaluateBatch(EvaluationRequest[] batchEvaluationRequests)
      throws EvaluationException {
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

    String batchEvaluationRequestSerialized;
    try {
      batchEvaluationRequestSerialized =
          this.objectMapper.writeValueAsString(evaluationRequests.toArray());
    } catch (JsonProcessingException e) {
      throw new EvaluationException(e);
    }
    Pointer value = CLibrary.INSTANCE.evaluate_batch(this.engine, batchEvaluationRequestSerialized);

    TypeReference<Result<BatchEvaluationResponse>> typeRef =
        new TypeReference<Result<BatchEvaluationResponse>>() {};

    Result<BatchEvaluationResponse> resp = this.readValue(value, typeRef);
    if (!FliptEvaluationClient.STATUS_SUCCESS.equals(resp.getStatus())) {
      throw new EvaluationException(resp.getErrorMessage().get());
    }
    return resp.getResult().get();
  }

  /**
   * listFlags lists all flags in the namespace.
   *
   * @return the list of flags
   * @throws EvaluationException if the list of flags could not be retrieved
   */
  public ArrayList<Flag> listFlags() throws EvaluationException {
    Pointer value = CLibrary.INSTANCE.list_flags(this.engine);

    TypeReference<Result<ArrayList<Flag>>> typeRef =
        new TypeReference<Result<ArrayList<Flag>>>() {};

    Result<ArrayList<Flag>> resp = this.readValue(value, typeRef);
    if (!FliptEvaluationClient.STATUS_SUCCESS.equals(resp.getStatus())) {
      throw new EvaluationException(resp.getErrorMessage().get());
    }
    return resp.getResult().get();
  }

  /** close closes the FliptEvaluationClient. */
  public void close() {
    CLibrary.INSTANCE.destroy_engine(this.engine);
  }

  private <T> T readValue(Pointer ptr, TypeReference<T> typeRef) throws EvaluationException {
    try {
      String value = ptr.getString(0, "UTF-8");
      return this.objectMapper.readValue(value, typeRef);
    } catch (JsonProcessingException e) {
      throw new EvaluationException(e);
    } finally {
      CLibrary.INSTANCE.destroy_string(ptr);
    }
  }
}
