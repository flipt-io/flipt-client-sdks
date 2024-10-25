using System.Text.Json;
using System.Text.Json.Serialization;

namespace FliptClient.Models
{
    public class ClientOptions
    {
        [JsonPropertyName("url")]
        public string Url { get; set; } = "http://localhost:8080";

        [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingDefault)]
        [JsonPropertyName("update_interval")]
        public int UpdateInterval { get; set; } = 120;

        [JsonPropertyName("authentication")]
        public Authentication Authentication { get; set; }

        [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingDefault)]
        [JsonPropertyName("reference")]
        public string Reference { get; set; }

        [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingDefault)]
        [JsonPropertyName("fetch_mode")]
        public FetchMode FetchMode { get; set; } = FetchMode.Polling;
    }

    [JsonConverter(typeof(JsonStringEnumConverter))]
    public enum FetchMode
    {
        [JsonPropertyName("polling")]
        Polling,
        [JsonPropertyName("streaming")]
        Streaming,
    }

    public class Authentication
    {
        [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingDefault)]
        [JsonPropertyName("client_token")]
        public string ClientToken { get; set; }

        [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingDefault)]
        [JsonPropertyName("jwt_token")]
        public string JwtToken { get; set; }
    }
}