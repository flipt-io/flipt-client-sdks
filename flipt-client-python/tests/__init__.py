import os
import unittest
from flipt_client import FliptEvaluationClient
from flipt_client.models import (
    ClientTokenAuthentication,
    EngineOpts,
    EvaluationRequest,
)


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

    def test_list_flags(self):
        flags = self.flipt_client.list_flags()
        self.assertIsNone(flags.error_message)
        self.assertEqual("success", flags.status)
        self.assertEqual(2, len(flags.result))

    def test_batch(self):
        batch = self.flipt_client.evaluate_batch(
            [
                EvaluationRequest(
                    flag_key="flag1",
                    entity_id="someentity",
                    context={"fizz": "buzz"},
                ),
                EvaluationRequest(
                    flag_key="flag_boolean",
                    entity_id="someentity",
                    context={"fizz": "buzz"},
                ),
                EvaluationRequest(
                    flag_key="notfound",
                    entity_id="someentity",
                    context={"fizz": "buzz"},
                ),
            ]
        )

        self.assertIsNone(batch.error_message)
        self.assertEqual("success", batch.status)
        self.assertEqual(3, len(batch.result.responses))

        variant = batch.result.responses[0]
        self.assertEqual("VARIANT_EVALUATION_RESPONSE_TYPE", variant.type)
        self.assertEqual("flag1", variant.variant_evaluation_response.flag_key)
        self.assertTrue(variant.variant_evaluation_response.match)
        self.assertEqual(
            "MATCH_EVALUATION_REASON", variant.variant_evaluation_response.reason
        )
        self.assertEqual("variant1", variant.variant_evaluation_response.variant_key)
        self.assertIn("segment1", variant.variant_evaluation_response.segment_keys)

        boolean = batch.result.responses[1]
        self.assertEqual("BOOLEAN_EVALUATION_RESPONSE_TYPE", boolean.type)
        self.assertTrue(boolean.boolean_evaluation_response.enabled)
        self.assertEqual("flag_boolean", boolean.boolean_evaluation_response.flag_key)
        self.assertEqual(
            "MATCH_EVALUATION_REASON", boolean.boolean_evaluation_response.reason
        )

        error = batch.result.responses[2]
        self.assertEqual("ERROR_EVALUATION_RESPONSE_TYPE", error.type)
        self.assertEqual("notfound", error.error_evaluation_response.flag_key)
        self.assertEqual("default", error.error_evaluation_response.namespace_key)
        self.assertEqual(
            "NOT_FOUND_ERROR_EVALUATION_REASON", error.error_evaluation_response.reason
        )

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
