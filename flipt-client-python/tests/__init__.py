import os
import unittest
from flipt_client import FliptEvaluationClient
from flipt_client.models import (
    ClientTokenAuthentication,
    ClientOptions,
    EvaluationRequest,
)


class TestFliptEvaluationClient(unittest.TestCase):
    def setUp(self) -> None:
        url = os.environ.get("FLIPT_URL")
        if url is None:
            raise Exception("FLIPT_URL not set")

        auth_token = os.environ.get("FLIPT_AUTH_TOKEN")
        if auth_token is None:
            raise Exception("FLIPT_AUTH_TOKEN not set")

        self.flipt_client = FliptEvaluationClient(
            opts=ClientOptions(
                url=url,
                authentication=ClientTokenAuthentication(client_token=auth_token),
            )
        )

    def test_variant(self):
        variant = self.flipt_client.evaluate_variant(
            "flag1", "someentity", {"fizz": "buzz"}
        )
        self.assertEqual("flag1", variant.flag_key)
        self.assertTrue(variant.match)
        self.assertEqual("MATCH_EVALUATION_REASON", variant.reason)
        self.assertEqual("variant1", variant.variant_key)
        self.assertIn("segment1", variant.segment_keys)

    def test_boolean(self):
        boolean = self.flipt_client.evaluate_boolean(
            "flag_boolean", "someentity", {"fizz": "buzz"}
        )
        self.assertTrue(boolean.enabled)
        self.assertEqual("flag_boolean", boolean.flag_key)
        self.assertEqual("MATCH_EVALUATION_REASON", boolean.reason)

    def test_list_flags(self):
        flags = self.flipt_client.list_flags()
        self.assertEqual(2, len(flags))

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

        self.assertEqual(3, len(batch.responses))

        variant_response = batch.responses[0]
        self.assertEqual("VARIANT_EVALUATION_RESPONSE_TYPE", variant_response.type)
        variant = variant_response.variant_evaluation_response
        self.assertEqual("flag1", variant.flag_key)
        self.assertTrue(variant.match)
        self.assertEqual("MATCH_EVALUATION_REASON", variant.reason)
        self.assertEqual("variant1", variant.variant_key)
        self.assertIn("segment1", variant.segment_keys)

        boolean_response = batch.responses[1]
        self.assertEqual("BOOLEAN_EVALUATION_RESPONSE_TYPE", boolean_response.type)
        boolean = boolean_response.boolean_evaluation_response
        self.assertTrue(boolean.enabled)
        self.assertEqual("flag_boolean", boolean.flag_key)
        self.assertEqual("MATCH_EVALUATION_REASON", boolean.reason)

        error_response = batch.responses[2]
        self.assertEqual("ERROR_EVALUATION_RESPONSE_TYPE", error_response.type)
        error = error_response.error_evaluation_response
        self.assertEqual("notfound", error.flag_key)
        self.assertEqual("default", error.namespace_key)
        self.assertEqual("NOT_FOUND_ERROR_EVALUATION_REASON", error.reason)

    def test_failure_variant(self):
        with self.assertRaises(Exception) as context:
            self.flipt_client.evaluate_variant(
                "nonexistent", "someentity", {"fizz": "buzz"}
            )

        self.assertEqual(
            "invalid request: failed to get flag information default/nonexistent",
            str(context.exception),
        )

    def test_failure_boolean(self):
        with self.assertRaises(Exception) as context:
            self.flipt_client.evaluate_boolean(
                "nonexistent", "someentity", {"fizz": "buzz"}
            )

        self.assertEqual(
            str(context.exception),
            "invalid request: failed to get flag information default/nonexistent",
        )

    def test_variant_no_context(self):
        variant = self.flipt_client.evaluate_variant("flag1", "someentity")
        self.assertEqual("flag1", variant.flag_key)
        self.assertFalse(variant.match)

    def test_boolean_no_context(self):
        boolean = self.flipt_client.evaluate_boolean("flag_boolean", "someentity")
        self.assertTrue(boolean.enabled)
        self.assertEqual("flag_boolean", boolean.flag_key)


if __name__ == "__main__":
    unittest.main()
