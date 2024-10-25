using System.Text.Json.Serialization;

namespace FliptClient.Models
{
    public class Flag
    {
        [JsonPropertyName("key")]
        public string Key { get; set; }

        [JsonPropertyName("enabled")]
        public bool Enabled { get; set; }

        [JsonPropertyName("type")]
        public string Type { get; set; }
    }

    public class VariantEvaluationResponse
    {
        [JsonPropertyName("flag_key")]
        public string FlagKey { get; set; }

        [JsonPropertyName("match")]
        public bool Match { get; set; }

        [JsonPropertyName("segment_keys")]
        public string[] SegmentKeys { get; set; }

        [JsonPropertyName("reason")]
        public string Reason { get; set; }

        [JsonPropertyName("variant_key")]
        public string VariantKey { get; set; }

        [JsonPropertyName("variant_attachment")]
        public string? VariantAttachment { get; set; }
    }

    public class BooleanEvaluationResponse
    {
        [JsonPropertyName("flag_key")]
        public string FlagKey { get; set; }

        [JsonPropertyName("enabled")]
        public bool Enabled { get; set; }

        [JsonPropertyName("reason")]
        public string Reason { get; set; }
    }

    public class BatchEvaluationResponse
    {
        [JsonPropertyName("request_id")]
        public string RequestId { get; set; }

        [JsonPropertyName("responses")]
        public Response[] Responses { get; set; }

        [JsonPropertyName("request_duration_millis")]
        public float RequestDurationMillis { get; set; }
    }


    public class Response
    {
        [JsonPropertyName("type")]
        public string Type { get; set; }

        [JsonPropertyName("boolean_evaluation_response")]
        public BooleanEvaluationResponse? BooleanEvaluationResponse { get; set; }

        [JsonPropertyName("variant_evaluation_response")]
        public VariantEvaluationResponse? VariantEvaluationResponse { get; set; }

        [JsonPropertyName("error_evaluation_response")]
        public ErrorEvaluationResponse? ErrorEvaluationResponse { get; set; }
    }


    public class ErrorEvaluationResponse
    {
        [JsonPropertyName("flag_key")]
        public string? FlagKey { get; set; }

        [JsonPropertyName("namespace_key")]
        public string? NamespaceKey { get; set; }

        [JsonPropertyName("reason")]
        public string? Reason { get; set; }
    }

    public class Result<T>
    {
        [JsonPropertyName("status")]
        public string Status { get; set; }

        [JsonPropertyName("result")]
        public T? Response { get; set; }

        [JsonPropertyName("error_message")]
        public string? ErrorMessage { get; set; }
    }

    public class VariantResult : Result<VariantEvaluationResponse> { }
    public class BooleanResult : Result<BooleanEvaluationResponse> { }
    public class BatchResult : Result<BatchEvaluationResponse> { }
    public class ListFlagsResult : Result<Flag[]> { }
}
