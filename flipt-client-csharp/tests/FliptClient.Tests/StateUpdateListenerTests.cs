using System;
using System.Collections.Concurrent;
using System.Collections.Generic;
using System.Net;
using System.Net.Sockets;
using System.Text;
using System.Text.Json;
using System.Threading;
using System.Threading.Tasks;
using FliptClient;
using FliptClient.Models;
using Xunit;

namespace FliptClient.Tests
{
    /// <summary>
    /// Exercises OnStateUpdate against a local HTTP server, so no Flipt instance is needed.
    /// </summary>
    public class StateUpdateListenerTests : IDisposable
    {
        private const string Empty = "{\"namespace\":{\"key\":\"default\"},\"flags\":[]}";
        private const string WithFlag = "{\"namespace\":{\"key\":\"default\"},\"flags\":[{\"key\":\"new_flag\",\"name\":\"New\",\"type\":\"BOOLEAN_FLAG_TYPE\",\"enabled\":true}]}";
        private const string Barrier = "{\"namespace\":{\"key\":\"default\"},\"flags\":[{\"key\":\"barrier\",\"name\":\"Barrier\",\"type\":\"BOOLEAN_FLAG_TYPE\",\"enabled\":true}]}";
        private const string EndStream = "<end>";
        private static readonly TimeSpan Wait = TimeSpan.FromSeconds(5);

        private readonly HttpListener _server;
        private readonly string _url;

        // Documents the stream endpoint sends, one per line, as the test pushes them.
        private readonly BlockingCollection<string> _streamDocs = new BlockingCollection<string>();
        private volatile string _body = Empty;

        public StateUpdateListenerTests()
        {
            // HttpListener can't bind port 0, so probe for a free port. Another process may take it
            // between the probe and Start, so retry a few times.
            for (int attempt = 0; ; attempt++)
            {
                var probe = new TcpListener(IPAddress.Loopback, 0);
                probe.Start();
                int port = ((IPEndPoint)probe.LocalEndpoint).Port;
                probe.Stop();

                var server = new HttpListener();
                string url = $"http://127.0.0.1:{port}";
                server.Prefixes.Add(url + "/");
                try
                {
                    server.Start();
                    _server = server;
                    _url = url;
                    break;
                }
                catch (HttpListenerException) when (attempt < 4)
                {
                    server.Close();
                }
            }

            Task.Run(AcceptLoop);
        }

        public void Dispose()
        {
            _streamDocs.Add(EndStream);
            _server.Close();
            _streamDocs.Dispose();
        }

        [Fact]
        public void ListenerFiresOnChangeAndCanEvaluate()
        {
            int calls = 0;
            bool? evaluated = null;
            using var fired = new ManualResetEventSlim();
            FliptClient? client = null;
            client = NewClient(() =>
            {
                Interlocked.Increment(ref calls);

                // Evaluating from inside the listener must not deadlock.
                evaluated = client!.EvaluateBoolean("new_flag", "entity", new Dictionary<string, string>())?.Enabled;
                fired.Set();
            });
            using (client)
            {
                // Unchanged polls must not fire the listener.
                Thread.Sleep(1500);
                Assert.Equal(0, Volatile.Read(ref calls));

                _body = WithFlag;
                Assert.True(fired.Wait(Wait), "listener did not fire");
                Assert.True(evaluated);
            }
        }

        [Fact]
        public void ListenerFiresOnStreamedChange()
        {
            using var fired = new ManualResetEventSlim();
            _streamDocs.Add(Empty);
            using var client = NewClient(fired.Set, FetchMode.Streaming);
            _streamDocs.Add(WithFlag);
            Assert.True(fired.Wait(Wait), "listener did not fire");
        }

        [Fact]
        public void DisposeWaitsForRunningListener()
        {
            using var started = new ManualResetEventSlim();
            using var release = new ManualResetEventSlim();
            bool finished = false;
            var client = NewClient(() =>
            {
                started.Set();
                release.Wait(Wait);
                finished = true;
            });
            try
            {
                _body = WithFlag;
                Assert.True(started.Wait(Wait), "listener did not start");

                var closer = new Thread(client.Dispose);
                closer.Start();
                Thread.Sleep(300);

                // Dispose must still be waiting for the listener.
                Assert.True(closer.IsAlive);
                release.Set();
                Assert.True(closer.Join(Wait));
                Assert.True(finished);
            }
            finally
            {
                release.Set();
                client.Dispose();
            }
        }

        [Fact]
        public void ThrowingListenerDoesNotBreakClient()
        {
            using var fired = new ManualResetEventSlim();
            using var client = NewClient(() => throw new InvalidOperationException("boom"));
            client.AddStateUpdateListener(fired.Set);
            _body = WithFlag;
            Assert.True(fired.Wait(Wait), "second listener did not fire");
            Assert.True(client.EvaluateBoolean("new_flag", "entity", new Dictionary<string, string>())?.Enabled);
        }

