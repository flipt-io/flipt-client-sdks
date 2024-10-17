using System;
using System.Runtime.InteropServices;
using System.Text.Json;

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
