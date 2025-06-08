using System;
using System.Runtime.InteropServices;
using System.Text.Json;
using FliptClient.Models;

namespace FliptClient
{
    /// <summary>
    /// Main client for interacting with the Flipt feature flag engine.
    /// </summary>
    public class FliptClient : IDisposable
    {
        private IntPtr _engine;

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
        /// Disposes the client and releases native resources.
        /// </summary>
        public void Dispose()
        {
            if (_engine != IntPtr.Zero)
            {
                NativeMethods.DestroyEngine(_engine);
                _engine = IntPtr.Zero;
            }
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