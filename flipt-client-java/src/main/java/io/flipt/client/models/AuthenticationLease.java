package io.flipt.client.models;

import java.time.Instant;
import java.util.Optional;
import java.util.OptionalInt;

/**
 * An authentication credential with an optional expiry time.
 *
 * <p>Returned by {@link AuthenticationProvider#get()} to provide both the credential and when it
 * expires. Use the fluent builder to create instances:
 *
 * <pre>{@code
 * // Expiring lease (triggers refresh before expiry)
 * AuthenticationLease.expiring(expiresAt).jwt(token).build()
 * AuthenticationLease.expiring(expiresAt).jwt(token).maxRetries(3).build()
 *
 * // Fixed lease (no expiry, no refresh scheduling)
 * AuthenticationLease.fixed().clientToken(token).build()
 * }</pre>
 */
public class AuthenticationLease {
  private static final int DEFAULT_MAX_RETRIES = 5;

  private final AuthenticationStrategy strategy;
  private final Instant expiresAt;
  private final int maxRetries;

  private AuthenticationLease(AuthenticationStrategy strategy, Instant expiresAt, int maxRetries) {
    if (strategy == null) {
      throw new IllegalArgumentException("strategy cannot be null");
    }
    this.strategy = strategy;
    this.expiresAt = expiresAt;
    this.maxRetries = maxRetries;
  }

  /**
   * Starts building a fixed lease with no expiry. No refresh will be scheduled.
   *
   * @return a builder to select the authentication type
   */
  public static FixedBuilder fixed() {
    return new FixedBuilder();
  }

  /**
   * Starts building an expiring lease that triggers a refresh before the given expiry time.
   *
   * @param expiresAt when this credential expires
   * @return a builder to select the authentication type and configure retries
   */
  public static ExpiringBuilder expiring(Instant expiresAt) {
    if (expiresAt == null) {
      throw new IllegalArgumentException("expiresAt cannot be null; use fixed() for no expiry");
    }
    return new ExpiringBuilder(expiresAt);
  }

  public AuthenticationStrategy getStrategy() {
    return strategy;
  }

  /**
   * Returns the expiry time of this lease, or empty if the lease does not expire.
   *
   * @return an Optional containing the expiry time, or empty for fixed leases
   */
  public Optional<Instant> getExpiresAt() {
    return Optional.ofNullable(expiresAt);
  }

  /**
   * Returns the maximum number of consecutive refresh retries, or empty for fixed leases.
   *
   * @return the max retries for expiring leases, empty for fixed leases
   */
  public OptionalInt getMaxRetries() {
    if (expiresAt == null) {
      return OptionalInt.empty();
    }
    return OptionalInt.of(maxRetries);
  }

  /** Builder for fixed leases (no expiry, no retries). */
  public static class FixedBuilder {
    private FixedBuilder() {}

    /**
     * Sets JWT authentication for this lease.
     *
     * @param token the JWT token value
     * @return the build step to finalize the lease
     */
    public FixedBuildStep jwt(String token) {
      return new FixedBuildStep(new JWTAuthentication(token));
    }

    /**
     * Sets client token authentication for this lease.
     *
     * @param token the client token value
     * @return the build step to finalize the lease
     */
    public FixedBuildStep clientToken(String token) {
      return new FixedBuildStep(new ClientTokenAuthentication(token));
    }
  }

  /** Final build step for fixed leases. */
  public static class FixedBuildStep {
    private final AuthenticationStrategy strategy;

    private FixedBuildStep(AuthenticationStrategy strategy) {
      this.strategy = strategy;
    }

    /**
     * Builds the fixed authentication lease.
     *
     * @return a new AuthenticationLease
     */
    public AuthenticationLease build() {
      return new AuthenticationLease(strategy, null, 0);
    }
  }

  /** Builder for expiring leases. */
  public static class ExpiringBuilder {
    private final Instant expiresAt;

    private ExpiringBuilder(Instant expiresAt) {
      this.expiresAt = expiresAt;
    }

    /**
     * Sets JWT authentication for this lease.
     *
     * @param token the JWT token value
     * @return the build step to configure retries and finalize the lease
     */
    public ExpiringBuildStep jwt(String token) {
      return new ExpiringBuildStep(new JWTAuthentication(token), expiresAt);
    }

    /**
     * Sets client token authentication for this lease.
     *
     * @param token the client token value
     * @return the build step to configure retries and finalize the lease
     */
    public ExpiringBuildStep clientToken(String token) {
      return new ExpiringBuildStep(new ClientTokenAuthentication(token), expiresAt);
    }
  }

  /** Final build step for expiring leases with configurable retries. */
  public static class ExpiringBuildStep {
    private final AuthenticationStrategy strategy;
    private final Instant expiresAt;
    private int maxRetries = DEFAULT_MAX_RETRIES;

    private ExpiringBuildStep(AuthenticationStrategy strategy, Instant expiresAt) {
      this.strategy = strategy;
      this.expiresAt = expiresAt;
    }

    /**
     * Sets the maximum number of consecutive refresh failures before stopping. Defaults to 5.
     *
     * @param maxRetries the maximum number of retries
     * @return this build step
     */
    public ExpiringBuildStep maxRetries(int maxRetries) {
      if (maxRetries < 0) {
        throw new IllegalArgumentException("maxRetries must be non-negative");
      }
      this.maxRetries = maxRetries;
      return this;
    }

    /**
     * Builds the expiring authentication lease.
     *
     * @return a new AuthenticationLease
     */
    public AuthenticationLease build() {
      return new AuthenticationLease(strategy, expiresAt, maxRetries);
    }
  }
}
