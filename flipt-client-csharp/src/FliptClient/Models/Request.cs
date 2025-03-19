using System.Text.Json.Serialization;

namespace FliptClient;

public class EvaluationRequest
{
    [JsonPropertyName("flag_key")]
    public required string FlagKey { get; set; }

    [JsonPropertyName("entity_id")]
    public required string EntityId { get; set; }

    [JsonPropertyName("context")]
    public required Dictionary<string, string> Context { get; set; }
}
