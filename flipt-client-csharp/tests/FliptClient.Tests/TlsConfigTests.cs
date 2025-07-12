using System.Text.Json;
using Xunit;
using FliptClient.Models;

namespace FliptClient.Tests
{
    public class TlsConfigTests
    {
        [Fact]
        public void TestInsecureSkipHostnameVerifySerializationWhenTrue()
        {
            var tlsConfig = new TlsConfig
            {
                InsecureSkipHostnameVerify = true
            };

            string json = JsonSerializer.Serialize(tlsConfig);
            Assert.Contains("\"insecure_skip_hostname_verify\":true", json);
        }

        [Fact]
        public void TestInsecureSkipHostnameVerifySerializationWhenFalse()
        {
            var tlsConfig = new TlsConfig
            {
                InsecureSkipHostnameVerify = false
            };

            string json = JsonSerializer.Serialize(tlsConfig);
            Assert.Contains("\"insecure_skip_hostname_verify\":false", json);
        }

        [Fact]
        public void TestInsecureSkipHostnameVerifySerializationWhenNull()
        {
            var tlsConfig = new TlsConfig
            {
                InsecureSkipHostnameVerify = null
            };

            string json = JsonSerializer.Serialize(tlsConfig);
            Assert.DoesNotContain("insecure_skip_hostname_verify", json);
        }

        [Fact]
        public void TestInsecureSkipHostnameVerifyJsonPropertyName()
        {
            var tlsConfig = new TlsConfig
            {
                InsecureSkipHostnameVerify = true
            };

            string json = JsonSerializer.Serialize(tlsConfig);
            
            // Verify it serializes with correct JSON property name
            Assert.Contains("\"insecure_skip_hostname_verify\":true", json);
        }
    }
}