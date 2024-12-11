using System;
using System.Collections.Generic;
using Xunit;
using Xunit.Abstractions;
using FliptClient;
using FliptClient.Models;

namespace FliptClient.Tests
{
    public class EvaluationClientTests : IDisposable
    {
        private readonly EvaluationClient _client;

        public EvaluationClientTests(ITestOutputHelper output)
        {
            Console.SetOut(new ConsoleWriter(output));
            string fliptUrl = Environment.GetEnvironmentVariable("FLIPT_URL");
            string authToken = Environment.GetEnvironmentVariable("FLIPT_AUTH_TOKEN");

            if (string.IsNullOrEmpty(fliptUrl))
            {
                throw new Exception("FLIPT_URL environment variable is not set");
            }

            if (string.IsNullOrEmpty(authToken))
            {
                throw new Exception("FLIPT_AUTH_TOKEN environment variable is not set");
            }

            var options = new ClientOptions
            {
                Url = fliptUrl,
                Authentication = new Authentication { ClientToken = authToken }
            };

            _client = new EvaluationClient("default", options);
        }

        public class ConsoleWriter : StringWriter
        {
            private ITestOutputHelper _output;
            public ConsoleWriter(ITestOutputHelper output)
            {
                _output = output;
            }

            public override void WriteLine(string m)
            {
                _output.WriteLine(m);
            }
        }

        public void Dispose()
        {
            _client.Dispose();
        }

        [Fact]
        public void TestNullFlagKey()
        {
            Assert.Throws<ArgumentException>(() => _client.EvaluateBoolean(null, "someentity", new Dictionary<string, string>()));
        }

        [Fact]
        public void TestEmptyFlagKey()
        {
            Assert.Throws<ArgumentException>(() => _client.EvaluateBoolean("", "someentity", new Dictionary<string, string>()));
        }

        [Fact]
        public void TestNullEntityId()
        {
            Assert.Throws<ArgumentException>(() => _client.EvaluateBoolean("flag1", null, new Dictionary<string, string>()));
        }

        [Fact]
        public void TestEmptyEntityId()
        {
            Assert.Throws<ArgumentException>(() => _client.EvaluateBoolean("flag1", "", new Dictionary<string, string>()));
        }

        [Fact]
        public void TestEvaluateVariant()
        {
            var context = new Dictionary<string, string> { { "fizz", "buzz" } };
            var response = _client.EvaluateVariant("flag1", "someentity", context);

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
            var response = _client.EvaluateBoolean("flag_boolean", "someentity", context);

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

            var response = _client.EvaluateBatch(requests);

            Assert.Equal(3, response.Responses.Length);

            var variantResponse = response.Responses[0];
            Assert.Equal("VARIANT_EVALUATION_RESPONSE_TYPE", variantResponse.Type);
            Assert.NotNull(variantResponse.VariantEvaluationResponse);
            Assert.Equal("flag1", variantResponse.VariantEvaluationResponse.FlagKey);
            Assert.True(variantResponse.VariantEvaluationResponse.Match);
            Assert.Equal("MATCH_EVALUATION_REASON", variantResponse.VariantEvaluationResponse.Reason);
            Assert.Equal("variant1", variantResponse.VariantEvaluationResponse.VariantKey);
            Assert.Contains("segment1", variantResponse.VariantEvaluationResponse.SegmentKeys);

            var booleanResponse = response.Responses[1];
            Assert.Equal("BOOLEAN_EVALUATION_RESPONSE_TYPE", booleanResponse.Type);
            Assert.NotNull(booleanResponse.BooleanEvaluationResponse);
            Assert.Equal("flag_boolean", booleanResponse.BooleanEvaluationResponse.FlagKey);
            Assert.True(booleanResponse.BooleanEvaluationResponse.Enabled);
            Assert.Equal("MATCH_EVALUATION_REASON", booleanResponse.BooleanEvaluationResponse.Reason);

            var errorResponse = response.Responses[2];
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
            var exception = Assert.Throws<Exception>(() => _client.EvaluateVariant("nonexistent", "someentity", context));
            Assert.Equal("invalid request: failed to get flag information default/nonexistent", exception.Message);
        }

        [Fact]
        public void TestEvaluateBooleanFailure()
        {
            var context = new Dictionary<string, string> { { "fizz", "buzz" } };
            var exception = Assert.Throws<Exception>(() => _client.EvaluateBoolean("nonexistent", "someentity", context));
            Assert.Equal("invalid request: failed to get flag information default/nonexistent", exception.Message);
        }

        [Fact]
        public void TestListFlags()
        {
            var flags = _client.ListFlags();
            Assert.NotEmpty(flags);
            Assert.Equal(2, flags.Length);
        }
    }
}
