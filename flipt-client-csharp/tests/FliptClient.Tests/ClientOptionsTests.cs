using System.Text.Json;
using FliptClient.Models;
using Xunit;

namespace FliptClient.Tests
{
    public class ClientOptionsTests
    {
        [Fact]
        public void StreamingFetchModeIsSerializedLowercase()
        {
            // The engine rejects "Streaming" and then drops every other option too.
            var options = new ClientOptions { FetchMode = FetchMode.Streaming };
            Assert.Contains("\"fetch_mode\":\"streaming\"", JsonSerializer.Serialize(options));
        }
    }
}
