using System;
using System.Runtime.InteropServices;
using System.Text.Json;
using FliptClient.Models;

namespace FliptClient
{
    public class EvaluationClient : IDisposable
    {
        private IntPtr _engine;

        public EvaluationClient(string @namespace = "default", ClientOptions options = null)
        {
            options ??= new ClientOptions();
            string optsJson = JsonSerializer.Serialize(options);
            _engine = NativeMethods.InitializeEngine(@namespace, optsJson);
        }

        public VariantEvaluationResponse EvaluateVariant(string flagKey, string entityId, Dictionary<string, string> context)
        {
            var request = new EvaluationRequest
            {
                FlagKey = flagKey,
                EntityId = entityId,
                Context = context
            };

            string requestJson = JsonSerializer.Serialize(request);
            IntPtr resultPtr = NativeMethods.EvaluateVariant(_engine, requestJson);
            string resultJson = Marshal.PtrToStringAnsi(resultPtr);
            NativeMethods.DestroyString(resultPtr);

            var result = JsonSerializer.Deserialize<VariantResult>(resultJson);
            if (result.Status != "success")
            {
                throw new Exception(result.ErrorMessage);
            }

            return result.Result;
        }

        public BooleanEvaluationResponse EvaluateBoolean(string flagKey, string entityId, Dictionary<string, string> context)
        {
            var request = new EvaluationRequest
            {
                FlagKey = flagKey,
                EntityId = entityId,
                Context = context
            };

            string requestJson = JsonSerializer.Serialize(request);
            IntPtr resultPtr = NativeMethods.EvaluateBoolean(_engine, requestJson);
            string resultJson = Marshal.PtrToStringAnsi(resultPtr);
            NativeMethods.DestroyString(resultPtr);

            var result = JsonSerializer.Deserialize<BooleanResult>(resultJson);
            if (result.Status != "success")
            {
                throw new Exception(result.ErrorMessage);
            }

            return result.Result;
        }

        public BatchEvaluationResponse EvaluateBatch(List<EvaluationRequest> requests)
        {
            string requestJson = JsonSerializer.Serialize(requests);
            IntPtr resultPtr = NativeMethods.EvaluateBatch(_engine, requestJson);
            string resultJson = Marshal.PtrToStringAnsi(resultPtr);
            NativeMethods.DestroyString(resultPtr);

            var result = JsonSerializer.Deserialize<BatchResult>(resultJson);
            if (result.Status != "success")
            {
                throw new Exception(result.ErrorMessage);
            }

            return result.Result;
        }

        public ListFlagsResponse ListFlags()
        {
            IntPtr resultPtr = NativeMethods.ListFlags(_engine);
            string resultJson = Marshal.PtrToStringAnsi(resultPtr);
            NativeMethods.DestroyString(resultPtr);

            var result = JsonSerializer.Deserialize<ListFlagsResult>(resultJson);
            if (result.Status != "success")
            {
                throw new Exception(result.ErrorMessage);
            }

            return result.Result;
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
