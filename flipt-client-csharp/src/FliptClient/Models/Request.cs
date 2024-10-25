using System.Text.Json.Serialization;

namespace FliptClient.Models
{
    public class EvaluationRequest
    {
        [JsonPropertyName("flag_key")]
        public string FlagKey { get; set; }

        [JsonPropertyName("entity_id")]
        public string EntityId { get; set; }

        [JsonPropertyName("context")]
        public Dictionary<string, string> Context { get; set; }
    }
}

