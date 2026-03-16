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
 * AuthenticationLease.expiring(expiresAt).jwt(token)
 * AuthenticationLease.expiring(expiresAt).maxRetries(3).jwt(token)
 *
 * // Fixed lease (no expiry, no refresh scheduling)
 * AuthenticationLease.fixed().clientToken(token)
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
     * Creates the lease with JWT authentication.
     *
     * @param token the JWT token value
     * @return a new AuthenticationLease
     */
    public AuthenticationLease jwt(String token) {
      return new AuthenticationLease(new JWTAuthentication(token), null, 0);
    }

    /**
     * Creates the lease with client token authentication.
     *
     * @param token the client token value
     * @return a new AuthenticationLease
     */
    public AuthenticationLease clientToken(String token) {
      return new AuthenticationLease(new ClientTokenAuthentication(token), null, 0);
    }
  }

  /** Builder for expiring leases with configurable retries. */
  public static class ExpiringBuilder {
    private final Instant expiresAt;
    private int maxRetries = DEFAULT_MAX_RETRIES;

    private ExpiringBuilder(Instant expiresAt) {
      this.expiresAt = expiresAt;
    }

    /**
     * Sets the maximum number of consecutive refresh failures before stopping. Defaults to 5.
     *
     * @param maxRetries the maximum number of retries
     * @return this builder
     */
    public ExpiringBuilder maxRetries(int maxRetries) {
      this.maxRetries = maxRetries;
      return this;
    }

    /**
     * Creates the lease with JWT authentication.
     *
     * @param token the JWT token value
     * @return a new AuthenticationLease
     */
    public AuthenticationLease jwt(String token) {
      return new AuthenticationLease(new JWTAuthentication(token), expiresAt, maxRetries);
    }

    /**
     * Creates the lease with client token authentication.
     *
     * @param token the client token value
     * @return a new AuthenticationLease
     */
    public AuthenticationLease clientToken(String token) {
      return new AuthenticationLease(new ClientTokenAuthentication(token), expiresAt, maxRetries);
    }
  }
}
