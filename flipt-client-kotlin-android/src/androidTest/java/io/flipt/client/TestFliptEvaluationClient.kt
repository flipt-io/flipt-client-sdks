package io.flipt.client

import io.flipt.client.models.ClientTokenAuthentication
import io.flipt.client.models.EvaluationRequest
import org.junit.After
import org.junit.Before
import org.junit.Test


class TestFliptEvaluationClient {
    private var fliptClient: FliptEvaluationClient? = null

    @Before
    @Throws(Exception::class)
    fun initAll() {
        val fliptURL = BuildConfig.FLIPT_URL
        val clientToken = BuildConfig.FLIPT_AUTH_TOKEN

        assert(!fliptURL.isEmpty())
        assert(!clientToken.isEmpty())
        fliptClient = FliptEvaluationClient.builder()
            .url(url = fliptURL)
            .namespace("default")
            .authentication(ClientTokenAuthentication(clientToken))
            .build()
    }

    @Test
    @Throws(Exception::class)
    fun testEvaluateVariant() {
        val context: MutableMap<String, String> = HashMap()
        context["fizz"] = "buzz"

        val response  = fliptClient?.evaluateVariant("flag1", "entity", context)

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

        val evalRequests: Array<EvaluationRequest> = arrayOf(
            EvaluationRequest("flag1", "entity", context),
            EvaluationRequest("flag_boolean", "entity", context),
            EvaluationRequest("notfound", "entity", context)
        )

        val response = fliptClient?.evaluateBatch(evalRequests)

        assert(3 == response?.responses?.size)
        val responses = response?.responses

        assert(responses?.get(0)?.variantEvaluationResponse != null)
        val variantResponse  = responses?.get(0)?.variantEvaluationResponse
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

    @After
    @Throws(Exception::class)
    fun tearDownAll() {
        fliptClient?.close()
    }

}
