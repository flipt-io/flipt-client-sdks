import io.flipt.client.FliptEvaluationClient;
import io.flipt.client.models.*;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.Map;
import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Test;

public class TestFliptEvaluationClient {
  private static FliptEvaluationClient fliptClient;

  @BeforeAll
  static void initAll() throws Exception {
    String fliptURL = System.getenv().get("FLIPT_URL");
    String clientToken = System.getenv().get("FLIPT_AUTH_TOKEN");

    assert fliptURL != null && !fliptURL.isEmpty();
    assert clientToken != null && !clientToken.isEmpty();
    fliptClient =
        FliptEvaluationClient.builder()
            .url(fliptURL)
            .authentication(new ClientTokenAuthentication(clientToken))
            .build();
  }

  @Test
  void testEvaluateVariant() throws Exception {
    Map<String, String> context = new HashMap<>();
    context.put("fizz", "buzz");

    Result<VariantEvaluationResponse> result =
        fliptClient.evaluateVariant("flag1", "entity", context);

    Assertions.assertEquals("success", result.getStatus());
    Assertions.assertFalse(result.getErrorMessage().isPresent());
    Assertions.assertTrue(result.getResult().isPresent());

    VariantEvaluationResponse response = result.getResult().get();

    Assertions.assertEquals("flag1", response.getFlagKey());
    Assertions.assertTrue(response.isMatch());
    Assertions.assertEquals("MATCH_EVALUATION_REASON", response.getReason());
    Assertions.assertEquals("variant1", response.getVariantKey());
    Assertions.assertEquals("segment1", response.getSegmentKeys()[0]);
  }

  @Test
  void testEvaluateBoolean() throws Exception {
    Map<String, String> context = new HashMap<>();
    context.put("fizz", "buzz");

    Result<BooleanEvaluationResponse> result =
        fliptClient.evaluateBoolean("flag_boolean", "entity", context);

    Assertions.assertEquals("success", result.getStatus());
    Assertions.assertFalse(result.getErrorMessage().isPresent());
    Assertions.assertTrue(result.getResult().isPresent());

    BooleanEvaluationResponse response = result.getResult().get();
    Assertions.assertEquals("flag_boolean", response.getFlagKey());
    Assertions.assertTrue(response.isEnabled());
    Assertions.assertEquals("MATCH_EVALUATION_REASON", response.getReason());
  }

  @Test
  void testEvaluateBatch() throws Exception {
    Map<String, String> context = new HashMap<>();
    context.put("fizz", "buzz");

    EvaluationRequest[] evalRequests = {
      new EvaluationRequest("flag1", "entity", context),
      new EvaluationRequest("flag_boolean", "entity", context),
      new EvaluationRequest("notfound", "entity", context)
    };

    Result<BatchEvaluationResponse> result = fliptClient.evaluateBatch(evalRequests);

    Assertions.assertEquals("success", result.getStatus());
    Assertions.assertFalse(result.getErrorMessage().isPresent());
    Assertions.assertTrue(result.getResult().isPresent());
    BatchEvaluationResponse response = result.getResult().get();

    Assertions.assertEquals(3, response.getResponses().length);
    Response[] responses = response.getResponses();

    Assertions.assertTrue(responses[0].getVariantEvaluationResponse().isPresent());
    VariantEvaluationResponse variantResponse = responses[0].getVariantEvaluationResponse().get();
    Assertions.assertEquals("flag1", variantResponse.getFlagKey());
    Assertions.assertTrue(variantResponse.isMatch());
    Assertions.assertEquals("MATCH_EVALUATION_REASON", variantResponse.getReason());
    Assertions.assertEquals("variant1", variantResponse.getVariantKey());
    Assertions.assertEquals("segment1", variantResponse.getSegmentKeys()[0]);

    Assertions.assertTrue(responses[1].getBooleanEvaluationResponse().isPresent());
    BooleanEvaluationResponse booleanResponse = responses[1].getBooleanEvaluationResponse().get();
    Assertions.assertEquals("flag_boolean", booleanResponse.getFlagKey());
    Assertions.assertTrue(booleanResponse.isEnabled());
    Assertions.assertEquals("MATCH_EVALUATION_REASON", booleanResponse.getReason());

    Assertions.assertTrue(responses[2].getErrorEvaluationResponse().isPresent());
    ErrorEvaluationResponse errorResponse = responses[2].getErrorEvaluationResponse().get();
    Assertions.assertEquals("notfound", errorResponse.getFlagKey());
    Assertions.assertEquals("default", errorResponse.getNamespaceKey());
    Assertions.assertEquals("NOT_FOUND_ERROR_EVALUATION_REASON", errorResponse.getReason());
  }

  @Test
  void testListFlags() throws Exception {
    Result<ArrayList<Flag>> result = fliptClient.listFlags();

    Assertions.assertEquals("success", result.getStatus());
    Assertions.assertFalse(result.getErrorMessage().isPresent());
    Assertions.assertTrue(result.getResult().isPresent());

    ArrayList<Flag> flags = result.getResult().get();
    Assertions.assertEquals(2, flags.size());
  }

  @AfterAll
  static void tearDownAll() throws Exception {
    if (fliptClient != null) fliptClient.close();
  }
}
