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
import java.time.Instant;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import java.util.concurrent.CopyOnWriteArrayList;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.logging.Level;
import java.util.logging.Logger;
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
 *
 * // Get notified when flag state changes
 * var listeningClient = FliptClient.builder()
 *     .url("http://localhost:8080")
 *     .fetchMode(FetchMode.STREAMING)
 *     .onStateUpdate(this::refreshFlags)
 *     .build();
 *
 * // Or subscribe while the client is running
 * FliptClient.Subscription sub = listeningClient.addStateUpdateListener(this::refreshFlags);
 * sub.close(); // unsubscribe
 * }</pre>
 *
 * @since 1.0.0
 */
public class FliptClient implements AutoCloseable {
  private static final String STATUS_SUCCESS = "success";

  /**
   * Partial builder definition to add deprecation on {@code authentication}. Lombok fills in the
   * rest.
   */
  public static class FliptClientBuilder {
    /**
     * @deprecated Use {@code authenticationProvider(AuthenticationProvider)} instead with {@link
     *     AuthenticationLease#fixed()} for static credentials.
     */
    @Deprecated
    public FliptClientBuilder authentication(AuthenticationStrategy authentication) {
      this.authentication = authentication;
      return this;
    }
  }

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
  private AuthenticationProvider authenticationProvider;
  private int maxAuthRetries;
  private volatile Instant currentExpiry;
  private int consecutiveAuthFailures = 0;
  private final AtomicBoolean closed = new AtomicBoolean(false);
  private ScheduledExecutorService authScheduler;
  private static final Duration EXPIRY_BUFFER = Duration.ofSeconds(30);
  private static final Duration MIN_RETRY_DELAY = Duration.ofSeconds(5);
  private static final Logger logger = Logger.getLogger(FliptClient.class.getName());

  private final Pointer engine;
  private final ObjectMapper objectMapper;
  private final List<Runnable> stateUpdateListeners = new CopyOnWriteArrayList<>();
  // Serializes starting the update thread against close(), so it never subscribes to a destroyed
  // engine. Never held while joining the thread or destroying the engine.
  private final Object stateUpdateLock = new Object();
  // Started on the first listener. It owns the native subscription and frees it on exit.
  private Thread stateUpdateThread;
  private Pointer stateSubscription;

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

    Pointer update_authentication(Pointer engine, String auth_json);

    Pointer subscribe_state_updates(Pointer engine);

    /** Blocks until flag state changes or the subscription closes. */
    Pointer wait_for_state_update(Pointer subscription);

    void close_state_subscription(Pointer subscription);

