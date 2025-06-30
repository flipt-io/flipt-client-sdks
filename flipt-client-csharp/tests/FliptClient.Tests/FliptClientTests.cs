using System;
using System.Collections.Generic;
using System.Text.Json;
using Xunit;
using Xunit.Abstractions;
using FliptClient;
using FliptClient.Models;

namespace FliptClient.Tests
{
    public class FliptClientTests : IDisposable
    {
        private readonly FliptClient _client;

        public FliptClientTests(ITestOutputHelper output)
        {
            Console.SetOut(new ConsoleWriter(output));
            string? fliptUrl = Environment.GetEnvironmentVariable("FLIPT_URL");
            string? authToken = Environment.GetEnvironmentVariable("FLIPT_AUTH_TOKEN");

            if (string.IsNullOrEmpty(fliptUrl))
            {
                throw new ValidationException("FLIPT_URL environment variable is not set");
            }

            if (string.IsNullOrEmpty(authToken))
            {
                throw new ValidationException("FLIPT_AUTH_TOKEN environment variable is not set");
            }

            var options = new ClientOptions
            {
                Url = fliptUrl,
                Namespace = "default",
                Authentication = new Authentication { ClientToken = authToken }
            };

            // Configure TLS if HTTPS URL is provided
            if (fliptUrl.StartsWith("https://"))
            {
                string? caCertPath = Environment.GetEnvironmentVariable("FLIPT_CA_CERT_PATH");
                if (!string.IsNullOrEmpty(caCertPath))
                {
                    options.TlsConfig = TlsConfig.WithCaCertFile(caCertPath);
                }
                else
                {
                    // Fallback to insecure for local testing
                    options.TlsConfig = TlsConfig.Insecure();
                }
            }

            _client = new FliptClient(options);
        }

        public class ConsoleWriter : StringWriter
        {
            private readonly ITestOutputHelper _output;
            public ConsoleWriter(ITestOutputHelper output)
            {
                _output = output;
            }

            public override void WriteLine(string? message)
            {
                if (message != null)
                {
                    _output.WriteLine(message);
                }
            }
        }

        public void Dispose()
        {
            _client.Dispose();
        }

        [Fact]
        public void TestNullFlagKey()
        {
            Assert.Throws<ValidationException>(() => _client.EvaluateBoolean(null!, "someentity", new Dictionary<string, string>()));
        }

        [Fact]
        public void TestEmptyFlagKey()
        {
            Assert.Throws<ValidationException>(() => _client.EvaluateBoolean("", "someentity", new Dictionary<string, string>()));
        }

        [Fact]
        public void TestNullEntityId()
        {
            Assert.Throws<ValidationException>(() => _client.EvaluateBoolean("flag1", null!, new Dictionary<string, string>()));
        }

        [Fact]
        public void TestEmptyEntityId()
        {
            Assert.Throws<ValidationException>(() => _client.EvaluateBoolean("flag1", "", new Dictionary<string, string>()));
        }

        [Fact]
        public void TestEvaluateVariant()
        {
            var context = new Dictionary<string, string> { { "fizz", "buzz" } };
            var response = _client.EvaluateVariant("flag1", "someentity", context)!;

            Assert.Equal("flag1", response.FlagKey);
            Assert.True(response.Match);
            Assert.Equal("MATCH_EVALUATION_REASON", response.Reason);
            Assert.Equal("variant1", response.VariantKey);
            Assert.Contains("segment1", response.SegmentKeys);
        }

        [Fact]
        public void TestEvaluateBoolean()
        {
            var context = new Dictionary<string, string> { { "fizz", "buzz" } };
            var response = _client.EvaluateBoolean("flag_boolean", "someentity", context)!;

            Assert.Equal("flag_boolean", response.FlagKey);
            Assert.True(response.Enabled);
            Assert.Equal("MATCH_EVALUATION_REASON", response.Reason);
        }

