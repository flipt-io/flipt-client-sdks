import io.flipt.client.FliptEvaluationClient;
import io.flipt.client.models.*;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Paths;
import java.util.ArrayList;
import java.util.Base64;
import java.util.HashMap;
import java.util.Map;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.skyscreamer.jsonassert.JSONAssert;
import org.skyscreamer.jsonassert.JSONCompareMode;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class TestFliptEvaluationClient {
  private static final Logger logger = LoggerFactory.getLogger(TestFliptEvaluationClient.class);
  private static final String SNAPSHOT =
      Base64.getEncoder()
          .encodeToString(readResourceFile("snapshot.json").getBytes(StandardCharsets.UTF_8));
  private static final String EMPTY_SNAPSHOT =
      Base64.getEncoder()
          .encodeToString(readResourceFile("empty_snapshot.json").getBytes(StandardCharsets.UTF_8));

  private static String readResourceFile(String filename) {
    try {
      return new String(Files.readAllBytes(Paths.get("src/test/resources/" + filename)));
    } catch (IOException e) {
      throw new RuntimeException("Failed to read resource file: " + filename, e);
    }
  }

  private static FliptEvaluationClient fliptClient;

  @BeforeEach
  void init() throws Exception {
    logger.debug("Initializing FliptEvaluationClient");
    String fliptURL = System.getenv().get("FLIPT_URL");
    String clientToken = System.getenv().get("FLIPT_AUTH_TOKEN");

    assert fliptURL != null && !fliptURL.isEmpty();
    assert clientToken != null && !clientToken.isEmpty();
    fliptClient =
        FliptEvaluationClient.builder()
            .url(fliptURL)
            .authentication(new ClientTokenAuthentication(clientToken))
            .build();
    logger.debug("FliptEvaluationClient initialized");
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

    logger.debug(
        "Evaluating variant with flagKey: {}, entity: {}, context: {}", "flag1", "entity", context);
    VariantEvaluationResponse response = fliptClient.evaluateVariant("flag1", "entity", context);
    logger.debug("Variant evaluation response: {}", response);

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

    logger.debug(
        "Evaluating boolean with flagKey: {}, entity: {}, context: {}",
        "flag_boolean",
        "entity",
        context);
    BooleanEvaluationResponse response =
        fliptClient.evaluateBoolean("flag_boolean", "entity", context);
    logger.debug("Boolean evaluation response: {}", response);

    Assertions.assertEquals("flag_boolean", response.getFlagKey());
    Assertions.assertTrue(response.isEnabled());
    Assertions.assertEquals("MATCH_EVALUATION_REASON", response.getReason());
  }

  @Test
  void testEvaluateBatch() throws Exception {
    Map<String, String> context = new HashMap<>();
    context.put("fizz", "buzz");

    logger.debug("Evaluating batch with context: {}", context);
    EvaluationRequest[] evalRequests = {
      new EvaluationRequest("flag1", "entity", context),
      new EvaluationRequest("flag_boolean", "entity", context),
      new EvaluationRequest("notfound", "entity", context)
    };

    logger.debug("Batch evaluation requests: {}", (Object) evalRequests);
    BatchEvaluationResponse response = fliptClient.evaluateBatch(evalRequests);
    logger.debug("Batch evaluation response: {}", response);

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
    logger.debug("Listing flags");
    ArrayList<Flag> flags = fliptClient.listFlags();
    logger.debug("Flags listed: {}", flags);
    Assertions.assertEquals(2, flags.size());
  }

  @Test
  void testGetSnapshot() throws Exception {
    logger.debug("Getting snapshot");
    String snapshot = fliptClient.getSnapshot();
    logger.debug("Snapshot retrieved: {}", snapshot);
    Assertions.assertNotNull(snapshot);

    byte[] expectedBytes = Base64.getDecoder().decode(SNAPSHOT);
    byte[] actualBytes = Base64.getDecoder().decode(snapshot);

    String expectedJson = new String(expectedBytes, StandardCharsets.UTF_8);
    String actualJson = new String(actualBytes, StandardCharsets.UTF_8);

    JSONAssert.assertEquals(expectedJson, actualJson, JSONCompareMode.LENIENT);
  }

  @Test
  void testSetGetSnapshotWithInvalidSnapshot() throws Exception {
    FliptEvaluationClient invalidFliptClient =
        FliptEvaluationClient.builder()
            .url("http://localhost:8080")
            .errorStrategy(ErrorStrategy.FALLBACK)
            .snapshot("invalid")
            .build();
  }

  @Test
  void testSetGetSnapshotWithInvalidFliptURL() throws Exception {
    logger.debug("Running testSetGetSnapshotWithInvalidFliptURL");
    FliptEvaluationClient invalidFliptClient =
        FliptEvaluationClient.builder()
            .url("http://invalid.flipt.com")
            .errorStrategy(ErrorStrategy.FALLBACK)
            .snapshot(SNAPSHOT)
            .build();
    logger.debug("Built FliptEvaluationClient with invalid URL and fallback error strategy");

    Map<String, String> context = new HashMap<>();
    context.put("fizz", "buzz");

    for (int i = 0; i < 5; i++) {
      logger.debug(
          "Iteration {}: Evaluating variant with flagKey=flag1, entity=entity, context={}",
          i,
          context);
      VariantEvaluationResponse response =
          invalidFliptClient.evaluateVariant("flag1", "entity", context);
      logger.debug("Iteration {}: Variant evaluation response: {}", i, response);

      Assertions.assertEquals("flag1", response.getFlagKey());
      Assertions.assertTrue(response.isMatch());
      Assertions.assertEquals("MATCH_EVALUATION_REASON", response.getReason());
      Assertions.assertEquals("variant1", response.getVariantKey());
      Assertions.assertEquals("segment1", response.getSegmentKeys()[0]);

      logger.debug(
          "Iteration {}: Evaluating boolean with flagKey=flag_boolean, entity=entity, context={}",
          i,
          context);
      BooleanEvaluationResponse booleanResponse =
          invalidFliptClient.evaluateBoolean("flag_boolean", "entity", context);
      logger.debug("Iteration {}: Boolean evaluation response: {}", i, booleanResponse);

      Assertions.assertEquals("flag_boolean", booleanResponse.getFlagKey());
      Assertions.assertTrue(booleanResponse.isEnabled());
      Assertions.assertEquals("MATCH_EVALUATION_REASON", booleanResponse.getReason());

      logger.debug("Iteration {}: Listing flags", i);
      ArrayList<Flag> flags = invalidFliptClient.listFlags();
      logger.debug("Iteration {}: Flags listed: {}", i, flags);
      Assertions.assertEquals(2, flags.size());

      logger.debug("Iteration {}: Getting snapshot", i);
      String snapshot = invalidFliptClient.getSnapshot();
      logger.debug("Iteration {}: Snapshot retrieved: {}", i, snapshot);
      Assertions.assertNotNull(snapshot);
    }
    logger.debug("Closing FliptEvaluationClient");
    invalidFliptClient.close();
    logger.debug("FliptEvaluationClient closed");
  }

  @AfterEach
  void tearDown() throws Exception {
    logger.debug("Tearing down test");
    if (fliptClient != null) fliptClient.close();
    logger.debug("FliptEvaluationClient closed");
  }
}
