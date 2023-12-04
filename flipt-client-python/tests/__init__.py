import os
import unittest
from flipt_client_python import FliptEvaluationClient
from flipt_client_python.models import EngineOpts


class TestFliptEvaluationClient(unittest.TestCase):
    def setUp(self) -> None:
        engine_url = os.environ.get("FLIPT_URL")
        if engine_url is None:
            raise Exception("FLIPT_URL not set")

        self.flipt_client = FliptEvaluationClient(
            engine_opts=EngineOpts(url=engine_url)
        )

    def test_variant(self):
        variant = self.flipt_client.evaluate_variant("flag1", "someentity", {"fizz": "buzz"})
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


if __name__ == "__main__":
    unittest.main()
