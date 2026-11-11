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
  private static final String BARRIER =
      "{\"namespace\":{\"key\":\"default\"},\"flags\":[{\"key\":\"barrier\",\"name\":\"Barrier\","
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
            .onStateUpdate(() -> {
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
    // One entry per finished call: the flag keys the listener saw.
    BlockingQueue<String> seen = new LinkedBlockingQueue<>();
    AtomicReference<FliptClient> clientRef = new AtomicReference<>();

    FliptClient client =
        builder()
            .fetchMode(FetchMode.STREAMING)
            .onStateUpdate(() -> {
              calls.incrementAndGet();
              StringBuilder keys = new StringBuilder();
              clientRef.get().listFlags().forEach(f -> keys.append(f.getKey()));
              seen.offer(keys.toString());
            })
            .build();
    clientRef.set(client);
    try {
      // Same state as the initial fetch: must not fire.
      streamDocs.put(EMPTY);
      Thread.sleep(500);
      Assertions.assertEquals(0, calls.get());

      streamDocs.put(WITH_FLAG);
      Assertions.assertEquals("new_flag", seen.poll(5, TimeUnit.SECONDS), "first call");
      Assertions.assertTrue(
          client.evaluateBoolean("new_flag", "entity", new HashMap<>()).isEnabled());

      // The first call has returned. Now a resend of the same state, then a different barrier
      // document. The next call must already see the barrier: a call fired by the resend alone
      // would still see new_flag. (A spurious change that coalesces with the barrier can't be told
      // apart here; the engine tests cover that.)
      streamDocs.put(WITH_FLAG);
      streamDocs.put(BARRIER);
      Assertions.assertEquals("barrier", seen.poll(5, TimeUnit.SECONDS), "second call");
      Assertions.assertEquals(2, calls.get());
    } finally {
      // Closing with the stream still open must not hang.
      long start = System.nanoTime();
      client.close();
      Assertions.assertTrue(
          Duration.ofNanos(System.nanoTime() - start).compareTo(Duration.ofSeconds(5)) < 0,
          "close() took too long with an open stream");
    }
  }

  @Test
  void testCloseWaitsForRunningListener() throws Exception {
    AtomicInteger calls = new AtomicInteger();
    AtomicBoolean evaluatedAfterRelease = new AtomicBoolean();
    AtomicReference<FliptClient> clientRef = new AtomicReference<>();
    CountDownLatch entered = new CountDownLatch(1);
    CountDownLatch release = new CountDownLatch(1);

    FliptClient client =
        builder()
            .fetchMode(FetchMode.STREAMING)
            .onStateUpdate(() -> {
              calls.incrementAndGet();
              entered.countDown();
              try {
                release.await(10, TimeUnit.SECONDS);
              } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
              }
              // close() has started by now; the engine must still be alive.
              clientRef.get().listFlags();
              evaluatedAfterRelease.set(true);
            })
            .build();
    clientRef.set(client);

    streamDocs.put(WITH_FLAG);
    Assertions.assertTrue(entered.await(5, TimeUnit.SECONDS), "listener did not fire");
    // A change that lands while the listener runs is pending when close() starts.
    streamDocs.put(EMPTY);
    Thread.sleep(300);

    CountDownLatch closeReturned = new CountDownLatch(1);
    Thread closer =
        new Thread(() -> {
          client.close();
          closeReturned.countDown();
        });
    closer.start();
    // Once the closer is blocked in join(), closed is set and the pending change can't run.
    long deadline = System.nanoTime() + TimeUnit.SECONDS.toNanos(5);
    while (closer.getState() != Thread.State.WAITING
        && closer.getState() != Thread.State.TIMED_WAITING
        && System.nanoTime() < deadline) {
      Thread.sleep(10);
    }
    Assertions.assertFalse(
        closeReturned.await(300, TimeUnit.MILLISECONDS), "close() did not wait for listener");

    release.countDown();
    Assertions.assertTrue(closeReturned.await(5, TimeUnit.SECONDS), "close() hung");
    closer.join();
    Assertions.assertTrue(evaluatedAfterRelease.get());
    // The pending change is dropped once close() has started.
    Assertions.assertEquals(1, calls.get());
  }

  @Test
  void testThrowingListenerDoesNotBreakClient() throws Exception {
    CountDownLatch fired = new CountDownLatch(1);
    try (FliptClient client =
        builder()
            .onStateUpdate(() -> {
              throw new RuntimeException("boom");
            })
            .build()) {
      // A listener after the throwing one still runs.
      client.addStateUpdateListener(fired::countDown);
      body.set(WITH_FLAG);
      Assertions.assertTrue(fired.await(5, TimeUnit.SECONDS), "second listener did not fire");
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
            .onStateUpdate(() -> {
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
          client.addStateUpdateListener(() -> {
            first.incrementAndGet();
            firstFired.countDown();
          });
      client.addStateUpdateListener(() -> {
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
            .onStateUpdate(() -> {
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
