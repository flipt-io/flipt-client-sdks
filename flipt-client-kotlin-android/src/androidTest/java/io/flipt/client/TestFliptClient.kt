package io.flipt.client

import io.flipt.client.FliptClient
import io.flipt.client.models.*
import org.junit.After
import org.junit.Assert.*
import org.junit.Before
import org.junit.Test
import java.io.File
import java.nio.charset.StandardCharsets
import java.util.Base64

class TestFliptClient {
    companion object {
        private val SNAPSHOT: String by lazy {
            Base64.getEncoder().encodeToString(readResourceFile("snapshot.json").toByteArray(StandardCharsets.UTF_8))
        }
        private val EMPTY_SNAPSHOT: String by lazy {
            Base64.getEncoder().encodeToString(readResourceFile("empty_snapshot.json").toByteArray(StandardCharsets.UTF_8))
        }

        private fun readResourceFile(filename: String): String {
            val resourcePath = "src/test/resources/$filename"
            return File(resourcePath).readText(Charsets.UTF_8)
        }
    }

    private var fliptClient: FliptClient? = null

    @Before
    @Throws(Exception::class)
    fun setUp() {
        val fliptURL = System.getenv("FLIPT_URL") ?: "http://10.0.2.2:8080"
        val clientToken = System.getenv("FLIPT_AUTH_TOKEN") ?: "secret"

        require(fliptURL.isNotEmpty()) { "FLIPT_URL environment variable is required" }
        require(clientToken.isNotEmpty()) { "FLIPT_AUTH_TOKEN environment variable is required" }
        require(fliptURL == "http://10.0.2.2:8080") { "FLIPT_URL must be http://10.0.2.2:8080" }

        fliptClient =
            FliptClient
                .builder()
                .url(url = fliptURL)
                .namespace("default")
                .authentication(ClientTokenAuthentication(clientToken))
                .build()
    }

    @After
    @Throws(Exception::class)
    fun tearDown() {
        fliptClient?.close()
    }

    @Test
    @Throws(Exception::class)
    fun testEvaluateVariant() {
        val context: MutableMap<String, String> = HashMap()
        context["fizz"] = "buzz"

        val response = fliptClient?.evaluateVariant("flag1", "entity", context)

        assert("flag1" == response?.flagKey)
        assert(response?.match ?: false)
        assert("MATCH_EVALUATION_REASON" == response?.reason)
        assert("variant1" == response?.variantKey)
        assert("segment1" == response?.segmentKeys?.get(0))
    }

    @Test
    @Throws(Exception::class)
    fun testEvaluateBoolean() {
        val context: MutableMap<String, String> = HashMap()
        context["fizz"] = "buzz"

        val response = fliptClient?.evaluateBoolean("flag_boolean", "entity", context)

        assert("flag_boolean" == response?.flagKey)
        assert(response?.enabled ?: false)
        assert("MATCH_EVALUATION_REASON" == response?.reason)
    }

    @Test
    @Throws(Exception::class)
    fun testEvaluateBatch() {
        val context: MutableMap<String, String> = HashMap()
        context["fizz"] = "buzz"

        val evalRequests: Array<EvaluationRequest> =
            arrayOf(
                EvaluationRequest("flag1", "entity", context),
                EvaluationRequest("flag_boolean", "entity", context),
                EvaluationRequest("notfound", "entity", context),
            )

        val response = fliptClient?.evaluateBatch(evalRequests)

        assert(3 == response?.responses?.size)
        val responses = response?.responses

        assert(responses?.get(0)?.variantEvaluationResponse != null)
        val variantResponse = responses?.get(0)?.variantEvaluationResponse
        assert("flag1" == variantResponse?.flagKey)
        assert(variantResponse?.match ?: false)
        assert("MATCH_EVALUATION_REASON" == variantResponse?.reason)
        assert("variant1" == variantResponse?.variantKey)
        assert("segment1" == variantResponse?.segmentKeys?.get(0))

        assert(responses?.get(1)?.booleanEvaluationResponse != null)
        val booleanResponse = responses?.get(1)?.booleanEvaluationResponse
        assert("flag_boolean" == booleanResponse?.flagKey)
        assert(booleanResponse?.enabled ?: false)
        assert("MATCH_EVALUATION_REASON" == booleanResponse?.reason)

        assert(responses?.get(2)?.errorEvaluationResponse != null)
        val errorResponse = responses?.get(2)?.errorEvaluationResponse
        assert("notfound" == errorResponse?.flagKey)
        assert("default" == errorResponse?.namespaceKey)
        assert("NOT_FOUND_ERROR_EVALUATION_REASON" == errorResponse?.reason)
    }

    @Test
    @Throws(Exception::class)
    fun testListFlags() {
        val flags = fliptClient?.listFlags()
        assert(2 == flags?.size)
    }

    @Test
    fun testGetSnapshot() {
        val snapshot = fliptClient!!.getSnapshot()
        assertNotNull(snapshot)
        val expectedBytes = Base64.getDecoder().decode(SNAPSHOT)
        val actualBytes = Base64.getDecoder().decode(snapshot)
        val expectedJson = String(expectedBytes, StandardCharsets.UTF_8)
        val actualJson = String(actualBytes, StandardCharsets.UTF_8)
        // Use JSON comparison for leniency
        assertTrue(jsonEquals(expectedJson, actualJson))
    }

    @Test
    fun testSetGetSnapshotWithInvalidFliptURL() {
        val invalidFliptClient =
            FliptClient
                .builder()
                .url("http://invalid.flipt.com")
                .errorStrategy(ErrorStrategy.FALLBACK)
                .snapshot(SNAPSHOT)
                .build()

        val context: MutableMap<String, String> = HashMap()
        context["fizz"] = "buzz"

        repeat(5) {
            val response = invalidFliptClient.evaluateVariant("flag1", "entity", context)
            assert("flag1" == response.flagKey)
            assert(response.match)
            assert("MATCH_EVALUATION_REASON" == response.reason)
            assert("variant1" == response.variantKey)
            assert("segment1" == response.segmentKeys[0])

            val booleanResponse = invalidFliptClient.evaluateBoolean("flag_boolean", "entity", context)
            assert("flag_boolean" == booleanResponse.flagKey)
            assert(booleanResponse.enabled)
            assert("MATCH_EVALUATION_REASON" == booleanResponse.reason)

            val flags = invalidFliptClient.listFlags()
            assert(2 == flags.size)

            val snapshot = invalidFliptClient.getSnapshot()
            assertNotNull(snapshot)
        }

        invalidFliptClient.close()
    }

    // Simple JSON comparison ignoring order and whitespace
    private fun jsonEquals(
        expected: String,
        actual: String,
    ): Boolean {
        val expectedObj = com.fasterxml.jackson.module.kotlin.jacksonObjectMapper().readTree(expected)
        val actualObj = com.fasterxml.jackson.module.kotlin.jacksonObjectMapper().readTree(actual)
        return expectedObj == actualObj
    }
}
