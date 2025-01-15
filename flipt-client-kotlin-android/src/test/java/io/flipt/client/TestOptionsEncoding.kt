package io.flipt.client
import io.flipt.client.models.AuthenticationStrategy
import io.flipt.client.models.ClientTokenAuthentication
import io.flipt.client.models.JWTAuthentication
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Test

class TestOptionsEncoding {
    @Test
    fun testEncodeAuthenticationStrategy() {
        val json =
            Json {
                encodeDefaults = false
                ignoreUnknownKeys = true
            }
        var clientAuth: AuthenticationStrategy = ClientTokenAuthentication("clientToken")
        var encoded = json.encodeToString(clientAuth)
        var expected = """{"client_token":"clientToken"}"""
        assertEquals("output doesn't match expected", expected, encoded)
        var jwtAuth = JWTAuthentication("jwt")
        encoded = json.encodeToString(jwtAuth)
        expected = """{"jwt_token":"jwt"}"""
        assertEquals("output doesn't match expected", expected, encoded)
    }
}
