using System.Text.Json;
using System.Text.Json.Serialization;

namespace FliptClient.Models
{
    public class ClientOptions
    {
        /// <summary>
        /// The Flipt environment to use for evaluations. Defaults to 'default'.
        /// </summary>
        [JsonPropertyName("environment")]
        public string Environment { get; set; } = "default";

        /// <summary>
        /// The Flipt namespace to use for evaluations. Defaults to 'default'.
        /// </summary>
        [JsonPropertyName("namespace")]
        public string Namespace { get; set; } = "default";

        /// <summary>
        /// The Flipt API URL.
        /// </summary>
        [JsonPropertyName("url")]
        public string Url { get; set; } = "http://localhost:8080";

        /// <summary>
        /// Request timeout in milliseconds.
        /// </summary>
        [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
        [JsonPropertyName("request_timeout")]
        public int? RequestTimeout { get; set; }

        /// <summary>
        /// Update interval in milliseconds.
        /// </summary>
        [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
        [JsonPropertyName("update_interval")]
        public int? UpdateInterval { get; set; }

        /// <summary>
        /// Authentication options.
        /// </summary>
        [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
        [JsonPropertyName("authentication")]
        public Authentication? Authentication { get; set; }

        /// <summary>
        /// Reference string for advanced use cases.
        /// </summary>
        [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
        [JsonPropertyName("reference")]
        public string? Reference { get; set; }

        /// <summary>
        /// Fetch mode for flag data (polling or streaming).
        /// </summary>
        [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingDefault)]
        [JsonPropertyName("fetch_mode")]
        public FetchMode FetchMode { get; set; } = FetchMode.Polling;

        /// <summary>
        /// Error strategy for evaluation failures.
        /// </summary>
        [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingDefault)]
        [JsonPropertyName("error_strategy")]
        public ErrorStrategy ErrorStrategy { get; set; } = ErrorStrategy.Fail;
    }

    [JsonConverter(typeof(LowercaseEnumConverter<ErrorStrategy>))]
    public enum ErrorStrategy
    {
        Fail,
        Fallback,
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
        [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
        [JsonPropertyName("client_token")]
        public string? ClientToken { get; set; }

        [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
        [JsonPropertyName("jwt_token")]
        public string? JwtToken { get; set; }
    }

    public class LowercaseEnumConverter<T> : JsonConverter<T>
        where T : struct, Enum
    {
        public override T Read(ref Utf8JsonReader reader, Type typeToConvert, JsonSerializerOptions options)
        {
            var enumString = reader.GetString();
            if (Enum.TryParse<T>(enumString, true, out var value))
            {
                return value;
            }

            throw new JsonException($"Unable to convert \"{enumString}\" to {typeof(T)}.");
        }

        public override void Write(Utf8JsonWriter writer, T value, JsonSerializerOptions options)
        {
            writer.WriteStringValue(value.ToString().ToLower()); // Convert enum name to lowercase
        }
    }
}
