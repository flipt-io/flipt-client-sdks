using System.Text.Json.Serialization;

namespace FliptClient;

/// <summary>
/// Represents a feature flag.
/// </summary>
public class Flag
{
    /// <summary>
    /// The key of the flag.
    /// </summary>
    [JsonPropertyName("key")]
    public required string Key { get; set; }

    /// <summary>
    /// Whether the flag is enabled.
    /// </summary>
    [JsonPropertyName("enabled")]
    public bool Enabled { get; set; }

    /// <summary>
    /// The type of the flag.
    /// </summary>
    [JsonPropertyName("type")]
    public required string Type { get; set; }
}

/// <summary>
/// Response for a variant flag evaluation.
/// </summary>
public class VariantEvaluationResponse
{
    /// <summary>
    /// The key of the evaluated flag.
    /// </summary>
    [JsonPropertyName("flag_key")]
    public required string FlagKey { get; set; }

    /// <summary>
    /// Whether the evaluation matched a segment.
    /// </summary>
    [JsonPropertyName("match")]
    public bool Match { get; set; }

    /// <summary>
    /// The segment keys that matched.
    /// </summary>
    [JsonPropertyName("segment_keys")]
    public required List<string> SegmentKeys { get; set; }

    /// <summary>
    /// The reason for the evaluation result.
    /// </summary>
    [JsonPropertyName("reason")]
    public required string Reason { get; set; }

    /// <summary>
    /// The key of the selected variant.
    /// </summary>
    [JsonPropertyName("variant_key")]
    public required string VariantKey { get; set; }

    /// <summary>
    /// Optional attachment for the variant.
    /// </summary>
    [JsonPropertyName("variant_attachment")]
    public string? VariantAttachment { get; set; }

    /// <summary>
    /// Duration of the request in milliseconds.
    /// </summary>
    [JsonPropertyName("request_duration_millis")]
    public double RequestDurationMillis { get; set; }

    /// <summary>
    /// Timestamp of the evaluation.
    /// </summary>
    [JsonPropertyName("timestamp")]
    public string? Timestamp { get; set; }
}

/// <summary>
/// Response for a boolean flag evaluation.
/// </summary>
public class BooleanEvaluationResponse
{
    /// <summary>
    /// The key of the evaluated flag.
    /// </summary>
    [JsonPropertyName("flag_key")]
    public required string FlagKey { get; set; }

    /// <summary>
    /// Whether the flag is enabled for the evaluation.
    /// </summary>
    [JsonPropertyName("enabled")]
    public bool Enabled { get; set; }

    /// <summary>
    /// The reason for the evaluation result.
    /// </summary>
    [JsonPropertyName("reason")]
    public required string Reason { get; set; }

    /// <summary>
    /// Duration of the request in milliseconds.
    /// </summary>
    [JsonPropertyName("request_duration_millis")]
    public double RequestDurationMillis { get; set; }

    /// <summary>
    /// Timestamp of the evaluation.
    /// </summary>
    [JsonPropertyName("timestamp")]
    public string? Timestamp { get; set; }
}

/// <summary>
/// Response for a batch flag evaluation.
/// </summary>
public class BatchEvaluationResponse
{
    /// <summary>
    /// The list of individual evaluation responses.
    /// </summary>
    [JsonPropertyName("responses")]
    public required List<Response> Responses { get; set; }

    /// <summary>
    /// Duration of the request in milliseconds.
    /// </summary>
    [JsonPropertyName("request_duration_millis")]
    public double RequestDurationMillis { get; set; }
}

/// <summary>
/// Represents a single response in a batch evaluation.
/// </summary>
public class Response
{
    /// <summary>
    /// The type of the response.
    /// </summary>
    [JsonPropertyName("type")]
    public required string Type { get; set; }

    /// <summary>
    /// The boolean evaluation response, if applicable.
    /// </summary>
    [JsonPropertyName("boolean_evaluation_response")]
    public BooleanEvaluationResponse? BooleanEvaluationResponse { get; set; }

    /// <summary>
    /// The variant evaluation response, if applicable.
    /// </summary>
    [JsonPropertyName("variant_evaluation_response")]
    public VariantEvaluationResponse? VariantEvaluationResponse { get; set; }

    /// <summary>
    /// The error evaluation response, if applicable.
    /// </summary>
    [JsonPropertyName("error_evaluation_response")]
    public ErrorEvaluationResponse? ErrorEvaluationResponse { get; set; }
}

/// <summary>
/// Response for an error during evaluation.
/// </summary>
public class ErrorEvaluationResponse
{
    /// <summary>
    /// The key of the flag for which the error occurred.
    /// </summary>
    [JsonPropertyName("flag_key")]
    public required string FlagKey { get; set; }

    /// <summary>
    /// The namespace key for the flag.
    /// </summary>
    [JsonPropertyName("namespace_key")]
    public required string NamespaceKey { get; set; }

    /// <summary>
    /// The reason for the error.
    /// </summary>
    [JsonPropertyName("reason")]
    public required string Reason { get; set; }
}

/// <summary>
/// Generic result wrapper for evaluation responses.
/// </summary>
public class Result<T>
{
    /// <summary>
    /// The status of the result (e.g., 'success').
    /// </summary>
    [JsonPropertyName("status")]
    public required string Status { get; set; }

    /// <summary>
    /// The response object, if successful.
    /// </summary>
    [JsonPropertyName("result")]
    public T? Response { get; set; }

    /// <summary>
    /// The error message, if any.
    /// </summary>
    [JsonPropertyName("error_message")]
    public string? ErrorMessage { get; set; }
}

/// <summary>
/// Result for a variant evaluation.
/// </summary>
public class VariantResult : Result<VariantEvaluationResponse>
{
}

/// <summary>
/// Result for a boolean evaluation.
/// </summary>
public class BooleanResult : Result<BooleanEvaluationResponse>
{
}

/// <summary>
/// Result for a batch evaluation.
/// </summary>
public class BatchResult : Result<BatchEvaluationResponse>
{
}

/// <summary>
/// Result for listing flags.
/// </summary>
public class ListFlagsResult : Result<Flag[]>
{
}