    void destroy_state_subscription(Pointer subscription);
  }

  @Builder
  private FliptClient(
      String environment,
      String namespace,
      String url,
      AuthenticationStrategy authentication,
      AuthenticationProvider authenticationProvider,
      String reference,
      Duration requestTimeout,
      Duration updateInterval,
      FetchMode fetchMode,
      ErrorStrategy errorStrategy,
      String snapshot,
      TlsConfig tlsConfig,
      Runnable onStateUpdate) {
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
    this.authenticationProvider = authenticationProvider;

    if (authenticationProvider != null && authentication != null) {
      throw new IllegalArgumentException(
          "Cannot set both authentication and authenticationProvider");
    }

    if (this.authenticationProvider != null) {
      AuthenticationLease initial = this.authenticationProvider.get();
      this.authentication = initial.getStrategy();
      this.currentExpiry = initial.getExpiresAt().orElse(null);
      this.maxAuthRetries = initial.getMaxRetries().orElse(0);
    }

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

    if (onStateUpdate != null) {
      try {
        addStateUpdateListener(onStateUpdate);
      } catch (Throwable t) {
        // The caller never gets a client to close, so tear the engine down here.
        CLibrary.INSTANCE.destroy_engine(this.engine);
        throw t;
      }
    }

    // Start auth refresh scheduler only for bounded leases (with expiry)
    if (this.authenticationProvider != null && this.currentExpiry != null) {
      startAuthRefreshScheduler();
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

  /** A registered state update listener. Closing it unsubscribes; closing twice is a no-op. */
  public interface Subscription extends AutoCloseable {
    @Override
    void close();
  }

  /**
   * Adds a listener that is called after each change to flag state. Listeners can be added at any
   * time. Read the flags you care about once after adding a listener: it is called for changes
   * applied after it was added, and may occasionally see one extra call.
   *
   * <p>Listeners run in the order they were added, on a dedicated daemon thread named {@code
   * flipt-state-update}. Updates that arrive while they run are coalesced into a single follow-up
   * call. A listener may evaluate flags, add or remove listeners, or call {@link #close()}. {@link
   * #close()} from another thread waits for a running listener to return, so a listener must not
   * block on anything held by the thread that closes the client. An exception thrown by one
   * listener is logged and does not stop the others.
   *
   * <p>The client must be closed to stop its update thread.
   *
   * @param listener the listener to call
   * @return a subscription that removes the listener when closed
   * @throws IllegalStateException if the client is closed
   * @throws FliptException.EvaluationException if the engine cannot subscribe to updates
   */
  public Subscription addStateUpdateListener(Runnable listener) {
    if (listener == null) {
      throw new IllegalArgumentException("listener is required");
    }
    if (this.closed.get()) {
      throw new IllegalStateException("client is closed");
    }
    // A distinct wrapper per call, so adding the same Runnable twice yields two independent
    // subscriptions.
    Runnable entry = listener::run;
    this.stateUpdateListeners.add(entry);
    try {
      ensureStateUpdateThread();
    } catch (RuntimeException e) {
      this.stateUpdateListeners.remove(entry);
      throw e;
    }
    // close() may have finished between the checks above and the add.
    if (this.closed.get()) {
      this.stateUpdateListeners.remove(entry);
      throw new IllegalStateException("client is closed");
    }
    return () -> this.stateUpdateListeners.remove(entry);
  }

  /**
   * Starts the single thread that waits on the engine and fans out to all listeners, on first use.
   * It runs until close(), so adding or removing listeners never calls into the engine, including
   * from inside a listener.
   */
  private void ensureStateUpdateThread() {
    synchronized (this.stateUpdateLock) {
      if (this.stateUpdateThread != null) {
        return;
      }
      if (this.closed.get()) {
        throw new IllegalStateException("client is closed");
      }
      Pointer subscription = CLibrary.INSTANCE.subscribe_state_updates(this.engine);
      if (subscription == null) {
        throw new FliptException.EvaluationException("Failed to subscribe to state updates");
      }
      Thread thread = new Thread(() -> runStateUpdateLoop(subscription), "flipt-state-update");
      // Never keep the JVM alive just to wait for flag changes.
      thread.setDaemon(true);
      try {
        thread.start();
      } catch (Throwable t) {
        CLibrary.INSTANCE.destroy_state_subscription(subscription);
        throw t;
      }
      // The thread can't exit before these are set: its cleanup takes this lock.
      this.stateSubscription = subscription;
      this.stateUpdateThread = thread;
    }
  }

  private void runStateUpdateLoop(Pointer subscription) {
    TypeReference<Result<String>> typeRef = new TypeReference<Result<String>>() {};
    try {
      while (!this.closed.get()) {
        Result<String> resp =
            this.readValue(CLibrary.INSTANCE.wait_for_state_update(subscription), typeRef);
        if (!STATUS_SUCCESS.equals(resp.getStatus())) {
          logger.warning(
              "Failed to wait for state update: " + resp.getErrorMessage().orElse("Unknown error"));
          return;
        }
        if (!"changed".equals(resp.getResult().orElse(null))) {
          return;
        }
        for (Runnable listener : this.stateUpdateListeners) {
          if (this.closed.get()) {
            return;
          }
          try {
            listener.run();
          } catch (Throwable t) {
            logger.log(Level.WARNING, "State update listener threw", t);
          }
        }
      }
    } catch (Throwable t) {
      logger.log(Level.WARNING, "State update thread failed", t);
    } finally {
      // This thread is the only one that waits on the subscription, so it frees it. Under the lock,
      // so close() never wakes a handle that is already freed.
      synchronized (this.stateUpdateLock) {
        this.stateSubscription = null;
        CLibrary.INSTANCE.destroy_state_subscription(subscription);
        // If this thread died early, the next listener starts a new one.
        if (this.stateUpdateThread == Thread.currentThread()) {
          this.stateUpdateThread = null;
        }
      }
    }
  }

  private void startAuthRefreshScheduler() {
    this.authScheduler =
        Executors.newSingleThreadScheduledExecutor(
            r -> {
              Thread t = new Thread(r, "flipt-auth-refresh");
              t.setDaemon(true);
              return t;
            });

    scheduleNextAuthCheck();
  }

  private void scheduleNextAuthCheck() {
    Duration delay = Duration.between(Instant.now(), this.currentExpiry.minus(EXPIRY_BUFFER));
    if (delay.isNegative() || delay.isZero()) {
      delay = MIN_RETRY_DELAY;
    }

    this.authScheduler.schedule(
        () -> {
          if (this.closed.get()) {
            return;
          }
          try {
            AuthenticationLease lease = this.authenticationProvider.get();
            if (this.closed.get()) {
              return;
            }
            String authJson = this.objectMapper.writeValueAsString(lease.getStrategy());
            Pointer response = CLibrary.INSTANCE.update_authentication(this.engine, authJson);

            TypeReference<Result<Void>> typeRef = new TypeReference<Result<Void>>() {};
            Result<Void> resp = this.readValue(response, typeRef);
            if (!STATUS_SUCCESS.equals(resp.getStatus())) {
              this.consecutiveAuthFailures++;
              logger.warning(
                  "Failed to update engine authentication: "
                      + resp.getErrorMessage().orElse("Unknown error"));
            } else {
              this.consecutiveAuthFailures = 0;
              this.currentExpiry = lease.getExpiresAt().orElse(null);
            }
          } catch (Exception e) {
            this.consecutiveAuthFailures++;
            logger.warning("Failed to refresh authentication: " + e.getMessage());
          }

          if (this.closed.get()) {
            return;
          }
          if (this.currentExpiry == null) {
            return;
          }
          if (this.consecutiveAuthFailures >= this.maxAuthRetries) {
            logger.severe(
                "Authentication refresh failed after "
                    + this.maxAuthRetries
                    + " consecutive attempts, stopping refresh");
            return;
          }
          scheduleNextAuthCheck();
        },
        delay.toMillis(),
        TimeUnit.MILLISECONDS);
  }

  @Override
  public void close() {
    if (!this.closed.compareAndSet(false, true)) {
      return;
    }
    if (this.authScheduler != null) {
      this.authScheduler.shutdown();
      try {
        if (!this.authScheduler.awaitTermination(5, TimeUnit.SECONDS)) {
          logger.warning("Auth refresh scheduler did not terminate in time, forcing shutdown");
          this.authScheduler.shutdownNow();
        }
      } catch (InterruptedException e) {
        logger.warning("Interrupted while waiting for auth refresh scheduler to terminate");
        this.authScheduler.shutdownNow();
      }
    }
    // Wait out a thread start already in progress; later ones see closed and give up. The lock is
    // released before joining, since a running listener may add another listener.
    Thread thread;
    synchronized (this.stateUpdateLock) {
      thread = this.stateUpdateThread;
      if (this.stateSubscription != null) {
        // Wakes the update thread if it is waiting. It still owns and frees the handle.
        CLibrary.INSTANCE.close_state_subscription(this.stateSubscription);
      }
    }
    if (thread != null && thread != Thread.currentThread()) {
      // Let a running listener finish before the engine it may be evaluating against is freed.
      boolean interrupted = false;
      while (thread.isAlive()) {
        try {
          thread.join();
        } catch (InterruptedException e) {
          interrupted = true;
        }
      }
      if (interrupted) {
        Thread.currentThread().interrupt();
      }
    }
    if (this.engine != null) {
      CLibrary.INSTANCE.destroy_engine(this.engine);
    }
    this.stateUpdateListeners.clear();
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
