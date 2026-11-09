package io.flipt.client

import androidx.test.ext.junit.runners.AndroidJUnit4
import io.flipt.client.models.ErrorStrategy
import io.flipt.client.models.FetchMode
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import java.net.InetAddress
import java.net.ServerSocket
import java.net.Socket
import java.util.concurrent.CountDownLatch
import java.util.concurrent.LinkedBlockingQueue
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicBoolean
import java.util.concurrent.atomic.AtomicInteger
import java.util.concurrent.atomic.AtomicReference
import kotlin.concurrent.thread
import kotlin.time.Duration.Companion.seconds

/**
 * Exercises state update listeners against a tiny local HTTP server, so no Flipt instance is
 * needed. The snapshot endpoint returns whatever [body] holds; the stream endpoint sends the
 * documents pushed to [streamDocs] as a chunked response.
 */
@RunWith(AndroidJUnit4::class)
class StateUpdateListenerTest {
    private val body = AtomicReference(EMPTY)

    // Snapshot responses written so far. Tests wait on this instead of sleeping, so they know a
    // poll has really happened.
    private val requests = AtomicInteger()

    // Documents the stream endpoint sends, as the test pushes them.
    private val streamDocs = LinkedBlockingQueue<String>()
    private val streamsOpened = AtomicInteger()

    private lateinit var server: ServerSocket
    private val running = AtomicBoolean(true)

    @Before
    fun startServer() {
        server = ServerSocket(0, 50, InetAddress.getByName("127.0.0.1"))
        thread(isDaemon = true, name = "test-http-server") {
            while (running.get()) {
                val socket =
                    try {
                        server.accept()
                    } catch (e: Exception) {
                        break
                    }
                thread(isDaemon = true) { serve(socket) }
            }
        }
    }

    @After
    fun stopServer() {
        running.set(false)
        streamDocs.offer(END_STREAM)
        server.close()
    }

    private fun serve(socket: Socket) {
        socket.use {
            val reader = it.getInputStream().bufferedReader()
            val requestLine = reader.readLine() ?: return
            // Drain the headers.
            while (true) {
                val line = reader.readLine() ?: return
                if (line.isEmpty()) break
            }
            if (requestLine.contains("/stream")) {
                serveStream(it)
            } else {
                serveSnapshot(it)
            }
        }
    }

    // Answers with the current snapshot and closes the connection.
    private fun serveSnapshot(socket: Socket) {
        val bytes = body.get().toByteArray(Charsets.UTF_8)
        val out = socket.getOutputStream()
        out.write(
            (
                "HTTP/1.1 200 OK\r\n" +
                    "Content-Type: application/json\r\n" +
                    "Content-Length: ${bytes.size}\r\n" +
                    "Connection: close\r\n\r\n"
            ).toByteArray(Charsets.UTF_8),
        )
        out.write(bytes)
        out.flush()
        requests.incrementAndGet()
    }

    // Keeps a chunked response open and sends each pushed document as one line.
    private fun serveStream(socket: Socket) {
        val out = socket.getOutputStream()
        try {
            out.write(
                (
                    "HTTP/1.1 200 OK\r\n" +
                        "Content-Type: application/json\r\n" +
                        "Transfer-Encoding: chunked\r\n\r\n"
                ).toByteArray(Charsets.UTF_8),
            )
            out.flush()
            streamsOpened.incrementAndGet()
            while (true) {
                val doc = streamDocs.poll(30, TimeUnit.SECONDS)
                if (doc == null || doc == END_STREAM) break
                val chunk = "{\"result\":$doc}\n".toByteArray(Charsets.UTF_8)
                out.write("${Integer.toHexString(chunk.size)}\r\n".toByteArray(Charsets.UTF_8))
                out.write(chunk)
                out.write("\r\n".toByteArray(Charsets.UTF_8))
                out.flush()
            }
            out.write("0\r\n\r\n".toByteArray(Charsets.UTF_8))
            out.flush()
        } catch (e: Exception) {
            // The client went away or the server is stopping.
        }
    }

    /**
     * Waits until [count] more snapshot responses have been written. Waiting for two after
     * registering listeners guarantees the client's initial fetch is done and at least one poll
     * ran with the listeners subscribed, so a later change can only arrive through a poll.
     */
    private fun awaitPolls(count: Int) {
        val target = requests.get() + count
        awaitTrue("server saw no poll") { requests.get() >= target }
    }

    private fun awaitTrue(
        message: String,
        condition: () -> Boolean,
    ) {
        val deadline = System.nanoTime() + TimeUnit.SECONDS.toNanos(10)
        while (!condition()) {
            check(System.nanoTime() < deadline) { message }
            Thread.sleep(20)
        }
    }

    private fun liveUpdateThreads(): Set<Thread> =
        Thread
            .getAllStackTraces()
            .keys
            .filter { it.name == "flipt-state-update" && it.isAlive }
            .toSet()

