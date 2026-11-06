import queue
import threading
import time
import unittest
from datetime import timedelta
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

from flipt_client import FliptClient
from flipt_client.errors import FliptError
from flipt_client.models import ClientOptions, ErrorStrategy, FetchMode, model_to_json

EMPTY = '{"namespace":{"key":"default"},"flags":[]}'
WITH_FLAG = (
    '{"namespace":{"key":"default"},"flags":[{"key":"new_flag","name":"New",'
    '"type":"BOOLEAN_FLAG_TYPE","enabled":true}]}'
)
BARRIER = (
    '{"namespace":{"key":"default"},"flags":[{"key":"barrier","name":"Barrier",'
    '"type":"BOOLEAN_FLAG_TYPE","enabled":true}]}'
)
END_STREAM = "<end>"


class TestStateUpdateListener(unittest.TestCase):
    """Exercises on_state_update against a local HTTP server, so no Flipt instance is needed."""

    def setUp(self) -> None:
        self.body = EMPTY
        # Documents the stream endpoint sends, one per line, as the test pushes them.
        self.stream_docs: queue.Queue[str] = queue.Queue()
        test = self

        class Handler(BaseHTTPRequestHandler):
            protocol_version = "HTTP/1.1"

            def log_message(self, format, *args):
                pass

            def do_GET(self):
                if self.path.startswith(
                    "/internal/v1/evaluation/snapshot/namespace/default"
                ):
                    data = test.body.encode("utf-8")
                    self.send_response(200)
                    self.send_header("Content-Type", "application/json")
                    self.send_header("Content-Length", str(len(data)))
                    self.end_headers()
                    self.wfile.write(data)
                elif self.path.startswith(
                    "/client/v2/environments/default/namespaces/default/stream"
                ):
                    self.send_response(200)
                    self.send_header("Content-Type", "application/json")
                    self.send_header("Transfer-Encoding", "chunked")
                    self.end_headers()
                    try:
                        while True:
                            try:
                                doc = test.stream_docs.get(timeout=30)
                            except queue.Empty:
                                break
                            if doc == END_STREAM:
                                break
                            line = ('{"result":' + doc + "}\n").encode("utf-8")
                            self.wfile.write(b"%x\r\n%s\r\n" % (len(line), line))
                            self.wfile.flush()
                        self.wfile.write(b"0\r\n\r\n")
                    except OSError:
                        # The client went away or the server is stopping.
                        pass
                    self.close_connection = True
                else:
                    self.send_response(404)
                    self.send_header("Content-Length", "0")
                    self.end_headers()

        self.server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
        self.server.daemon_threads = True
        threading.Thread(target=self.server.serve_forever, daemon=True).start()

    def tearDown(self) -> None:
        self.stream_docs.put(END_STREAM)
        self.server.shutdown()
        self.server.server_close()

    def client(self, **kwargs) -> FliptClient:
        return FliptClient(
            opts=ClientOptions(
                url=f"http://127.0.0.1:{self.server.server_address[1]}",
                update_interval=timedelta(seconds=1),
                error_strategy=ErrorStrategy.FALLBACK,
                **kwargs,
            )
        )

    def test_on_state_update_is_not_serialized(self):
        opts = ClientOptions(url="http://localhost:8080", on_state_update=lambda: None)
        self.assertNotIn("on_state_update", model_to_json(opts, exclude_none=True))
        self.assertNotIn("on_state_update", model_to_json(opts))

    def test_listener_fires_on_change_and_can_evaluate(self):
        calls = []
        evaluated = []
        fired = threading.Event()
        holder = {}

        def listener():
            calls.append(1)
            # Evaluating from inside the listener must not deadlock.
            evaluated.append(
                holder["client"].evaluate_boolean("new_flag", "entity", {}).enabled
            )
            fired.set()

        client = self.client(on_state_update=listener)
        holder["client"] = client
        try:
            # Unchanged polls must not fire the listener.
            time.sleep(1.5)
            self.assertEqual(0, len(calls))

            self.body = WITH_FLAG
            self.assertTrue(fired.wait(5), "listener did not fire")
            self.assertEqual([True], evaluated)
        finally:
            client.close()

    def test_listener_fires_on_streamed_change(self):
        fired = threading.Event()
        self.stream_docs.put(EMPTY)
        client = self.client(fetch_mode=FetchMode.STREAMING, on_state_update=fired.set)
        try:
            self.stream_docs.put(WITH_FLAG)
            self.assertTrue(fired.wait(5), "listener did not fire")
        finally:
            client.close()

    def test_close_waits_for_running_listener(self):
        started = threading.Event()
        release = threading.Event()
        finished = threading.Event()

        def listener():
            started.set()
            release.wait(5)
            finished.set()

        client = self.client(on_state_update=listener)
        try:
            self.body = WITH_FLAG
            self.assertTrue(started.wait(5), "listener did not start")

            closer = threading.Thread(target=client.close)
            closer.start()
            time.sleep(0.3)
            # close() must still be waiting for the listener.
            self.assertTrue(closer.is_alive())
            release.set()
            closer.join(5)
            self.assertFalse(closer.is_alive())
            self.assertTrue(finished.is_set())
        finally:
            release.set()
            client.close()

    def test_raising_listener_does_not_break_client(self):
        fired = threading.Event()

        def bad():
            raise RuntimeError("boom")

        client = self.client(on_state_update=bad)
        try:
            client.add_state_update_listener(fired.set)
            self.body = WITH_FLAG
            self.assertTrue(fired.wait(5), "second listener did not fire")
            self.assertTrue(client.evaluate_boolean("new_flag", "entity").enabled)
        finally:
            client.close()

    def test_close_from_inside_listener(self):
        done = threading.Event()
        later_calls = []
        holder = {}

        def listener():
            holder["client"].close()
            done.set()

        client = self.client(on_state_update=listener)
        holder["client"] = client
        # Registered after the closing listener, so it must never run.
        client.add_state_update_listener(lambda: later_calls.append(1))
        self.body = WITH_FLAG
        self.assertTrue(done.wait(5), "listener did not close the client")
        self.assertIsNone(client.engine)
        time.sleep(0.3)
        self.assertEqual([], later_calls)

    def test_add_and_remove_listener_while_running(self):
        first = threading.Event()
        removed_calls = []
        client = self.client()
        try:
            sub = client.add_state_update_listener(lambda: removed_calls.append(1))
            sub.close()
            sub.close()  # closing twice is a no-op
            client.add_state_update_listener(first.set)
            self.body = WITH_FLAG
            self.assertTrue(first.wait(5), "listener did not fire")
            self.assertEqual([], removed_calls)
        finally:
            client.close()

    def test_add_listener_from_inside_listener(self):
        added_fired = threading.Event()
        added = threading.Event()
        holder: dict[str, FliptClient] = {}

        def listener():
            if not added.is_set():
                added.set()
                holder["client"].add_state_update_listener(added_fired.set)

        client = self.client(on_state_update=listener)
        holder["client"] = client
        try:
            self.body = WITH_FLAG
            self.assertTrue(added.wait(5), "listener did not add another")
            self.body = BARRIER
            self.assertTrue(added_fired.wait(5), "added listener did not fire")
        finally:
            client.close()

    def test_add_listener_after_close_raises(self):
        # Start the update thread first, so the closed check is what raises.
        client = self.client(on_state_update=lambda: None)
        client.close()
        with self.assertRaisesRegex(FliptError, "client is closed"):
            client.add_state_update_listener(lambda: None)

    def test_close_stops_listener(self):
        calls = []
        client = self.client(on_state_update=lambda: calls.append(1))
        client.close()
        self.body = WITH_FLAG
        time.sleep(1.5)
        self.assertEqual([], calls)


if __name__ == "__main__":
    unittest.main()
