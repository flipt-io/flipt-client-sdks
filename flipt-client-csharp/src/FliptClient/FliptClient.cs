using System;
using System.Runtime.InteropServices;
using System.Text.Json;
using System.Text.Json.Serialization;
using FliptClient.Models;

namespace FliptClient
{
    /// <summary>
    /// Main client for interacting with the Flipt feature flag engine.
    /// </summary>
    public class FliptClient : IDisposable
    {
        // Guards _closed, _stateUpdateThread and _stateSubscription. Never held while running
        // listeners, joining the update thread or destroying the engine.
        private readonly object _stateUpdateLock = new object();
        private IntPtr _engine;
        private volatile bool _closed;

        // Replaced, never mutated, so the update thread can iterate a snapshot without the lock.
        private volatile Action[] _stateUpdateListeners = Array.Empty<Action>();

        // Started on the first listener. It owns the native subscription and frees it on exit.
        private Thread? _stateUpdateThread;
        private IntPtr _stateSubscription;

        /// <summary>
        /// Initializes a new instance of the <see cref="FliptClient"/> class.
        /// </summary>
        /// <param name="options">Client options, including configuration.</param>
        /// <exception cref="ValidationException">Thrown if options are invalid.</exception>
        public FliptClient(ClientOptions options)
        {
            if (options == null)
            {
                throw new ValidationException("ClientOptions cannot be null");
            }

            string optsJson = JsonSerializer.Serialize(options);
            _engine = NativeMethods.InitializeEngine(optsJson);

            if (options.OnStateUpdate != null)
            {
                try
                {
                    AddStateUpdateListener(options.OnStateUpdate);
                }
                catch
                {
                    // The caller never gets a client to dispose, so tear the engine down here.
                    Dispose();
                    throw;
                }
            }
        }

        /// <summary>
        /// Adds a listener that is called after each change to flag state. Listeners can be added at any
        /// time. Read the flags you care about once after adding a listener: it is called for changes
        /// applied after it was added, and may occasionally see one extra call.
        /// </summary>
        /// <remarks>
        /// Listeners run in the order they were added, on a dedicated background thread named
        /// <c>flipt-state-update</c>. Updates that arrive while they run are coalesced into a single
        /// follow-up call. A listener may evaluate flags, add or remove listeners, or dispose the client.
        /// <see cref="Dispose"/> from another thread waits for a running listener to return, so a listener
        /// must not block on anything held by the thread that disposes the client. An exception thrown by
        /// one listener is written to standard error and does not stop the others. A listener removed
        /// while a notification is running may still be called once for that notification. The client
        /// must be disposed to stop its update thread.
        /// </remarks>
        /// <param name="listener">The listener to call.</param>
        /// <returns>A subscription that removes the listener when disposed.</returns>
        /// <exception cref="ValidationException">Thrown if the listener is null.</exception>
        /// <exception cref="ObjectDisposedException">Thrown if the client is disposed.</exception>
        /// <exception cref="EvaluationException">Thrown if the engine cannot subscribe to updates.</exception>
        public IDisposable AddStateUpdateListener(Action listener)
        {
            if (listener == null)
            {
                throw new ValidationException("listener cannot be null");
            }

            // A distinct wrapper per call, so adding the same delegate twice yields two independent
            // subscriptions.
            Action entry = () => listener();
            lock (_stateUpdateLock)
            {
                if (_closed)
                {
                    throw new ObjectDisposedException(nameof(FliptClient));
                }

                EnsureStateUpdateThread();
                var listeners = _stateUpdateListeners;
                var updated = new Action[listeners.Length + 1];
                listeners.CopyTo(updated, 0);
                updated[listeners.Length] = entry;
                _stateUpdateListeners = updated;
            }

            return new Subscription(() => RemoveStateUpdateListener(entry));
        }

        /// <summary>
        /// Evaluates a variant flag for the given entity and context.
        /// </summary>
        /// <returns></returns>
        public VariantEvaluationResponse? EvaluateVariant(string flagKey, string entityId, Dictionary<string, string> context)
        {
            if (string.IsNullOrWhiteSpace(flagKey))
            {
                throw new ValidationException("flagKey cannot be empty or null");
            }

            if (string.IsNullOrWhiteSpace(entityId))
            {
                throw new ValidationException("entityId cannot be empty or null");
            }

            if (context == null)
            {
                context = new Dictionary<string, string>();
            }

            var request = new EvaluationRequest
            {
                FlagKey = flagKey,
                EntityId = entityId,
                Context = context
            };
            string requestJson = JsonSerializer.Serialize(request);
            IntPtr resultPtr = NativeMethods.EvaluateVariant(_engine, requestJson);
            string resultJson = Marshal.PtrToStringAnsi(resultPtr) ?? throw new FliptException("Failed to get result from native code");
            NativeMethods.DestroyString(resultPtr);
            var result = JsonSerializer.Deserialize<VariantResult>(resultJson) ?? throw new EvaluationException("Failed to deserialize response");
            if (result.Status != "success")
            {
                throw new EvaluationException(result.ErrorMessage ?? "Unknown error");
            }

            return result.Response;
        }

