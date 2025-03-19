using System;
using System.Runtime.InteropServices;
using System.Text.Json;
using FliptClient.Models;

namespace FliptClient
{
    public class EvaluationClient : IDisposable
    {
        private IntPtr _engine;

        public EvaluationClient(string @namespace = "default", ClientOptions? options = null)
        {
            options ??= new ClientOptions();
            string optsJson = JsonSerializer.Serialize(options);
            _engine = NativeMethods.InitializeEngine(@namespace, optsJson);
        }

        public VariantEvaluationResponse? EvaluateVariant(string flagKey, string entityId, Dictionary<string, string> context)
        {
            if (string.IsNullOrWhiteSpace(flagKey))
            {
                throw new ArgumentException("flagKey cannot be empty or null");
            }

            if (string.IsNullOrWhiteSpace(entityId))
            {
                throw new ArgumentException("entityId cannot be empty or null");
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
            string resultJson = Marshal.PtrToStringAnsi(resultPtr) ?? throw new InvalidOperationException("Failed to get result from native code");
            NativeMethods.DestroyString(resultPtr);

            var result = JsonSerializer.Deserialize<VariantResult>(resultJson) ?? throw new InvalidOperationException("Failed to deserialize response");
            if (result.Status != "success")
            {
                throw new Exception(result.ErrorMessage ?? "Unknown error");
            }

            return result.Response;
        }

        public BooleanEvaluationResponse? EvaluateBoolean(string flagKey, string entityId, Dictionary<string, string> context)
        {
            if (string.IsNullOrWhiteSpace(flagKey))
            {
                throw new ArgumentException("flagKey cannot be empty or null");
            }

            if (string.IsNullOrWhiteSpace(entityId))
            {
                throw new ArgumentException("entityId cannot be empty or null");
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
            string resultJson = Marshal.PtrToStringAnsi(resultPtr) ?? throw new InvalidOperationException("Failed to get result from native code");
            NativeMethods.DestroyString(resultPtr);

            var result = JsonSerializer.Deserialize<BooleanResult>(resultJson) ?? throw new InvalidOperationException("Failed to deserialize response");
            if (result.Status != "success")
            {
                throw new Exception(result.ErrorMessage ?? "Unknown error");
            }

            return result.Response;
        }

        public BatchEvaluationResponse? EvaluateBatch(List<EvaluationRequest> requests)
        {
            if (requests == null || requests.Count == 0)
            {
                throw new ArgumentException("requests cannot be empty or null");
            }

            string requestJson = JsonSerializer.Serialize(requests);
            IntPtr resultPtr = NativeMethods.EvaluateBatch(_engine, requestJson);
            string resultJson = Marshal.PtrToStringAnsi(resultPtr) ?? throw new InvalidOperationException("Failed to get result from native code");
            NativeMethods.DestroyString(resultPtr);

            var result = JsonSerializer.Deserialize<BatchResult>(resultJson) ?? throw new InvalidOperationException("Failed to deserialize response");
            if (result.Status != "success")
            {
                throw new Exception(result.ErrorMessage ?? "Unknown error");
            }

            return result.Response;
        }

        public Flag[]? ListFlags()
        {
            IntPtr resultPtr = NativeMethods.ListFlags(_engine);
            string resultJson = Marshal.PtrToStringAnsi(resultPtr) ?? throw new InvalidOperationException("Failed to get result from native code");
            NativeMethods.DestroyString(resultPtr);

            var result = JsonSerializer.Deserialize<ListFlagsResult>(resultJson) ?? throw new InvalidOperationException("Failed to deserialize response");
            if (result.Status != "success")
            {
                throw new Exception(result.ErrorMessage ?? "Unknown error");
            }

            return result.Response;
        }

        public void Dispose()
        {
            if (_engine != IntPtr.Zero)
            {
                NativeMethods.DestroyEngine(_engine);
                _engine = IntPtr.Zero;
            }
        }
    }
}
