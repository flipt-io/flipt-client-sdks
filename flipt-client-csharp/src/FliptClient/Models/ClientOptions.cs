using System.Text.Json.Serialization;

namespace FliptClient.Models
{
    public class ClientOptions
    {
        [JsonPropertyName("url")]
        public string Url { get; set; } = "http://localhost:8080";

        [JsonPropertyName("update_interval")]
        public int UpdateInterval { get; set; } = 120;

        [JsonPropertyName("authentication")]
        public Authentication Authentication { get; set; }

        [JsonPropertyName("reference")]
        public string Reference { get; set; }

        [JsonPropertyName("fetch_mode")]
        public FetchMode FetchMode { get; set; } = FetchMode.Polling;
    }

    public enum FetchMode
    {
        [JsonPropertyName("polling")]
        Polling,
        [JsonPropertyName("streaming")]
        Streaming,
    }

    public class Authentication
    {
        [JsonPropertyName("client_token")]
        public string ClientToken { get; set; }

        [JsonPropertyName("jwt_token")]
        public string JwtToken { get; set; }
    }
}