        [Fact]
        public void DisposeFromInsideListener()
        {
            using var done = new ManualResetEventSlim();
            int laterCalls = 0;
            FliptClient? client = null;
            client = NewClient(() =>
            {
                client!.Dispose();
                done.Set();
            });

            // Registered after the disposing listener, so it must never run.
            client.AddStateUpdateListener(() => Interlocked.Increment(ref laterCalls));
            _body = WithFlag;
            Assert.True(done.Wait(Wait), "listener did not dispose the client");
            Thread.Sleep(300);
            Assert.Equal(0, Volatile.Read(ref laterCalls));
        }

        [Fact]
        public void AddAndRemoveListenerWhileRunning()
        {
            using var fired = new ManualResetEventSlim();
            int removedCalls = 0;
            using var client = NewClient();
            var sub = client.AddStateUpdateListener(() => Interlocked.Increment(ref removedCalls));
            sub.Dispose();
            sub.Dispose(); // disposing twice is a no-op
            client.AddStateUpdateListener(fired.Set);
            _body = WithFlag;
            Assert.True(fired.Wait(Wait), "listener did not fire");
            Assert.Equal(0, Volatile.Read(ref removedCalls));
        }

        [Fact]
        public void AddListenerFromInsideListener()
        {
            using var added = new ManualResetEventSlim();
            using var addedFired = new ManualResetEventSlim();
            FliptClient? client = null;
            client = NewClient(() =>
            {
                if (added.IsSet)
                {
                    return;
                }

                client!.AddStateUpdateListener(addedFired.Set);
                added.Set();
            });
            using (client)
            {
                _body = WithFlag;
                Assert.True(added.Wait(Wait), "listener did not add another");
                _body = Barrier;
                Assert.True(addedFired.Wait(Wait), "added listener did not fire");
            }
        }

        [Fact]
        public void AddListenerAfterDisposeThrows()
        {
            // Start the update thread first, so the disposed check is what throws.
            var client = NewClient(() => { });
            client.Dispose();
            Assert.Throws<ObjectDisposedException>(() => client.AddStateUpdateListener(() => { }));
        }

        [Fact]
        public void NullListenerThrows()
        {
            using var client = NewClient();
            Assert.Throws<ValidationException>(() => client.AddStateUpdateListener(null!));
        }

        [Fact]
        public void DisposeStopsListener()
        {
            int calls = 0;
            var client = NewClient(() => Interlocked.Increment(ref calls));
            client.Dispose();
            client.Dispose(); // disposing twice is a no-op
            _body = WithFlag;
            Thread.Sleep(1500);
            Assert.Equal(0, Volatile.Read(ref calls));
        }

        [Fact]
        public void OnStateUpdateIsNotSerialized()
        {
            var options = new ClientOptions { Url = _url, OnStateUpdate = () => { } };
            Assert.DoesNotContain("OnStateUpdate", JsonSerializer.Serialize(options), StringComparison.OrdinalIgnoreCase);
            Assert.DoesNotContain("on_state_update", JsonSerializer.Serialize(options));
        }

        private FliptClient NewClient(Action? onStateUpdate = null, FetchMode fetchMode = FetchMode.Polling)
        {
            return new FliptClient(new ClientOptions
            {
                Url = _url,
                UpdateInterval = TimeSpan.FromSeconds(1),
                ErrorStrategy = ErrorStrategy.Fallback,
                FetchMode = fetchMode,
                OnStateUpdate = onStateUpdate,
            });
        }

        private async Task AcceptLoop()
        {
            while (_server.IsListening)
            {
                HttpListenerContext context;
                try
                {
                    context = await _server.GetContextAsync();
                }
                catch (Exception)
                {
                    // The server was closed.
                    return;
                }

                // The stream handler blocks, so each request gets its own task.
                _ = Task.Run(() => Handle(context));
            }
        }

        private void Handle(HttpListenerContext context)
        {
            var response = context.Response;
            try
            {
                string path = context.Request.Url?.AbsolutePath ?? string.Empty;
                response.ContentType = "application/json";
                if (path.StartsWith("/internal/v1/evaluation/snapshot/namespace/default"))
                {
                    byte[] data = Encoding.UTF8.GetBytes(_body);
                    response.ContentLength64 = data.Length;
                    response.OutputStream.Write(data, 0, data.Length);
                }
                else if (path.StartsWith("/client/v2/environments/default/namespaces/default/stream"))
                {
                    response.SendChunked = true;
                    while (_streamDocs.TryTake(out string? doc, TimeSpan.FromSeconds(30)) && doc != EndStream)
                    {
                        byte[] line = Encoding.UTF8.GetBytes("{\"result\":" + doc + "}\n");
                        response.OutputStream.Write(line, 0, line.Length);
                        response.OutputStream.Flush();
                    }
                }
                else
                {
                    response.StatusCode = 404;
                }

                response.Close();
            }
            catch (Exception)
            {
                // The client went away or the server is stopping.
                response.Abort();
            }
        }
    }
}
