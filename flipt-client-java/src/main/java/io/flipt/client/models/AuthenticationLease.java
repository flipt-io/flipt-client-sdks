package io.flipt.client.models;

import java.time.Instant;

/**
 * An authentication credential with an expiry time.
 *
 * <p>Returned by {@link AuthenticationProvider#get()} to provide both the credential and when it
 * expires. Use the factory methods to create instances:
 *
 * <pre>{@code
 * AuthenticationLease.jwt(token, expiresAt)
 * AuthenticationLease.clientToken(token, expiresAt)
 * }</pre>
 */
public class AuthenticationLease {
  private final AuthenticationStrategy strategy;
  private final Instant expiresAt;

  private AuthenticationLease(AuthenticationStrategy strategy, Instant expiresAt) {
    if (strategy == null) {
      throw new IllegalArgumentException("strategy cannot be null");
    }
    if (expiresAt == null) {
      throw new IllegalArgumentException("expiresAt cannot be null");
    }
    this.strategy = strategy;
    this.expiresAt = expiresAt;
  }

  /**
   * Creates a lease for a JWT token.
   *
   * @param token the JWT token value
   * @param expiresAt when this token expires
   * @return a new AuthenticationLease
   */
  public static AuthenticationLease jwt(String token, Instant expiresAt) {
    return new AuthenticationLease(new JWTAuthentication(token), expiresAt);
  }

  /**
   * Creates a lease for a client token.
   *
   * @param token the client token value
   * @param expiresAt when this token expires
   * @return a new AuthenticationLease
   */
  public static AuthenticationLease clientToken(String token, Instant expiresAt) {
    return new AuthenticationLease(new ClientTokenAuthentication(token), expiresAt);
  }

  public AuthenticationStrategy getStrategy() {
    return strategy;
  }

  public Instant getExpiresAt() {
    return expiresAt;
  }
}
