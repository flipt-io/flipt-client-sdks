using System.Text.Json;
using System.Text.Json.Serialization;

namespace FliptClient.Models
{
    public class ClientOptions
    {
        /// <summary>
        /// Gets or sets the Flipt environment to use for evaluations. Defaults to 'default'.
        /// </summary>
        [JsonPropertyName("environment")]
        public string Environment { get; set; } = "default";

        /// <summary>
        /// Gets or sets the Flipt namespace to use for evaluations. Defaults to 'default'.
        /// </summary>
        [JsonPropertyName("namespace")]
        public string Namespace { get; set; } = "default";

        /// <summary>
        /// Gets or sets the Flipt API URL.
        /// </summary>
        [JsonPropertyName("url")]
        public string Url { get; set; } = "http://localhost:8080";

        /// <summary>
        /// Gets or sets request timeout as a duration (serialized as seconds).
        /// </summary>
        [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
        [JsonPropertyName("request_timeout")]
        [JsonConverter(typeof(TimeSpanSecondsConverter))]
        public TimeSpan? RequestTimeout { get; set; }

        /// <summary>
        /// Gets or sets update interval as a duration (serialized as seconds).
        /// </summary>
        [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
        [JsonPropertyName("update_interval")]
        [JsonConverter(typeof(TimeSpanSecondsConverter))]
        public TimeSpan? UpdateInterval { get; set; }

        /// <summary>
        /// Gets or sets authentication options.
        /// </summary>
        [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
        [JsonPropertyName("authentication")]
        public Authentication? Authentication { get; set; }

        /// <summary>
        /// Gets or sets reference string for advanced use cases.
        /// </summary>
        [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
        [JsonPropertyName("reference")]
        public string? Reference { get; set; }

        /// <summary>
        /// Gets or sets fetch mode for flag data (polling or streaming).
        /// </summary>
        [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingDefault)]
        [JsonPropertyName("fetch_mode")]
        public FetchMode FetchMode { get; set; } = FetchMode.Polling;

        /// <summary>
        /// Gets or sets error strategy for evaluation failures.
        /// </summary>
        [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingDefault)]
        [JsonPropertyName("error_strategy")]
        public ErrorStrategy ErrorStrategy { get; set; } = ErrorStrategy.Fail;

        /// <summary>
        /// Gets or sets the initial snapshot to use when instantiating the client.
        /// </summary>
        [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
        [JsonPropertyName("snapshot")]
        public string? Snapshot { get; set; }

        /// <summary>
        /// Gets or sets TLS configuration for connecting to servers with custom certificates.
        /// </summary>
        [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
        [JsonPropertyName("tls_config")]
        public TlsConfig? TlsConfig { get; set; }
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

    public class TimeSpanSecondsConverter : JsonConverter<TimeSpan?>
    {
        public override TimeSpan? Read(ref Utf8JsonReader reader, Type typeToConvert, JsonSerializerOptions options)
        {
            if (reader.TokenType == JsonTokenType.Null)
            {
                return null;
            }

            if (reader.TokenType == JsonTokenType.Number && reader.TryGetInt32(out int seconds))
            {
                return TimeSpan.FromSeconds(seconds);
            }

            throw new JsonException($"Invalid value for TimeSpan: {reader.GetString()}");
        }

        public override void Write(Utf8JsonWriter writer, TimeSpan? value, JsonSerializerOptions options)
        {
            if (value.HasValue)
            {
                writer.WriteNumberValue((int)value.Value.TotalSeconds);
            }
            else
            {
                writer.WriteNullValue();
            }
        }
    }
}
