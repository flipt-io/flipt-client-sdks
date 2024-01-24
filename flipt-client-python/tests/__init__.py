import os
import unittest
from flipt_client import FliptEvaluationClient
from flipt_client.models import ClientTokenAuthentication, EngineOpts


class TestFliptEvaluationClient(unittest.TestCase):
    def setUp(self) -> None:
        engine_url = os.environ.get("FLIPT_URL")
        if engine_url is None:
            raise Exception("FLIPT_URL not set")

        auth_token = os.environ.get("FLIPT_AUTH_TOKEN")
        if auth_token is None:
            raise Exception("FLIPT_AUTH_TOKEN not set")

        self.flipt_client = FliptEvaluationClient(
            engine_opts=EngineOpts(
                url=engine_url,
                authentication=ClientTokenAuthentication(client_token=auth_token),
            )
        )

    def test_variant(self):
        variant = self.flipt_client.evaluate_variant(
            "flag1", "someentity", {"fizz": "buzz"}
        )
        self.assertIsNone(variant.error_message)
        self.assertEqual("success", variant.status)
        self.assertEqual("flag1", variant.result.flag_key)
        self.assertTrue(variant.result.match)
        self.assertEqual("MATCH_EVALUATION_REASON", variant.result.reason)
        self.assertEqual("variant1", variant.result.variant_key)
        self.assertIn("segment1", variant.result.segment_keys)

    def test_boolean(self):
        boolean = self.flipt_client.evaluate_boolean(
            "flag_boolean", "someentity", {"fizz": "buzz"}
        )
        self.assertIsNone(boolean.error_message)
        self.assertEqual("success", boolean.status)
        self.assertTrue(boolean.result.enabled)
        self.assertEqual("flag_boolean", boolean.result.flag_key)
        self.assertEqual("MATCH_EVALUATION_REASON", boolean.result.reason)

    def test_failure_variant(self):
        variant = self.flipt_client.evaluate_variant(
            "nonexistent", "someentity", {"fizz": "buzz"}
        )
        self.assertIsNone(variant.result)
        self.assertEqual(
            variant.error_message,
            "invalid request: failed to get flag information default/nonexistent",
        )
        self.assertEqual("failure", variant.status)

    def test_failure_boolean(self):
        boolean = self.flipt_client.evaluate_boolean(
            "nonexistent", "someentity", {"fizz": "buzz"}
        )
        self.assertIsNone(boolean.result)
        self.assertEqual(
            boolean.error_message,
            "invalid request: failed to get flag information default/nonexistent",
        )
        self.assertEqual("failure", boolean.status)

    def test_variant_no_context(self):
        variant = self.flipt_client.evaluate_variant("flag1", "someentity")
        self.assertIsNone(variant.error_message)
        self.assertEqual("success", variant.status)
        self.assertEqual("flag1", variant.result.flag_key)
        self.assertFalse(variant.result.match)

    def test_boolean_no_context(self):
        boolean = self.flipt_client.evaluate_boolean("flag_boolean", "someentity")
        self.assertIsNone(boolean.error_message)
        self.assertEqual("success", boolean.status)
        self.assertTrue(boolean.result.enabled)
        self.assertEqual("flag_boolean", boolean.result.flag_key)


if __name__ == "__main__":
    unittest.main()