        /// <summary>
        /// Evaluates a boolean flag for the given entity and context.
        /// </summary>
        /// <returns></returns>
        public BooleanEvaluationResponse? EvaluateBoolean(string flagKey, string entityId, Dictionary<string, string> context)
        {
            if (string.IsNullOrWhiteSpace(flagKey))
            {
                throw new ValidationException("flagKey cannot be empty or null");
            }

            if (string.IsNullOrWhiteSpace(entityId))
            {
                throw new ValidationException("entityId cannot be empty or null");
            }

            if (context == null)
            {
                context = new Dictionary<string, string>();
            }

            var request = new EvaluationRequest
            {
                FlagKey = flagKey,
                EntityId = entityId,
                Context = context
            };
            string requestJson = JsonSerializer.Serialize(request);
            IntPtr resultPtr = NativeMethods.EvaluateBoolean(_engine, requestJson);
            string resultJson = Marshal.PtrToStringAnsi(resultPtr) ?? throw new FliptException("Failed to get result from native code");
            NativeMethods.DestroyString(resultPtr);
            var result = JsonSerializer.Deserialize<BooleanResult>(resultJson) ?? throw new EvaluationException("Failed to deserialize response");
            if (result.Status != "success")
            {
                throw new EvaluationException(result.ErrorMessage ?? "Unknown error");
            }

            return result.Response;
        }

        /// <summary>
        /// Evaluates a batch of flag requests.
        /// </summary>
        /// <returns></returns>
        public BatchEvaluationResponse? EvaluateBatch(List<EvaluationRequest> requests)
        {
            if (requests == null || requests.Count == 0)
            {
                throw new ValidationException("requests cannot be empty or null");
            }

            string requestJson = JsonSerializer.Serialize(requests);
            IntPtr resultPtr = NativeMethods.EvaluateBatch(_engine, requestJson);
            string resultJson = Marshal.PtrToStringAnsi(resultPtr) ?? throw new FliptException("Failed to get result from native code");
            NativeMethods.DestroyString(resultPtr);
            var result = JsonSerializer.Deserialize<BatchResult>(resultJson) ?? throw new EvaluationException("Failed to deserialize response");
            if (result.Status != "success")
            {
                throw new EvaluationException(result.ErrorMessage ?? "Unknown error");
            }

            return result.Response;
        }

        /// <summary>
        /// Lists all flags in the current namespace.
        /// </summary>
        /// <returns></returns>
        public Flag[]? ListFlags()
        {
            IntPtr resultPtr = NativeMethods.ListFlags(_engine);
            string resultJson = Marshal.PtrToStringAnsi(resultPtr) ?? throw new FliptException("Failed to get result from native code");
            NativeMethods.DestroyString(resultPtr);
            var result = JsonSerializer.Deserialize<ListFlagsResult>(resultJson) ?? throw new EvaluationException("Failed to deserialize response");
            if (result.Status != "success")
            {
                throw new EvaluationException(result.ErrorMessage ?? "Unknown error");
            }

            return result.Response;
        }

        /// <summary>
        /// Gets the snapshot for the client.
        /// </summary>
        /// <returns></returns>
        public string? GetSnapshot()
        {
            IntPtr resultPtr = NativeMethods.GetSnapshot(_engine);
            string resultStr = Marshal.PtrToStringAnsi(resultPtr) ?? throw new FliptException("Failed to get result from native code");
            NativeMethods.DestroyString(resultPtr);
            return resultStr;
        }

        /// <summary>
        /// Disposes the client and releases native resources. When called from a thread other than the
        /// state update thread, waits for a running listener to return first. Disposing twice is a no-op.
        /// </summary>
        public void Dispose()
        {
            Thread? thread;
            lock (_stateUpdateLock)
            {
                if (_closed)
                {
                    return;
                }

                _closed = true;
                thread = _stateUpdateThread;
                if (_stateSubscription != IntPtr.Zero)
                {
                    // Wakes the update thread if it is waiting. It still owns and frees the handle.
                    NativeMethods.CloseStateSubscription(_stateSubscription);
                }
            }

            if (thread != null && !ReferenceEquals(thread, Thread.CurrentThread))
            {
                // Let a running listener finish before the engine it may be evaluating against is freed.
                thread.Join();
            }

            IntPtr engine = _engine;
            _engine = IntPtr.Zero;
            if (engine != IntPtr.Zero)
            {
                NativeMethods.DestroyEngine(engine);
            }

            _stateUpdateListeners = Array.Empty<Action>();
        }

