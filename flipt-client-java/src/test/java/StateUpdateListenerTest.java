import com.sun.net.httpserver.HttpServer;
import io.flipt.client.FliptClient;
import io.flipt.client.models.ErrorStrategy;
import io.flipt.client.models.FetchMode;
import java.io.IOException;
import java.io.OutputStream;
import java.net.InetSocketAddress;
import java.nio.charset.StandardCharsets;
import java.time.Duration;
import java.util.HashMap;
import java.util.concurrent.BlockingQueue;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.Executors;
import java.util.concurrent.LinkedBlockingQueue;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.concurrent.atomic.AtomicInteger;
import java.util.concurrent.atomic.AtomicReference;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

/** Exercises onStateUpdate against a local HTTP server, so no Flipt instance is needed. */
public class StateUpdateListenerTest {
  private static final String EMPTY = "{\"namespace\":{\"key\":\"default\"},\"flags\":[]}";
  private static final String WITH_FLAG =
      "{\"namespace\":{\"key\":\"default\"},\"flags\":[{\"key\":\"new_flag\",\"name\":\"New\","
          + "\"type\":\"BOOLEAN_FLAG_TYPE\",\"enabled\":true}]}";

  private static final String END_STREAM = "<end>";

  private HttpServer server;
  private final AtomicReference<String> body = new AtomicReference<>(EMPTY);
  // Documents the stream endpoint sends, one per line, as the test pushes them.
  private final BlockingQueue<String> streamDocs = new LinkedBlockingQueue<>();

  @BeforeEach
  void startServer() throws Exception {
    server = HttpServer.create(new InetSocketAddress("127.0.0.1", 0), 0);
    server.createContext(
        "/internal/v1/evaluation/snapshot/namespace/default",
        exchange -> {
          byte[] bytes = body.get().getBytes(StandardCharsets.UTF_8);
          exchange.getResponseHeaders().add("Content-Type", "application/json");
          exchange.sendResponseHeaders(200, bytes.length);
          exchange.getResponseBody().write(bytes);
          exchange.close();
        });
    server.createContext(
        "/client/v2/environments/default/namespaces/default/stream",
        exchange -> {
          exchange.getResponseHeaders().add("Content-Type", "application/json");
          // Length 0 means a chunked body that stays open until we stop writing.
          exchange.sendResponseHeaders(200, 0);
          try (OutputStream out = exchange.getResponseBody()) {
            while (true) {
              String doc = streamDocs.poll(30, TimeUnit.SECONDS);
              if (doc == null || END_STREAM.equals(doc)) {
                break;
              }
              out.write(("{\"result\":" + doc + "}\n").getBytes(StandardCharsets.UTF_8));
              out.flush();
            }
          } catch (InterruptedException | IOException e) {
            // The client went away or the server is stopping.
          }
        });
    // The stream handler blocks, so other requests need their own threads.
    server.setExecutor(Executors.newCachedThreadPool());
    server.start();
  }

  @AfterEach
  void stopServer() {
    streamDocs.offer(END_STREAM);
    server.stop(0);
  }

  private FliptClient.FliptClientBuilder builder() {
    return FliptClient.builder()
        .url("http://127.0.0.1:" + server.getAddress().getPort())
        .updateInterval(Duration.ofSeconds(1))
        .errorStrategy(ErrorStrategy.FALLBACK);
  }

  @Test
  void testListenerFiresOnChangeAndCanEvaluate() throws Exception {
    AtomicInteger calls = new AtomicInteger();
    AtomicBoolean evaluatedInListener = new AtomicBoolean();
    AtomicReference<FliptClient> clientRef = new AtomicReference<>();
    CountDownLatch fired = new CountDownLatch(1);

    try (FliptClient client =
        builder()
            .onStateUpdate(
                () -> {
                  calls.incrementAndGet();
                  // Evaluating from inside the listener must not deadlock.
                  evaluatedInListener.set(
                      clientRef
                          .get()
                          .evaluateBoolean("new_flag", "entity", new HashMap<>())
                          .isEnabled());
                  fired.countDown();
                })
            .build()) {
      clientRef.set(client);

      // Unchanged polls must not fire the listener.
      Thread.sleep(1500);
      Assertions.assertEquals(0, calls.get());

      body.set(WITH_FLAG);
      Assertions.assertTrue(fired.await(5, TimeUnit.SECONDS), "listener did not fire");
      Assertions.assertTrue(evaluatedInListener.get());

      // Further unchanged polls must stay quiet.
      Thread.sleep(2500);
      Assertions.assertEquals(1, calls.get());
    }
  }

