package io.flipt.client.models;

import java.time.Instant;

/**
 * Holds an authentication strategy and its expiry time.
 *
 * <p>Returned by {@link AuthenticationProvider#getAuthentication()} to provide both the token and
 * information about when it expires.
 */
public class AuthResult {
  private final AuthenticationStrategy strategy;
  private final Instant expiresAt;

  /**
   * Creates a new AuthResult.
   *
   * @param strategy the authentication strategy (e.g., {@link ClientTokenAuthentication} or {@link
   *     JWTAuthentication})
   * @param expiresAt when this token expires
   */
  public AuthResult(AuthenticationStrategy strategy, Instant expiresAt) {
    if (strategy == null) {
      throw new IllegalArgumentException("strategy cannot be null");
    }
    if (expiresAt == null) {
      throw new IllegalArgumentException("expiresAt cannot be null");
    }
    this.strategy = strategy;
    this.expiresAt = expiresAt;
  }

  public AuthenticationStrategy getStrategy() {
    return strategy;
  }

  public Instant getExpiresAt() {
    return expiresAt;
  }
}
