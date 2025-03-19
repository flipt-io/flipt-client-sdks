using System.Text.Json.Serialization;

namespace FliptClient;

public class Flag
{
    [JsonPropertyName("key")]
    public required string Key { get; set; }

    [JsonPropertyName("enabled")]
    public bool Enabled { get; set; }

    [JsonPropertyName("type")]
    public required string Type { get; set; }
}

public class VariantEvaluationResponse
{
    [JsonPropertyName("flag_key")]
    public required string FlagKey { get; set; }

    [JsonPropertyName("match")]
    public bool Match { get; set; }

    [JsonPropertyName("segment_keys")]
    public required List<string> SegmentKeys { get; set; }

    [JsonPropertyName("reason")]
    public required string Reason { get; set; }

    [JsonPropertyName("variant_key")]
    public required string VariantKey { get; set; }

    [JsonPropertyName("variant_attachment")]
    public string? VariantAttachment { get; set; }

    [JsonPropertyName("request_duration_millis")]
    public double RequestDurationMillis { get; set; }

    [JsonPropertyName("timestamp")]
    public string? Timestamp { get; set; }
}

public class BooleanEvaluationResponse
{
    [JsonPropertyName("flag_key")]
    public required string FlagKey { get; set; }

    [JsonPropertyName("enabled")]
    public bool Enabled { get; set; }

    [JsonPropertyName("reason")]
    public required string Reason { get; set; }

    [JsonPropertyName("request_duration_millis")]
    public double RequestDurationMillis { get; set; }

    [JsonPropertyName("timestamp")]
    public string? Timestamp { get; set; }
}

public class BatchEvaluationResponse
{
    [JsonPropertyName("responses")]
    public required List<Response> Responses { get; set; }

    [JsonPropertyName("request_duration_millis")]
    public double RequestDurationMillis { get; set; }
}

public class Response
{
    [JsonPropertyName("type")]
    public required string Type { get; set; }

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
    public required string FlagKey { get; set; }

    [JsonPropertyName("namespace_key")]
    public required string NamespaceKey { get; set; }

    [JsonPropertyName("reason")]
    public required string Reason { get; set; }
}

public class Result<T>
{
    [JsonPropertyName("status")]
    public required string Status { get; set; }

    [JsonPropertyName("result")]
    public T? Response { get; set; }

    [JsonPropertyName("error_message")]
    public string? ErrorMessage { get; set; }
}

public class VariantResult : Result<VariantEvaluationResponse>
{
}

public class BooleanResult : Result<BooleanEvaluationResponse>
{
}

public class BatchResult : Result<BatchEvaluationResponse>
{
}

public class ListFlagsResult : Result<Flag[]>
{
}