    private fun builder(): FliptClient.FliptClientBuilder =
        FliptClient
            .builder()
            .url("http://127.0.0.1:${server.localPort}")
            .updateInterval(1.seconds)
            .errorStrategy(ErrorStrategy.FALLBACK)

    @Test
    fun testListenerFiresOnChangeAndCanEvaluate() {
        val calls = AtomicInteger()
        val evaluatedInListener = AtomicBoolean()
        val clientRef = AtomicReference<FliptClient>()
        val fired = CountDownLatch(1)

        val client =
            builder()
                .onStateUpdate {
                    calls.incrementAndGet()
                    // Evaluating from inside the listener must not deadlock.
                    evaluatedInListener.set(
                        clientRef.get().evaluateBoolean("new_flag", "entity", emptyMap()).enabled,
                    )
                    fired.countDown()
                }.build()
        clientRef.set(client)
        try {
            // Unchanged polls must not call the listener.
            awaitPolls(2)
            assertEquals(0, calls.get())

            body.set(WITH_FLAG)
            assertTrue("listener did not fire", fired.await(5, TimeUnit.SECONDS))
            assertTrue(evaluatedInListener.get())

            // Further unchanged polls must stay quiet.
            awaitPolls(2)
            assertEquals(1, calls.get())
        } finally {
            client.close()
        }
    }

    @Test
    fun testListenerFiresOnStreamedChange() {
        val calls = AtomicInteger()
        // The flag keys each call sees, in order.
        val seen = LinkedBlockingQueue<String>()
        val clientRef = AtomicReference<FliptClient>()

        val client =
            builder()
                .fetchMode(FetchMode.STREAMING)
                .onStateUpdate {
                    calls.incrementAndGet()
                    seen.put(
                        clientRef
                            .get()
                            .listFlags()
                            .joinToString(",") { it.key },
                    )
                }.build()
        clientRef.set(client)
        try {
            awaitTrue("stream never opened") { streamsOpened.get() >= 1 }

            // Same state as the initial fetch: no call.
            streamDocs.put(EMPTY)
            streamDocs.put(WITH_FLAG)
            assertEquals("new_flag", seen.poll(5, TimeUnit.SECONDS))

            // A resent identical document is not a change. The barrier after it proves the
            // resend was processed before we count.
            streamDocs.put(WITH_FLAG)
            streamDocs.put(BARRIER)
            assertEquals("barrier", seen.poll(5, TimeUnit.SECONDS))
            assertEquals(2, calls.get())
        } finally {
            // close() must not hang on the open stream.
            val closed = CountDownLatch(1)
            thread {
                client.close()
                closed.countDown()
            }
            assertTrue("close() hung", closed.await(5, TimeUnit.SECONDS))
        }
    }

    @Test
    fun testAddAndRemoveListenerWhileRunning() {
        val first = AtomicInteger()
        val second = AtomicInteger()
        val firstFired = CountDownLatch(1)
        val secondFired = CountDownLatch(2)

        val client = builder().build()
        try {
            val sub =
                client.addStateUpdateListener {
                    first.incrementAndGet()
                    firstFired.countDown()
                }
            client.addStateUpdateListener {
                second.incrementAndGet()
                secondFired.countDown()
            }
            awaitPolls(2)

            body.set(WITH_FLAG)
            assertTrue("first did not fire", firstFired.await(5, TimeUnit.SECONDS))

            // After unsubscribing, only the remaining listener sees the next change.
            sub.close()
            sub.close()
            body.set(EMPTY)
            assertTrue("second did not fire twice", secondFired.await(5, TimeUnit.SECONDS))
            assertEquals(1, first.get())
            assertEquals(2, second.get())
        } finally {
            client.close()
        }
    }

    @Test
    fun testAddListenerFromInsideListener() {
        val clientRef = AtomicReference<FliptClient>()
        val outerCalls = AtomicInteger()
        val outerFired = CountDownLatch(1)
        val innerFired = CountDownLatch(1)

        val client =
            builder()
                .onStateUpdate {
                    if (outerCalls.incrementAndGet() == 1) {
                        // Adding a listener from inside one must not deadlock.
                        clientRef.get().addStateUpdateListener { innerFired.countDown() }
                    }
                    outerFired.countDown()
                }.build()
        clientRef.set(client)
        try {
            awaitPolls(2)
            body.set(WITH_FLAG)
            assertTrue("outer did not fire", outerFired.await(5, TimeUnit.SECONDS))

            // The listener added during the first call hears about the next change.
            body.set(EMPTY)
            assertTrue("inner did not fire", innerFired.await(5, TimeUnit.SECONDS))
        } finally {
            client.close()
        }
    }