        private void RemoveStateUpdateListener(Action entry)
        {
            lock (_stateUpdateLock)
            {
                _stateUpdateListeners = Array.FindAll(_stateUpdateListeners, existing => !ReferenceEquals(existing, entry));
            }
        }

        // Called with _stateUpdateLock held. The thread runs until Dispose, so adding or removing
        // listeners never calls into the engine, including from inside a listener.
        private void EnsureStateUpdateThread()
        {
            if (_stateUpdateThread != null)
            {
                return;
            }

            IntPtr subscription = NativeMethods.SubscribeStateUpdates(_engine);
            if (subscription == IntPtr.Zero)
            {
                throw new EvaluationException("Failed to subscribe to state updates");
            }

            // Never keep the process alive just to wait for flag changes.
            var thread = new Thread(() => RunStateUpdateLoop(subscription))
            {
                IsBackground = true,
                Name = "flipt-state-update",
            };
            try
            {
                thread.Start();
            }
            catch
            {
                NativeMethods.DestroyStateSubscription(subscription);
                throw;
            }

            // The thread can't exit before these are set: its cleanup takes the lock we hold.
            _stateSubscription = subscription;
            _stateUpdateThread = thread;
        }

        private void RunStateUpdateLoop(IntPtr subscription)
        {
            try
            {
                while (!_closed)
                {
                    IntPtr resultPtr = NativeMethods.WaitForStateUpdate(subscription);
                    string? resultJson = Marshal.PtrToStringUTF8(resultPtr);
                    if (resultPtr != IntPtr.Zero)
                    {
                        NativeMethods.DestroyString(resultPtr);
                    }

                    var result = resultJson == null ? null : JsonSerializer.Deserialize<StateUpdateResult>(resultJson);
                    if (result == null || result.Status != "success")
                    {
                        // Existing listeners stay silent until the next AddStateUpdateListener starts a
                        // new thread.
                        Console.Error.WriteLine($"[flipt] Failed to wait for state update: {result?.ErrorMessage ?? "Unknown error"}");
                        return;
                    }

                    if (result.Response != "changed")
                    {
                        return;
                    }

                    foreach (var listener in _stateUpdateListeners)
                    {
                        if (_closed)
                        {
                            return;
                        }

                        try
                        {
                            listener();
                        }
                        catch (Exception e)
                        {
                            Console.Error.WriteLine($"[flipt] State update listener threw: {e}");
                        }
                    }
                }
            }
            catch (Exception e)
            {
                Console.Error.WriteLine($"[flipt] State update thread failed: {e}");
            }
            finally
            {
                // This thread is the only one that waits on the subscription, so it frees it. Under the
                // lock, so Dispose never wakes a handle that is already freed.
                lock (_stateUpdateLock)
                {
                    _stateSubscription = IntPtr.Zero;
                    NativeMethods.DestroyStateSubscription(subscription);

                    // If this thread died early, the next listener starts a new one.
                    if (ReferenceEquals(_stateUpdateThread, Thread.CurrentThread))
                    {
                        _stateUpdateThread = null;
                    }
                }
            }
        }

        private sealed class Subscription : IDisposable
        {
            private Action? _unsubscribe;

            public Subscription(Action unsubscribe)
            {
                _unsubscribe = unsubscribe;
            }

            public void Dispose()
            {
                Interlocked.Exchange(ref _unsubscribe, null)?.Invoke();
            }
        }

        private sealed class StateUpdateResult
        {
            [JsonPropertyName("status")]
            public string? Status { get; set; }

            [JsonPropertyName("result")]
            public string? Response { get; set; }

            [JsonPropertyName("error_message")]
            public string? ErrorMessage { get; set; }
        }
    }

    /// <summary>
    /// Base exception for Flipt client errors.
    /// </summary>
    public class FliptException : Exception
    {
        public FliptException(string message)
            : base(message)
        {
        }
    }

    /// <summary>
    /// Exception for validation errors.
    /// </summary>
    public class ValidationException : FliptException
    {
        public ValidationException(string message)
            : base(message)
        {
        }
    }

    /// <summary>
    /// Exception for evaluation errors.
    /// </summary>
    public class EvaluationException : FliptException
    {
        public EvaluationException(string message)
            : base(message)
        {
        }
    }
}