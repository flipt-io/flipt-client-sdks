import os
import unittest
import base64
import json
from datetime import timedelta

from flipt_client import FliptClient
from flipt_client.models import (
    ClientOptions,
    ClientTokenAuthentication,
    EvaluationRequest,
    FlagType,
    TlsConfig,
    model_to_json,
)
from flipt_client.errors import ValidationError, EvaluationError


class TestFliptClient(unittest.TestCase):
    def setUp(self) -> None:
        url = os.environ.get("FLIPT_URL")
        if url is None:
            raise Exception("FLIPT_URL not set")

        auth_token = os.environ.get("FLIPT_AUTH_TOKEN")
        if auth_token is None:
            raise Exception("FLIPT_AUTH_TOKEN not set")

        # Configure TLS if HTTPS URL is provided
        tls_config = None
        if url.startswith("https://"):
            ca_cert_path = os.environ.get("FLIPT_CA_CERT_PATH")
            if ca_cert_path:
                tls_config = TlsConfig(ca_cert_file=ca_cert_path)
            else:
                # Fallback to insecure for local testing
                tls_config = TlsConfig(insecure_skip_verify=True)

        self.flipt_client = FliptClient(
            opts=ClientOptions(
                url=url,
                authentication=ClientTokenAuthentication(client_token=auth_token),
                tls_config=tls_config,
            )
        )

    def tearDown(self) -> None:
        if hasattr(self, "flipt_client") and self.flipt_client is not None:
            self.flipt_client.close()

    def test_null_flag_key(self):
        with self.assertRaises(ValidationError) as context:
            self.flipt_client.evaluate_boolean(None, "someentity", {"fizz": "buzz"})
        self.assertEqual("flag_key cannot be empty or null", str(context.exception))

    def test_empty_flag_key(self):
        with self.assertRaises(ValidationError) as context:
            self.flipt_client.evaluate_boolean("", "someentity", {"fizz": "buzz"})
        self.assertEqual("flag_key cannot be empty or null", str(context.exception))

    def test_null_entity_id(self):
        with self.assertRaises(ValidationError) as context:
            self.flipt_client.evaluate_boolean("flag1", None, {"fizz": "buzz"})
        self.assertEqual("entity_id cannot be empty or null", str(context.exception))

    def test_empty_entity_id(self):
        with self.assertRaises(ValidationError) as context:
            self.flipt_client.evaluate_boolean("flag1", "", {"fizz": "buzz"})
        self.assertEqual("entity_id cannot be empty or null", str(context.exception))

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
        for flag in flags:
            self.assertEqual(flag.description, "flag description")
            if flag.key == "flag1":
                self.assertEqual(flag.type, FlagType.VARIANT)
            elif flag.key == "flag_boolean":
                self.assertEqual(flag.type, FlagType.BOOLEAN)

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
        with self.assertRaises(EvaluationError) as context:
            self.flipt_client.evaluate_variant(
                "nonexistent", "someentity", {"fizz": "buzz"}
            )
        self.assertEqual(
            "invalid request: failed to get flag information default/nonexistent",
            str(context.exception),
        )

    def test_failure_boolean(self):
        with self.assertRaises(EvaluationError) as context:
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

    def test_get_snapshot(self):
        snapshot = self.flipt_client.get_snapshot()
        self.assertIsNotNone(snapshot)

        try:
            decoded = base64.b64decode(snapshot)
            json_obj = json.loads(decoded)
            self.assertIsInstance(json_obj, dict)
        except Exception as e:
            self.fail(f"Snapshot is not valid base64-encoded JSON: {e}")

    def test_set_get_snapshot_with_invalid_url(self):
        # Get a snapshot from a working client
        snapshot = self.flipt_client.get_snapshot()
        self.assertIsNotNone(snapshot)
        self.flipt_client.close()

        # Now create a client with an invalid URL but with the snapshot
        invalid_url = "http://invalid.flipt.com"
        opts = ClientOptions(
            url=invalid_url,
            error_strategy="fallback",
            snapshot=snapshot,
            authentication=ClientTokenAuthentication(
                client_token=os.environ.get("FLIPT_AUTH_TOKEN")
            ),
        )
        client_with_snapshot = FliptClient(opts=opts)
        context = {"fizz": "buzz"}

        # Should be able to evaluate using the snapshot
        for _ in range(3):
            variant = client_with_snapshot.evaluate_variant(
                flag_key="flag1",
                entity_id="someentity",
                context=context,
            )
            self.assertEqual("flag1", variant.flag_key)
            self.assertTrue(variant.match)
            self.assertEqual("MATCH_EVALUATION_REASON", variant.reason)
            self.assertEqual("variant1", variant.variant_key)
            self.assertIn("segment1", variant.segment_keys)

            boolean = client_with_snapshot.evaluate_boolean(
                flag_key="flag_boolean",
                entity_id="someentity",
                context=context,
            )
            self.assertEqual("flag_boolean", boolean.flag_key)
            self.assertTrue(boolean.enabled)
            self.assertEqual("MATCH_EVALUATION_REASON", boolean.reason)

            flags = client_with_snapshot.list_flags()
            self.assertEqual(2, len(flags))

            snapshot2 = client_with_snapshot.get_snapshot()
            self.assertIsNotNone(snapshot2)

        client_with_snapshot.close()

    def test_client_options_serialization_update_interval_timedelta(self):
        """Test that ClientOptions serializes update_interval timedelta correctly"""
        opts = ClientOptions(
            url="http://localhost:8080",
            update_interval=timedelta(seconds=10),
        )
        
        # Serialize to JSON
        json_str = model_to_json(opts, exclude_none=True)
        json_obj = json.loads(json_str)
        
        # Should serialize timedelta to seconds as int
        self.assertEqual(json_obj["update_interval"], 10)
        
    def test_client_options_serialization_update_interval_int(self):
        """Test that ClientOptions serializes update_interval int correctly"""
        opts = ClientOptions(
            url="http://localhost:8080",
            update_interval=10,  # This gets converted to timedelta internally
        )
        
        # Serialize to JSON
        json_str = model_to_json(opts, exclude_none=True)
        json_obj = json.loads(json_str)
        
        # Should serialize to seconds as int
        self.assertEqual(json_obj["update_interval"], 10)
        
    def test_client_options_serialization_update_interval_none(self):
        """Test that ClientOptions excludes None update_interval"""
        opts = ClientOptions(
            url="http://localhost:8080",
            update_interval=None,
        )
        
        # Serialize to JSON excluding None values
        json_str = model_to_json(opts, exclude_none=True)
        json_obj = json.loads(json_str)
        
        # update_interval should not be present in the JSON
        self.assertNotIn("update_interval", json_obj)
        
    def test_client_options_serialization_update_interval_default(self):
        """Test that ClientOptions with no update_interval excludes it"""
        opts = ClientOptions(
            url="http://localhost:8080",
        )
        
        # Serialize to JSON excluding None values
        json_str = model_to_json(opts, exclude_none=True)
        json_obj = json.loads(json_str)
        
        # update_interval should not be present in the JSON
        self.assertNotIn("update_interval", json_obj)
