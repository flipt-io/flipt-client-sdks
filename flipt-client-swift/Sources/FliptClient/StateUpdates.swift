import FliptEngineFFI
import Foundation

/// A registered state update listener. Closing it unsubscribes; closing twice is a no-op.
public final class StateUpdateSubscription: @unchecked Sendable {
    private let lock = NSLock()
    private var unsubscribe: (@Sendable () -> Void)?

    init(_ unsubscribe: @escaping @Sendable () -> Void) {
        self.unsubscribe = unsubscribe
    }

    /// Removes the listener.
    public func close() {
        lock.lock()
        let unsubscribe = unsubscribe
        self.unsubscribe = nil
        lock.unlock()
        unsubscribe?()
    }
}

/// Owns the single thread that waits on the engine for flag state changes and fans out to the
/// client's listeners. Internal to FliptClient. The thread holds this object, not the client, so a
/// client that is no longer referenced can still deinit and close.
final class StateUpdates: @unchecked Sendable {
    static let closedMessage = "client is closed"

    private let engine: UnsafeMutableRawPointer?
    // Guards everything below. Never held while running listeners or joining the thread.
    private let lock = NSLock()
    private var closed = false
    private var listeners: [(id: Int, call: @Sendable () -> Void)] = []
    private var nextID = 0
    // Started on the first listener. It owns the native subscription and frees it on exit.
    private var thread: Thread?
    private var subscription: UnsafeMutableRawPointer?
    // Left by the thread when it exits, so every close caller can wait for it.
    private var finished: DispatchGroup?

    init(engine: UnsafeMutableRawPointer?) {
        self.engine = engine
    }

    func add(_ listener: @escaping @Sendable () -> Void) throws -> StateUpdateSubscription {
        lock.lock()
        defer { lock.unlock() }
        if closed {
            throw FliptClient.ClientError.evaluationFailed(message: StateUpdates.closedMessage)
        }
        try ensureThread()
        // An id per call, so adding the same closure twice yields two independent subscriptions.
        let id = nextID
        nextID += 1
        listeners.append((id: id, call: listener))
        return StateUpdateSubscription { [weak self] in self?.remove(id) }
    }

    /// Marks the client closed, wakes the thread and waits for a running listener to return, unless
    /// called from that listener. Every caller waits, but only the first returns true, so exactly one
    /// caller tears down the engine, and only after listeners have stopped.
    @discardableResult
    func close() -> Bool {
        // Released after the lock, since dropping a listener may deinit a client that calls close.
        var dropped: [(id: Int, call: @Sendable () -> Void)] = []
        lock.lock()
        let first = !closed
        closed = true
        let thread = thread
        let finished = finished
        if first, let subscription {
            // Wakes the thread if it is waiting. It still owns and frees the handle.
            close_state_subscription(subscription)
        }
        swap(&dropped, &listeners)
        lock.unlock()

        // Let a running listener finish before the engine it may be evaluating against is freed.
        if let thread, let finished, thread !== Thread.current {
            finished.wait()
        }
        return first
    }

    private var isClosed: Bool {
        lock.lock()
        defer { lock.unlock() }
        return closed
    }

    private func remove(_ id: Int) {
        lock.lock()
        // Released after the lock, since dropping a listener may deinit a client that calls close.
        let dropped = listeners.filter { $0.id == id }
        listeners.removeAll { $0.id == id }
        lock.unlock()
        _ = dropped
    }

    /// Called with the lock held. The thread runs until close, so adding or removing listeners
    /// never calls into the engine, including from inside a listener.
    private func ensureThread() throws {
        if thread != nil {
            return
        }
        guard let subscription = subscribe_state_updates(engine) else {
            throw FliptClient.ClientError.evaluationFailed(message: "Failed to subscribe to state updates")
        }
        let finished = DispatchGroup()
        finished.enter()
        let thread = Thread { [self] in
            run(subscription, finished: finished)
        }
        thread.name = "flipt-state-update"
        // The thread can't exit before these are set: its cleanup takes the lock we hold.
        self.subscription = subscription
        self.finished = finished
        self.thread = thread
        thread.start()
    }

    private func run(_ subscription: UnsafeMutableRawPointer, finished: DispatchGroup) {
        defer {
            // This thread is the only one that waits on the subscription, so it frees it. Under the
            // lock, so close never wakes a handle that is already freed.
            lock.lock()
            destroy_state_subscription(subscription)
            // If this thread died early, the next listener starts a new one.
            if thread === Thread.current {
                self.subscription = nil
                self.finished = nil
                thread = nil
            }
            lock.unlock()
            finished.leave()
        }

        while !isClosed {
            guard let response = wait_for_state_update(subscription) else {
                log("Failed to wait for state update: no response from engine")
                return
            }
            let responseString = String(cString: response)
            destroy_string(UnsafeMutablePointer(mutating: response))

            guard let result = try? JSONDecoder().decode(Result<String>.self, from: Data(responseString.utf8)),
                  result.status == "success"
            else {
                log("Failed to wait for state update: \(responseString)")
                return
            }
            if result.result != "changed" {
                return
            }

            lock.lock()
            let current = listeners
            lock.unlock()
            for listener in current {
                if isClosed {
                    return
                }
                listener.call()
            }
        }
    }

    private func log(_ message: String) {
        FileHandle.standardError.write(Data("[flipt] \(message)\n".utf8))
    }
}