  @Test
  void testListenerFiresOnStreamedChange() throws Exception {
    AtomicInteger calls = new AtomicInteger();
    CountDownLatch fired = new CountDownLatch(1);

    FliptClient client =
        builder()
            .fetchMode(FetchMode.STREAMING)
            .onStateUpdate(
                () -> {
                  calls.incrementAndGet();
                  fired.countDown();
                })
            .build();
    try {
      // Same state as the initial fetch: must not fire.
      streamDocs.put(EMPTY);
      Thread.sleep(500);
      Assertions.assertEquals(0, calls.get());

      // A real change followed by a resend of it: must fire exactly once.
      streamDocs.put(WITH_FLAG);
      streamDocs.put(WITH_FLAG);
      Assertions.assertTrue(fired.await(5, TimeUnit.SECONDS), "listener did not fire");
      Assertions.assertTrue(
          client.evaluateBoolean("new_flag", "entity", new HashMap<>()).isEnabled());

      Thread.sleep(500);
      Assertions.assertEquals(1, calls.get());
    } finally {
      // Closing with the stream still open must not hang.
      long start = System.nanoTime();
      client.close();
      Assertions.assertTrue(
          Duration.ofNanos(System.nanoTime() - start).compareTo(Duration.ofSeconds(5)) < 0,
          "close() took too long with an open stream");
    }

    // Nothing fires after close, even if the server keeps sending.
    streamDocs.put(EMPTY);
    Thread.sleep(500);
    Assertions.assertEquals(1, calls.get());
  }

  @Test
  void testThrowingListenerDoesNotBreakClient() throws Exception {
    CountDownLatch fired = new CountDownLatch(1);
    try (FliptClient client =
        builder()
            .onStateUpdate(
                () -> {
                  fired.countDown();
                  throw new RuntimeException("boom");
                })
            .build()) {
      body.set(WITH_FLAG);
      Assertions.assertTrue(fired.await(5, TimeUnit.SECONDS), "listener did not fire");
      Assertions.assertTrue(
          client.evaluateBoolean("new_flag", "entity", new HashMap<>()).isEnabled());
    }
  }

  @Test
  void testCloseFromInsideListener() throws Exception {
    AtomicInteger calls = new AtomicInteger();
    AtomicReference<FliptClient> clientRef = new AtomicReference<>();
    CountDownLatch closed = new CountDownLatch(1);

    FliptClient client =
        builder()
            .fetchMode(FetchMode.STREAMING)
            .onStateUpdate(
                () -> {
                  calls.incrementAndGet();
                  clientRef.get().close();
                  closed.countDown();
                })
            .build();
    clientRef.set(client);

    streamDocs.put(WITH_FLAG);
    Assertions.assertTrue(closed.await(5, TimeUnit.SECONDS), "close() from listener hung");

    // Further changes must not reach the listener.
    streamDocs.put(EMPTY);
    Thread.sleep(500);
    Assertions.assertEquals(1, calls.get());
    // A second close() from the test thread is a no-op.
    client.close();
  }

  @Test
  void testAddAndRemoveListenerWhileRunning() throws Exception {
    AtomicInteger first = new AtomicInteger();
    AtomicInteger second = new AtomicInteger();
    CountDownLatch firstFired = new CountDownLatch(1);
    CountDownLatch secondFired = new CountDownLatch(2);

    try (FliptClient client = builder().fetchMode(FetchMode.STREAMING).build()) {
      FliptClient.Subscription sub =
          client.addStateUpdateListener(
              () -> {
                first.incrementAndGet();
                firstFired.countDown();
              });
      client.addStateUpdateListener(
          () -> {
            second.incrementAndGet();
            secondFired.countDown();
          });

      streamDocs.put(WITH_FLAG);
      Assertions.assertTrue(firstFired.await(5, TimeUnit.SECONDS), "first did not fire");

      // After unsubscribing, only the remaining listener sees the next change.
      sub.close();
      sub.close();
      streamDocs.put(EMPTY);
      Assertions.assertTrue(secondFired.await(5, TimeUnit.SECONDS), "second did not fire twice");
      Assertions.assertEquals(1, first.get());
      Assertions.assertEquals(2, second.get());
    }
  }

  @Test
  void testAddListenerFromInsideListener() throws Exception {
    AtomicReference<FliptClient> clientRef = new AtomicReference<>();
    AtomicInteger outer = new AtomicInteger();
    CountDownLatch innerFired = new CountDownLatch(1);

    try (FliptClient client =
        builder()
            .fetchMode(FetchMode.STREAMING)
            .onStateUpdate(
                () -> {
                  if (outer.incrementAndGet() == 1) {
                    clientRef.get().addStateUpdateListener(innerFired::countDown);
                  }
                })
            .build()) {
      clientRef.set(client);

      streamDocs.put(WITH_FLAG);
      // The inner listener is added during this change; it only sees the next one.
      Assertions.assertTrue(waitFor(() -> outer.get() == 1));
      streamDocs.put(EMPTY);
      Assertions.assertTrue(innerFired.await(5, TimeUnit.SECONDS), "inner did not fire");
    }
  }

  @Test
  void testAddListenerAfterCloseThrows() {
    FliptClient client = builder().build();
    client.close();
    Assertions.assertThrows(
        IllegalStateException.class, () -> client.addStateUpdateListener(() -> {}));
  }

  private static boolean waitFor(java.util.function.BooleanSupplier cond)
      throws InterruptedException {
    long deadline = System.nanoTime() + TimeUnit.SECONDS.toNanos(5);
    while (System.nanoTime() < deadline) {
      if (cond.getAsBoolean()) {
        return true;
      }
      Thread.sleep(20);
    }
    return cond.getAsBoolean();
  }

  @Test
  void testCloseStopsListener() throws Exception {
    AtomicInteger calls = new AtomicInteger();
    FliptClient client = builder().onStateUpdate(calls::incrementAndGet).build();
    client.close();

    body.set(WITH_FLAG);
    Thread.sleep(2000);
    Assertions.assertEquals(0, calls.get());
  }
}
