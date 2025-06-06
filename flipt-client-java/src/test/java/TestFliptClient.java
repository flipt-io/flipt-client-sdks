import io.flipt.client.FliptClient;
import io.flipt.client.models.*;
import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

public class TestFliptClient {
  private static FliptClient fliptClient;

  @BeforeEach
  void init() {
    String fliptURL = System.getenv().get("FLIPT_URL");
    String clientToken = System.getenv().get("FLIPT_AUTH_TOKEN");

    assert fliptURL != null && !fliptURL.isEmpty();
    assert clientToken != null && !clientToken.isEmpty();
    fliptClient =
        FliptClient.builder()
            .url(fliptURL)
            .authentication(new ClientTokenAuthentication(clientToken))
            .build();
  }

  @Test
  void testNullFlagKey() throws Exception {
    Assertions.assertThrows(
        IllegalArgumentException.class,
        () -> {
          fliptClient.evaluateBoolean(null, "entity", new HashMap<>());
        });
  }

  @Test
  void testEmptyFlagKey() throws Exception {
    Assertions.assertThrows(
        IllegalArgumentException.class,
        () -> {
          fliptClient.evaluateBoolean("", "entity", new HashMap<>());
        });
  }

  @Test
  void testNullEntityId() throws Exception {
    Assertions.assertThrows(
        IllegalArgumentException.class,
        () -> {
          fliptClient.evaluateBoolean("flag1", null, new HashMap<>());
        });
  }

  @Test
  void testEmptyEntityId() throws Exception {
    Assertions.assertThrows(
        IllegalArgumentException.class,
        () -> {
          fliptClient.evaluateBoolean("flag1", "", new HashMap<>());
        });
  }

  @Test
  void testEvaluateVariant() throws Exception {
    Map<String, String> context = new HashMap<>();
    context.put("fizz", "buzz");

    VariantEvaluationResponse response = fliptClient.evaluateVariant("flag1", "entity", context);

    Assertions.assertEquals("flag1", response.getFlagKey());
    Assertions.assertTrue(response.isMatch());
    Assertions.assertEquals("MATCH_EVALUATION_REASON", response.getReason());
    Assertions.assertEquals("variant1", response.getVariantKey());
    Assertions.assertEquals("segment1", response.getSegmentKeys().get(0));
  }

  @Test
  void testEvaluateBoolean() throws Exception {
    Map<String, String> context = new HashMap<>();
    context.put("fizz", "buzz");

    BooleanEvaluationResponse response =
        fliptClient.evaluateBoolean("flag_boolean", "entity", context);

    Assertions.assertEquals("flag_boolean", response.getFlagKey());
    Assertions.assertTrue(response.isEnabled());
    Assertions.assertEquals("MATCH_EVALUATION_REASON", response.getReason());
  }

  @Test
  void testEvaluateBatch() throws Exception {
    Map<String, String> context = new HashMap<>();
    context.put("fizz", "buzz");

    List<EvaluationRequest> evalRequests =
        Arrays.asList(
            new EvaluationRequest("flag1", "entity", context),
            new EvaluationRequest("flag_boolean", "entity", context),
            new EvaluationRequest("notfound", "entity", context));

    BatchEvaluationResponse response = fliptClient.evaluateBatch(evalRequests);

    Assertions.assertEquals(3, response.getResponses().size());
    List<Response> responses = response.getResponses();

    Assertions.assertTrue(responses.get(0).getVariantEvaluationResponse().isPresent());
    VariantEvaluationResponse variantResponse =
        responses.get(0).getVariantEvaluationResponse().get();
    Assertions.assertEquals("flag1", variantResponse.getFlagKey());
    Assertions.assertTrue(variantResponse.isMatch());
    Assertions.assertEquals("MATCH_EVALUATION_REASON", variantResponse.getReason());
    Assertions.assertEquals("variant1", variantResponse.getVariantKey());
    Assertions.assertEquals("segment1", variantResponse.getSegmentKeys().get(0));

    Assertions.assertTrue(responses.get(1).getBooleanEvaluationResponse().isPresent());
    BooleanEvaluationResponse booleanResponse =
        responses.get(1).getBooleanEvaluationResponse().get();
    Assertions.assertEquals("flag_boolean", booleanResponse.getFlagKey());
    Assertions.assertTrue(booleanResponse.isEnabled());
    Assertions.assertEquals("MATCH_EVALUATION_REASON", booleanResponse.getReason());

    Assertions.assertTrue(responses.get(2).getErrorEvaluationResponse().isPresent());
    ErrorEvaluationResponse errorResponse = responses.get(2).getErrorEvaluationResponse().get();
    Assertions.assertEquals("notfound", errorResponse.getFlagKey());
    Assertions.assertEquals("default", errorResponse.getNamespaceKey());
    Assertions.assertEquals("NOT_FOUND_ERROR_EVALUATION_REASON", errorResponse.getReason());
  }

  @Test
  void testListFlags() throws Exception {
    List<Flag> flags = fliptClient.listFlags();
    Assertions.assertEquals(2, flags.size());
  }

  @Test
  void testGetSnapshot() throws Exception {
    String snapshot = fliptClient.getSnapshot();
    Assertions.assertNotNull(snapshot);
  }

  @Test
  void testSetGetSnapshotWithInvalidSnapshot() {
    FliptClient invalidFliptClient =
        FliptClient.builder()
            .url("http://localhost:8080")
            .errorStrategy(ErrorStrategy.FALLBACK)
            .snapshot("invalid")
            .build();
  }

  @Test
  void testSetGetSnapshotWithInvalidFliptURL() {
    // Get a snapshot from a working client
    String snapshot = fliptClient.getSnapshot();
    Assertions.assertNotNull(snapshot);

    // Create a client with the previous snapshot and an invalid URL
    FliptClient invalidFliptClient =
        FliptClient.builder()
            .url("http://invalid.flipt.com")
            .errorStrategy(ErrorStrategy.FALLBACK)
            .snapshot(snapshot)
            .build();

    Map<String, String> context = new HashMap<>();
    context.put("fizz", "buzz");

    for (int i = 0; i < 5; i++) {
      VariantEvaluationResponse response =
          invalidFliptClient.evaluateVariant("flag1", "entity", context);

      Assertions.assertEquals("flag1", response.getFlagKey());
      Assertions.assertTrue(response.isMatch());
      Assertions.assertEquals("MATCH_EVALUATION_REASON", response.getReason());
      Assertions.assertEquals("variant1", response.getVariantKey());
      Assertions.assertEquals("segment1", response.getSegmentKeys().get(0));

      BooleanEvaluationResponse booleanResponse =
          invalidFliptClient.evaluateBoolean("flag_boolean", "entity", context);

      Assertions.assertEquals("flag_boolean", booleanResponse.getFlagKey());
      Assertions.assertTrue(booleanResponse.isEnabled());
      Assertions.assertEquals("MATCH_EVALUATION_REASON", booleanResponse.getReason());

      List<Flag> flags = invalidFliptClient.listFlags();
      Assertions.assertEquals(2, flags.size());

      String snapshot = invalidFliptClient.getSnapshot();
      Assertions.assertNotNull(snapshot);
    }

    invalidFliptClient.close();
  }

  @AfterEach
  void tearDown() {
    if (fliptClient != null) fliptClient.close();
  }
}
