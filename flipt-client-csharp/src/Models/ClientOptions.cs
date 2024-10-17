using System.Text.Json.Serialization;

namespace FliptClient.Models
{
    public class ClientOptions
    {
        [JsonPropertyName("url")]
        public string Url { get; set; } = "http://localhost:8080";

        [JsonPropertyName("update_interval")]
        public int UpdateInterval { get; set; } = 60;

        [JsonPropertyName("authentication")]
        public Authentication Authentication { get; set; }
    }

    public class Authentication
    {
        [JsonPropertyName("client_token")]
        public string ClientToken { get; set; }

        [JsonPropertyName("jwt_token")]
        public string JwtToken { get; set; }
    }
}