        [Fact]
        public void TestEvaluateBatch()
        {
            var requests = new List<EvaluationRequest>
            {
                new EvaluationRequest { FlagKey = "flag1", EntityId = "someentity", Context = new Dictionary<string, string> { { "fizz", "buzz" } } },
                new EvaluationRequest { FlagKey = "flag_boolean", EntityId = "someentity", Context = new Dictionary<string, string> { { "fizz", "buzz" } } },
                new EvaluationRequest { FlagKey = "notfound", EntityId = "someentity", Context = new Dictionary<string, string> { { "fizz", "buzz" } } }
            };

            var result = _client.EvaluateBatch(requests)!;
            Assert.NotNull(result);
            Assert.Equal(3, result.Responses.Count);

            var variantResponse = result.Responses[0];
            Assert.Equal("VARIANT_EVALUATION_RESPONSE_TYPE", variantResponse.Type);
            Assert.NotNull(variantResponse.VariantEvaluationResponse);
            Assert.Equal("flag1", variantResponse.VariantEvaluationResponse.FlagKey);
            Assert.True(variantResponse.VariantEvaluationResponse.Match);
            Assert.Equal("MATCH_EVALUATION_REASON", variantResponse.VariantEvaluationResponse.Reason);
            Assert.Equal("variant1", variantResponse.VariantEvaluationResponse.VariantKey);
            Assert.Contains("segment1", variantResponse.VariantEvaluationResponse.SegmentKeys);

            var booleanResponse = result.Responses[1];
            Assert.Equal("BOOLEAN_EVALUATION_RESPONSE_TYPE", booleanResponse.Type);
            Assert.NotNull(booleanResponse.BooleanEvaluationResponse);
            Assert.Equal("flag_boolean", booleanResponse.BooleanEvaluationResponse.FlagKey);
            Assert.True(booleanResponse.BooleanEvaluationResponse.Enabled);
            Assert.Equal("MATCH_EVALUATION_REASON", booleanResponse.BooleanEvaluationResponse.Reason);

            var errorResponse = result.Responses[2];
            Assert.Equal("ERROR_EVALUATION_RESPONSE_TYPE", errorResponse.Type);
            Assert.NotNull(errorResponse.ErrorEvaluationResponse);
            Assert.Equal("notfound", errorResponse.ErrorEvaluationResponse.FlagKey);
            Assert.Equal("default", errorResponse.ErrorEvaluationResponse.NamespaceKey);
            Assert.Equal("NOT_FOUND_ERROR_EVALUATION_REASON", errorResponse.ErrorEvaluationResponse.Reason);
        }

        [Fact]
        public void TestEvaluateVariantFailure()
        {
            var context = new Dictionary<string, string> { { "fizz", "buzz" } };
            var exception = Assert.Throws<EvaluationException>(() => _client.EvaluateVariant("nonexistent", "someentity", context));
            Assert.Equal("invalid request: failed to get flag information default/nonexistent", exception.Message);
        }

        [Fact]
        public void TestEvaluateBooleanFailure()
        {
            var context = new Dictionary<string, string> { { "fizz", "buzz" } };
            var exception = Assert.Throws<EvaluationException>(() => _client.EvaluateBoolean("nonexistent", "someentity", context));
            Assert.Equal("invalid request: failed to get flag information default/nonexistent", exception.Message);
        }

        [Fact]
        public void TestListFlags()
        {
            var flags = _client.ListFlags()!;
            Assert.NotEmpty(flags);
            Assert.Equal(2, flags.Length);
        }

        [Fact]
        public void TestClientOptions()
        {
            var options = new ClientOptions { ErrorStrategy = ErrorStrategy.Fallback };
            string optsJson = JsonSerializer.Serialize(options);
            string expectedJson = """{"environment":"default","namespace":"default","url":"http://localhost:8080","error_strategy":"fallback"}""";
            Assert.Equal(expectedJson, optsJson);
        }

        [Fact]
        public void TestSetGetSnapshotWithInvalidFliptURL()
        {
            // Get a snapshot from a working client
            string? snapshot = _client.GetSnapshot();
            Assert.False(string.IsNullOrEmpty(snapshot));

            // Create a client with the previous snapshot and an invalid URL
            var authToken = Environment.GetEnvironmentVariable("FLIPT_AUTH_TOKEN") ?? throw new ValidationException("FLIPT_AUTH_TOKEN environment variable is not set");
            var options = new ClientOptions
            {
                Url = "http://invalid.flipt.com",
                Namespace = "default",
                Snapshot = snapshot!,
                ErrorStrategy = ErrorStrategy.Fallback,
                Authentication = new Authentication { ClientToken = authToken }
            };
            using var invalidClient = new FliptClient(options);

            var context = new Dictionary<string, string> { { "fizz", "buzz" } };

            for (int i = 0; i < 5; i++)
            {
                var variantResponse = invalidClient.EvaluateVariant("flag1", "someentity", context);
                Assert.NotNull(variantResponse);
                Assert.Equal("flag1", variantResponse!.FlagKey);
                Assert.True(variantResponse.Match);
                Assert.Equal("MATCH_EVALUATION_REASON", variantResponse.Reason);
                Assert.Equal("variant1", variantResponse.VariantKey);
                Assert.Contains("segment1", variantResponse.SegmentKeys);

                var booleanResponse = invalidClient.EvaluateBoolean("flag_boolean", "someentity", context);
                Assert.NotNull(booleanResponse);
                Assert.Equal("flag_boolean", booleanResponse!.FlagKey);
                Assert.True(booleanResponse.Enabled);
                Assert.Equal("MATCH_EVALUATION_REASON", booleanResponse.Reason);

                var flags = invalidClient.ListFlags();
                Assert.NotNull(flags);
                Assert.Equal(2, flags!.Length);

                snapshot = invalidClient.GetSnapshot();
                Assert.False(string.IsNullOrEmpty(snapshot));
            }
        }
    }
}
