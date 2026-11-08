@testable import FliptClient
import Foundation
import Network
import XCTest

private let empty = #"{"namespace":{"key":"default"},"flags":[]}"#
private let withFlag =
    #"{"namespace":{"key":"default"},"flags":[{"key":"new_flag","name":"New","type":"BOOLEAN_FLAG_TYPE","enabled":true}]}"#
private let barrier =
    #"{"namespace":{"key":"default"},"flags":[{"key":"barrier","name":"Barrier","type":"BOOLEAN_FLAG_TYPE","enabled":true}]}"#
private let waitSeconds: TimeInterval = 5

/// A minimal HTTP server for the snapshot and stream endpoints, so no Flipt instance is needed.
final class FakeFlipt {
    private let listener: NWListener
    private let queue = DispatchQueue(label: "fake-flipt")
    private let lock = NSLock()
    private var bodyValue = empty
    // Documents the stream endpoint sends, one per line, and the open stream to send them on.
    private var pendingDocs: [String] = []
    private var stream: NWConnection?

    init() throws {
        listener = try NWListener(using: .tcp, on: .any)
        let ready = DispatchSemaphore(value: 0)
        listener.stateUpdateHandler = { state in
            if case .ready = state {
                ready.signal()
            }
        }
        listener.newConnectionHandler = { [weak self] connection in
            self?.accept(connection)
        }
        listener.start(queue: queue)
        guard ready.wait(timeout: .now() + waitSeconds) == .success else {
            listener.cancel()
            throw URLError(.cannotConnectToHost)
        }
    }

    var url: String {
        "http://127.0.0.1:\(listener.port?.rawValue ?? 0)"
    }

    var body: String {
        get { lock.withLock { bodyValue } }
        set { lock.withLock { bodyValue = newValue } }
    }

    /// Sends a document on the open stream, or queues it for the next one.
    func send(_ doc: String) {
        queue.async { [self] in
            if let stream {
                write(doc, to: stream)
            } else {
                pendingDocs.append(doc)
            }
        }
    }

    func stop() {
        listener.cancel()
        queue.sync {
            stream?.cancel()
            stream = nil
        }
    }

    private func accept(_ connection: NWConnection) {
        connection.start(queue: queue)
        connection.receive(minimumIncompleteLength: 1, maximumLength: 65536) { [weak self] data, _, _, _ in
            guard let self, let data, let request = String(data: data, encoding: .utf8) else {
                connection.cancel()
                return
            }
            let path = request.split(separator: " ").dropFirst().first.map(String.init) ?? ""
            if path.hasPrefix("/internal/v1/evaluation/snapshot/namespace/default") {
                let body = body
                let response = "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\n"
                    + "Content-Length: \(body.utf8.count)\r\nConnection: close\r\n\r\n\(body)"
                connection.send(content: Data(response.utf8), completion: .contentProcessed { _ in
                    connection.cancel()
                })
            } else if path.hasPrefix("/client/v2/environments/default/namespaces/default/stream") {
                let header = "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nTransfer-Encoding: chunked\r\n\r\n"
                connection.send(content: Data(header.utf8), completion: .idempotent)
                // Only one stream at a time; it stays open until the server stops.
                stream = connection
                for doc in pendingDocs {
                    write(doc, to: connection)
                }
                pendingDocs.removeAll()
            } else {
                let response = "HTTP/1.1 404 Not Found\r\nContent-Length: 0\r\nConnection: close\r\n\r\n"
                connection.send(content: Data(response.utf8), completion: .contentProcessed { _ in
                    connection.cancel()
                })
            }
        }
    }

    private func write(_ doc: String, to connection: NWConnection) {
        let line = "{\"result\":\(doc)}\n"
        let chunk = String(line.utf8.count, radix: 16) + "\r\n" + line + "\r\n"
        connection.send(content: Data(chunk.utf8), completion: .idempotent)
    }
}

/// A thread-safe box for values a listener sets on another thread.
final class Box<T>: @unchecked Sendable {
    private let lock = NSLock()
    private var stored: T

    init(_ value: T) {
        stored = value
    }

    var value: T {
        get { lock.withLock { stored } }
        set { lock.withLock { stored = newValue } }
    }
}

/// A thread-safe counter for listener calls.
final class Counter: @unchecked Sendable {
    private let lock = NSLock()
    private var count = 0

    var value: Int {
        lock.withLock { count }
    }

    func increment() {
        lock.withLock { count += 1 }
    }
}

class StateUpdateListenerTests: XCTestCase {
    private var flipt: FakeFlipt?

    override func setUpWithError() throws {
        try super.setUpWithError()
        flipt = try FakeFlipt()
    }

    override func tearDown() {
        flipt?.stop()
        flipt = nil
        super.tearDown()
    }

    private func newClient(
        fetchMode: FetchMode = .polling,
        onStateUpdate: (@Sendable () -> Void)? = nil) throws -> FliptClient
    {
        try FliptClient(
            url: XCTUnwrap(flipt).url,
            updateInterval: .seconds(1),
            fetchMode: fetchMode,
            errorStrategy: .fallback,
            onStateUpdate: onStateUpdate)
    }

    func testListenerFiresOnChangeAndCanEvaluate() throws {
        let calls = Counter()
        let fired = expectation(description: "listener fired")
        fired.assertForOverFulfill = false
        let evaluated = Box<Bool?>(nil)
        let clientRef = WeakClient()
        let client = try newClient {
            calls.increment()
            // Evaluating from inside the listener must not deadlock.
            evaluated.value = try? clientRef.client?.evaluateBoolean(
                flagKey: "new_flag",
                entityID: "e",
                evalContext: [:]).enabled
            fired.fulfill()
        }
        clientRef.client = client
        defer { client.close() }

        // Unchanged polls must not fire the listener.
        Thread.sleep(forTimeInterval: 1.5)
        XCTAssertEqual(calls.value, 0)

        flipt?.body = withFlag
        wait(for: [fired], timeout: waitSeconds)
        XCTAssertEqual(evaluated.value, true)
    }

