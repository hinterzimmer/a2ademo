import unittest
from unittest.mock import patch, MagicMock
import json
import base64
import sys
import os

# Mock boto3 and botocore before importing index so tests run without external packages
if "boto3" not in sys.modules:
    mock_boto3 = MagicMock()
    sys.modules["boto3"] = mock_boto3

if "botocore.exceptions" not in sys.modules:
    mock_botocore = MagicMock()
    mock_exceptions = MagicMock()
    class ClientError(Exception):
        pass
    mock_exceptions.ClientError = ClientError
    sys.modules["botocore"] = mock_botocore
    sys.modules["botocore.exceptions"] = mock_exceptions

# Ensure lambda_src is on sys.path
sys.path.insert(0, os.path.dirname(__file__))

import index


class TestLambdaA2AHandler(unittest.TestCase):

    def test_normalize_session_id_collision_free_and_bounds(self):
        # 1. Uniqueness / no collisions
        s1 = index.normalize_session_id("audit-sess-1")
        s2 = index.normalize_session_id("audit-sess-10")
        s3 = index.normalize_session_id("audit-sess-100")
        s4 = index.normalize_session_id("a.b")
        s5 = index.normalize_session_id("a_b")

        self.assertNotEqual(s1, s2)
        self.assertNotEqual(s2, s3)
        self.assertNotEqual(s4, s5)

        # 2. Length constraint: 33 <= len <= 100
        for sid in [s1, s2, s3, s4, s5]:
            self.assertTrue(33 <= len(sid) <= 100, f"Length {len(sid)} out of bounds for {sid}")
            self.assertTrue(sid[0].isalnum(), f"Session ID {sid} must start with alphanumeric")
            self.assertTrue(all(c.isalnum() or c in "_-" for c in sid), f"Invalid characters in {sid}")

        # 3. None / empty generates valid unique ID
        empty_id1 = index.normalize_session_id(None)
        empty_id2 = index.normalize_session_id("")
        self.assertTrue(33 <= len(empty_id1) <= 100)
        self.assertTrue(33 <= len(empty_id2) <= 100)
        self.assertNotEqual(empty_id1, empty_id2)

    def test_scrub_aws_details(self):
        raw_error = "ValidationException: ARN arn:aws:bedrock-agentcore:eu-central-1:123456789012:harness/my-harness is invalid"
        scrubbed = index.scrub_aws_details(raw_error)
        self.assertNotIn("arn:aws:", scrubbed)
        self.assertNotIn("123456789012", scrubbed)
        self.assertIn("[REDACTED_ARN]", scrubbed)

    def test_routing_404_on_unknown_path(self):
        event = {
            "httpMethod": "GET",
            "path": "/unknown-endpoint"
        }
        resp = index.handler(event, None)
        self.assertEqual(resp["statusCode"], 404)

    def test_routing_405_on_get_rpc(self):
        event = {
            "httpMethod": "GET",
            "path": "/rpc"
        }
        resp = index.handler(event, None)
        self.assertEqual(resp["statusCode"], 405)

    def test_agent_card_discovery(self):
        event = {
            "httpMethod": "GET",
            "path": "/.well-known/agent-card.json"
        }
        resp = index.handler(event, None)
        self.assertEqual(resp["statusCode"], 200)
        body = json.loads(resp["body"])
        self.assertEqual(body["name"], "SecurityAuditorAgent")
        self.assertEqual(body["protocol_version"], "0.2.0")
        self.assertGreaterEqual(len(body["skills"]), 3)

    def test_jsonrpc_parse_error(self):
        event = {
            "httpMethod": "POST",
            "path": "/rpc",
            "body": "{invalid-json"
        }
        resp = index.handler(event, None)
        self.assertEqual(resp["statusCode"], 200)
        body = json.loads(resp["body"])
        self.assertEqual(body["jsonrpc"], "2.0")
        self.assertIsNone(body["id"])
        self.assertEqual(body["error"]["code"], -32700)

    def test_jsonrpc_invalid_request_version(self):
        event = {
            "httpMethod": "POST",
            "path": "/rpc",
            "body": json.dumps({"jsonrpc": "1.0", "method": "message/send", "params": {}, "id": "req-1"})
        }
        resp = index.handler(event, None)
        body = json.loads(resp["body"])
        self.assertEqual(body["id"], "req-1")
        self.assertEqual(body["error"]["code"], -32600)

    def test_jsonrpc_method_not_found(self):
        event = {
            "httpMethod": "POST",
            "path": "/rpc",
            "body": json.dumps({"jsonrpc": "2.0", "method": "unsupported/method", "params": {}, "id": "req-2"})
        }
        resp = index.handler(event, None)
        body = json.loads(resp["body"])
        self.assertEqual(body["id"], "req-2")
        self.assertEqual(body["error"]["code"], -32601)

    def test_jsonrpc_invalid_params(self):
        # Missing params
        event = {
            "httpMethod": "POST",
            "path": "/rpc",
            "body": json.dumps({"jsonrpc": "2.0", "method": "message/send", "id": "req-3"})
        }
        resp = index.handler(event, None)
        body = json.loads(resp["body"])
        self.assertEqual(body["error"]["code"], -32602)

        # Missing message in params
        event["body"] = json.dumps({"jsonrpc": "2.0", "method": "message/send", "params": {}, "id": "req-3"})
        resp = index.handler(event, None)
        body = json.loads(resp["body"])
        self.assertEqual(body["error"]["code"], -32602)

        # Empty parts in message
        event["body"] = json.dumps({
            "jsonrpc": "2.0",
            "method": "message/send",
            "params": {"message": {"parts": []}},
            "id": "req-3"
        })
        resp = index.handler(event, None)
        body = json.loads(resp["body"])
        self.assertEqual(body["error"]["code"], -32602)

    def test_base64_encoded_body(self):
        raw_payload = json.dumps({
            "jsonrpc": "2.0",
            "method": "message/send",
            "params": {"message": {"parts": [{"text": "Ping"}]}},
            "id": 101
        })
        b64_payload = base64.b64encode(raw_payload.encode("utf-8")).decode("utf-8")

        event = {
            "httpMethod": "POST",
            "path": "/rpc",
            "isBase64Encoded": True,
            "body": b64_payload
        }

        with patch("index.invoke_agent_reasoning", return_value="Pong analysis complete"):
            resp = index.handler(event, None)
            self.assertEqual(resp["statusCode"], 200)
            body = json.loads(resp["body"])
            self.assertEqual(body["jsonrpc"], "2.0")
            self.assertEqual(body["id"], 101)
            self.assertEqual(body["result"]["parts"][0]["text"], "Pong analysis complete")
            self.assertTrue(body["result"]["messageId"].startswith("msg-"))

    def test_multi_turn_context_id_preservation(self):
        raw_payload = json.dumps({
            "jsonrpc": "2.0",
            "method": "message/send",
            "params": {
                "message": {
                    "parts": [{"text": "Remember secret TOKEN-99"}],
                    "contextId": "custom-sess-42"
                }
            },
            "id": "req-turn-1"
        })

        event = {
            "httpMethod": "POST",
            "path": "/rpc",
            "body": raw_payload
        }

        with patch("index.invoke_agent_reasoning", return_value="Noted secret TOKEN-99") as mock_invoke:
            resp = index.handler(event, None)
            body = json.loads(resp["body"])
            self.assertEqual(body["id"], "req-turn-1")
            self.assertEqual(body["result"]["contextId"], "custom-sess-42")
            expected_sid = index.normalize_session_id("custom-sess-42")
            mock_invoke.assert_called_once_with("Remember secret TOKEN-99", expected_sid)


if __name__ == "__main__":
    unittest.main()