    @Test
    fun testThrowingListenerDoesNotStopOthers() {
        val fired = CountDownLatch(1)
        val client =
            builder()
                .onStateUpdate { throw RuntimeException("boom") }
                .build()
        try {
            // A listener added after the throwing one still runs.
            client.addStateUpdateListener { fired.countDown() }
            awaitPolls(2)
            body.set(WITH_FLAG)
            assertTrue("second listener did not fire", fired.await(5, TimeUnit.SECONDS))
            assertTrue(client.evaluateBoolean("new_flag", "entity", emptyMap()).enabled)
        } finally {
            client.close()
        }
    }

    @Test
    fun testCloseWaitsForRunningListener() {
        val calls = AtomicInteger()
        val evaluatedAfterRelease = AtomicBoolean()
        val clientRef = AtomicReference<FliptClient>()
        val entered = CountDownLatch(1)
        val release = CountDownLatch(1)

        val client =
            builder()
                .onStateUpdate {
                    calls.incrementAndGet()
                    entered.countDown()
                    release.await(10, TimeUnit.SECONDS)
                    // close() has started by now; the engine must still be alive.
                    clientRef.get().listFlags()
                    evaluatedAfterRelease.set(true)
                }.build()
        clientRef.set(client)
        try {
            awaitPolls(2)

            body.set(WITH_FLAG)
            assertTrue("listener did not fire", entered.await(5, TimeUnit.SECONDS))
            // Apply another change while the listener runs, so one is pending when close() starts.
            body.set(EMPTY)
            awaitTrue("pending change was not applied") { client.listFlags().isEmpty() }

            val closeReturned = CountDownLatch(1)
            val closer =
                thread {
                    client.close()
                    closeReturned.countDown()
                }
            // Once the closer is blocked in join(), closed is set and the pending change can't run.
            val deadline = System.nanoTime() + TimeUnit.SECONDS.toNanos(5)
            while (closer.state != Thread.State.WAITING &&
                closer.state != Thread.State.TIMED_WAITING &&
                System.nanoTime() < deadline
            ) {
                Thread.sleep(10)
            }
            assertFalse("close() did not wait for listener", closeReturned.await(300, TimeUnit.MILLISECONDS))

            release.countDown()
            assertTrue("close() hung", closeReturned.await(5, TimeUnit.SECONDS))
            closer.join()
            assertTrue(evaluatedAfterRelease.get())
            // The pending change is dropped once close() has started.
            assertEquals(1, calls.get())
        } finally {
            release.countDown()
            client.close()
        }
    }

    @Test
    fun testCloseFromInsideListener() {
        val before = liveUpdateThreads()
        val calls = AtomicInteger()
        val later = AtomicInteger()
        val clientRef = AtomicReference<FliptClient>()
        val closed = CountDownLatch(1)

        val client =
            builder()
                .onStateUpdate {
                    calls.incrementAndGet()
                    clientRef.get().close()
                    closed.countDown()
                }.build()
        clientRef.set(client)
        // Runs after the closing listener on the same thread, so it must be skipped.
        client.addStateUpdateListener { later.incrementAndGet() }
        try {
            awaitPolls(2)

            body.set(WITH_FLAG)
            assertTrue("close() from listener hung", closed.await(5, TimeUnit.SECONDS))
            awaitTrue("update thread did not exit") { (liveUpdateThreads() - before).isEmpty() }
            assertEquals(1, calls.get())
            assertEquals(0, later.get())
        } finally {
            client.close()
        }
    }

    @Test
    fun testCloseStopsListener() {
        val before = liveUpdateThreads()
        val calls = AtomicInteger()
        val client = builder().onStateUpdate { calls.incrementAndGet() }.build()
        awaitTrue("update thread did not start") { (liveUpdateThreads() - before).isNotEmpty() }

        client.close()
        // A second close() is a no-op and must not destroy the engine again.
        client.close()

        // The update thread is gone, so nothing can call the listener any more.
        awaitTrue("update thread did not exit") { (liveUpdateThreads() - before).isEmpty() }
        body.set(WITH_FLAG)
        assertEquals(0, calls.get())
    }

    @Test(expected = IllegalStateException::class)
    fun testAddListenerAfterCloseThrows() {
        val client = builder().build()
        client.close()
        client.addStateUpdateListener {}
    }

    companion object {
        private const val END_STREAM = "<end>"
        private const val EMPTY = """{"namespace":{"key":"default"},"flags":[]}"""
        private const val WITH_FLAG =
            """{"namespace":{"key":"default"},"flags":[{"key":"new_flag","name":"New","type":"BOOLEAN_FLAG_TYPE","enabled":true}]}"""
        private const val BARRIER =
            """{"namespace":{"key":"default"},"flags":[{"key":"barrier","name":"Barrier","type":"BOOLEAN_FLAG_TYPE","enabled":true}]}"""
    }
}