    func testListenerFiresOnStreamedChange() throws {
        let fired = expectation(description: "listener fired")
        fired.assertForOverFulfill = false
        flipt?.send(empty)
        let client = try newClient(fetchMode: .streaming) { fired.fulfill() }
        defer { client.close() }

        flipt?.send(withFlag)
        wait(for: [fired], timeout: waitSeconds)
    }

    func testCloseWaitsForRunningListener() throws {
        let started = DispatchSemaphore(value: 0)
        let release = DispatchSemaphore(value: 0)
        let finished = Counter()
        let client = try newClient {
            started.signal()
            _ = release.wait(timeout: .now() + waitSeconds)
            finished.increment()
        }
        defer {
            release.signal()
            client.close()
        }

        flipt?.body = withFlag
        XCTAssertEqual(started.wait(timeout: .now() + waitSeconds), .success, "listener did not start")

        let closed = DispatchSemaphore(value: 0)
        Thread.detachNewThread {
            client.close()
            closed.signal()
        }
        // close must still be waiting for the listener.
        XCTAssertEqual(closed.wait(timeout: .now() + 0.3), .timedOut)
        release.signal()
        XCTAssertEqual(closed.wait(timeout: .now() + waitSeconds), .success)
        XCTAssertEqual(finished.value, 1)
    }

    func testConcurrentCloseWaitsForRunningListener() throws {
        let started = DispatchSemaphore(value: 0)
        let release = DispatchSemaphore(value: 0)
        let client = try newClient {
            started.signal()
            _ = release.wait(timeout: .now() + waitSeconds)
        }
        defer {
            release.signal()
            client.close()
        }

        flipt?.body = withFlag
        XCTAssertEqual(started.wait(timeout: .now() + waitSeconds), .success, "listener did not start")

        // Both callers must wait for the running listener, not just the first.
        let closed = DispatchSemaphore(value: 0)
        for _ in 0 ..< 2 {
            Thread.detachNewThread {
                client.close()
                closed.signal()
            }
        }
        XCTAssertEqual(closed.wait(timeout: .now() + 0.3), .timedOut)
        release.signal()
        XCTAssertEqual(closed.wait(timeout: .now() + waitSeconds), .success)
        XCTAssertEqual(closed.wait(timeout: .now() + waitSeconds), .success)
    }

    func testCloseFromInsideListener() throws {
        let done = expectation(description: "listener closed the client")
        let laterCalls = Counter()
        let clientRef = WeakClient()
        let client = try newClient {
            clientRef.client?.close()
            done.fulfill()
        }
        clientRef.client = client
        defer { client.close() }

        // Registered after the closing listener, so it must never run.
        try client.addStateUpdateListener { laterCalls.increment() }
        flipt?.body = withFlag
        wait(for: [done], timeout: waitSeconds)
        Thread.sleep(forTimeInterval: 0.3)
        XCTAssertEqual(laterCalls.value, 0)
    }

    func testAddAndRemoveListenerWhileRunning() throws {
        let fired = expectation(description: "listener fired")
        fired.assertForOverFulfill = false
        let removedCalls = Counter()
        let client = try newClient()
        defer { client.close() }

        let subscription = try client.addStateUpdateListener { removedCalls.increment() }
        subscription.close()
        subscription.close() // closing twice is a no-op
        try client.addStateUpdateListener { fired.fulfill() }
        flipt?.body = withFlag
        wait(for: [fired], timeout: waitSeconds)
        XCTAssertEqual(removedCalls.value, 0)
    }

    func testAddListenerFromInsideListener() throws {
        let added = expectation(description: "listener added another")
        let addedFired = expectation(description: "added listener fired")
        addedFired.assertForOverFulfill = false
        let once = Counter()
        let clientRef = WeakClient()
        let client = try newClient {
            if once.value > 0 {
                return
            }
            once.increment()
            _ = try? clientRef.client?.addStateUpdateListener { addedFired.fulfill() }
            added.fulfill()
        }
        clientRef.client = client
        defer { client.close() }

        flipt?.body = withFlag
        wait(for: [added], timeout: waitSeconds)
        flipt?.body = barrier
        wait(for: [addedFired], timeout: waitSeconds)
    }

    func testAddListenerAfterCloseThrows() throws {
        // Start the update thread first, so the closed check is what throws.
        let client = try newClient {}
        client.close()
        client.close() // closing twice is a no-op
        XCTAssertThrowsError(try client.addStateUpdateListener {}) { error in
            guard case let FliptClient.ClientError.evaluationFailed(message) = error else {
                return XCTFail("unexpected error: \(error)")
            }
            XCTAssertEqual(message, "client is closed")
        }
    }

    func testCloseStopsListener() throws {
        let calls = Counter()
        let client = try newClient { calls.increment() }
        client.close()
        flipt?.body = withFlag
        Thread.sleep(forTimeInterval: 1.5)
        XCTAssertEqual(calls.value, 0)
    }

    func testDroppedClientDeinits() throws {
        // The update thread must not keep the client alive, so dropping the last reference closes it.
        weak var weakClient: FliptClient?
        try autoreleasepool {
            let client = try newClient {}
            weakClient = client
        }
        XCTAssertNil(weakClient)
    }
}

/// A weak reference a listener can read from another thread.
final class WeakClient: @unchecked Sendable {
    private let lock = NSLock()
    private weak var stored: FliptClient?

    var client: FliptClient? {
        get { lock.withLock { stored } }
        set { lock.withLock { stored = newValue } }
    }
}
