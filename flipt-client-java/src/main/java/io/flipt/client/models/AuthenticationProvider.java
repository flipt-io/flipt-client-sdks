package io.flipt.client.models;

/**
 * A provider for authentication credentials that supports token refresh.
 *
 * <p>Implement this interface to provide dynamic authentication tokens that can be refreshed when
 * they expire. The SDK calls {@link #get()} when the current token is about to expire.
 *
 * <pre>{@code
 * FliptClient.builder()
 *     .url("https://flipt.example.com")
 *     .authenticationProvider(() -> {
 *         Token token = myOAuthClient.getAccessToken();
 *         return AuthenticationLease.jwt(
 *             token.getValue(),
 *             token.getExpiresAt()
 *         );
 *     })
 *     .build();
 * }</pre>
 */
@FunctionalInterface
public interface AuthenticationProvider {
  /**
   * Returns the current authentication credentials and their expiry time.
   *
   * @return an {@link AuthenticationLease} containing the credential and expiry
   */
  AuthenticationLease get();
}
