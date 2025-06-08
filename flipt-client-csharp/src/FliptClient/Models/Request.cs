using System.Text.Json.Serialization;

namespace FliptClient;

/// <summary>
/// Represents a request to evaluate a flag for a given entity and context.
/// </summary>
public class EvaluationRequest
{
    /// <summary>
    /// The key of the flag to evaluate.
    /// </summary>
    [JsonPropertyName("flag_key")]
    public required string FlagKey { get; set; }

    /// <summary>
    /// The entity ID for which the flag is being evaluated.
    /// </summary>
    [JsonPropertyName("entity_id")]
    public required string EntityId { get; set; }

    /// <summary>
    /// The evaluation context as key-value pairs.
    /// </summary>
    [JsonPropertyName("context")]
    public required Dictionary<string, string> Context { get; set; }
}
