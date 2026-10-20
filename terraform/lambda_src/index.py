import json
import logging
import os
import uuid
import base64
import hashlib
import re
import datetime
import boto3
from botocore.exceptions import ClientError

logger = logging.getLogger()
logger.setLevel(logging.INFO)

AWS_REGION = os.environ.get("AWS_REGION", "eu-central-1")
HARNESS_ARN = os.environ.get("HARNESS_ARN", "")
MODEL_ID = os.environ.get("MODEL_ID", "eu.anthropic.claude-haiku-4-5-20251001-v1:0")
AGENT_NAME = os.environ.get("AGENT_NAME", "SecurityAuditorAgent")
AGENT_DESCRIPTION = os.environ.get(
    "AGENT_DESCRIPTION",
    "Autonomous cloud-deployed A2A agent running on AWS Bedrock Harness for zero-trust security auditing, protocol verification, and compliance."
)

bedrock_agentcore = boto3.client("bedrock-agentcore", region_name=AWS_REGION)
bedrock_runtime = boto3.client("bedrock-runtime", region_name=AWS_REGION)


def scrub_aws_details(text: str) -> str:
    """Removes sensitive AWS ARNs and Account IDs from error messages."""
    text = re.sub(r'arn:aws:[^:\s]+:[^:\s]+:\d+:[^\s]+', '[REDACTED_ARN]', text)
    text = re.sub(r'\b\d{12}\b', '[REDACTED_ACCOUNT]', text)
    return text


def normalize_session_id(raw_id: str | None) -> str:
    """
    Normalizes a caller-supplied context ID into a deterministic, collision-resistant
    session ID compliant with Bedrock AgentCore InvokeHarness:
    - Must start with an alphanumeric character
    - Length must be between 33 and 100 characters
    - Valid characters: [a-zA-Z0-9_-]
    """
    if not raw_id:
        return f"sess_{uuid.uuid4().hex}"
    digest = hashlib.sha256(raw_id.encode("utf-8")).hexdigest()
    # "sess_" (5) + 44 hex characters = exactly 49 characters
    return f"sess_{digest[:44]}"


def get_agent_card(base_url: str) -> dict:
    return {
        "name": AGENT_NAME,
        "description": AGENT_DESCRIPTION,
        "url": base_url,
        "version": "1.0.0",
        "protocol_version": "0.2.0",
        "skills": [
            {
                "id": "security_audit",
                "name": "Security Audit",
                "description": "Performs zero-trust threat modeling and security posture analysis on multi-agent architectures.",
                "tags": ["security", "audit", "threat-modeling"]
            },
            {
                "id": "protocol_verification",
                "name": "Protocol Verification",
                "description": "Validates A2A protocol message safety, sandbox integrity, and isolation boundaries.",
                "tags": ["a2a", "verification", "sandbox"]
            },
            {
                "id": "compliance_certification",
                "name": "Compliance Certification",
                "description": "Audits distributed agent communication against enterprise security and governance standards.",
                "tags": ["compliance", "governance", "soc2"]
            }
        ]
    }


def make_rpc_error(req_id: str | int | None, code: int, message: str) -> dict:
    return {
        "jsonrpc": "2.0",
        "error": {
            "code": code,
            "message": scrub_aws_details(message)
        },
        "id": req_id
    }


def invoke_agent_reasoning(user_prompt: str, session_id: str) -> str:
    system_prompt = (
        "You are the SecurityAuditorAgent, an autonomous Cloud & Protocol Security Auditor operating in an A2A protocol network. "
        "Your mission is to perform rigorous zero-trust threat modeling, evaluate protocol boundaries and isolation, "
        "identify vulnerabilities (prompt injection, unauthorized delegation, data exfiltration), "
        "and formulate concrete remediation controls and security audit sign-offs."
    )

    # 1. Invoke AWS Bedrock AgentCore Harness if configured (fail-fast, no silent fallback)
    if HARNESS_ARN:
        logger.info(f"Invoking Bedrock Harness: {HARNESS_ARN} (session: {session_id})")
        messages = [
            {
                "role": "user",
                "content": [{"text": user_prompt}]
            }
        ]
        try:
            response = bedrock_agentcore.invoke_harness(
                harnessArn=HARNESS_ARN,
                runtimeSessionId=session_id,
                messages=messages
            )
            stream = response.get("stream")
            if not stream:
                raise RuntimeError("Bedrock Harness did not return an event stream")

            output_parts = []
            for event in stream:
                if "contentBlockDelta" in event:
                    delta = event["contentBlockDelta"].get("delta", {})
                    if "text" in delta:
                        output_parts.append(delta["text"])
            if output_parts:
                return "".join(output_parts)
            raise RuntimeError("Bedrock Harness returned an empty stream")
        except Exception as e:
            logger.error(f"Bedrock Harness invocation failed: {e}")
            raise RuntimeError(f"Bedrock Harness execution failed: {str(e)}")

    # 2. Direct AWS Bedrock Runtime Converse API (only when HARNESS_ARN is unset)
    logger.info(f"Invoking Bedrock Converse API with model: {MODEL_ID}")
    try:
        response = bedrock_runtime.converse(
            modelId=MODEL_ID,
            messages=[
                {
                    "role": "user",
                    "content": [{"text": user_prompt}]
                }
            ],
            system=[{"text": system_prompt}],
            inferenceConfig={"maxTokens": 2048, "temperature": 0.2}
        )
        output_msg = response.get("output", {}).get("message", {})
        content_blocks = output_msg.get("content", [])
        text_outputs = [b.get("text", "") for b in content_blocks if "text" in b]
        return "".join(text_outputs)
    except Exception as e:
        logger.error(f"Bedrock Converse failed: {e}")
        raise RuntimeError(f"Bedrock LLM inference failed: {str(e)}")


def handler(event, context):
    http_context = event.get("requestContext", {}).get("http", {})
    method = http_context.get("method") or event.get("httpMethod") or "GET"
    raw_path = http_context.get("path") or event.get("rawPath") or event.get("path") or "/"
    path = raw_path.rstrip("/") if raw_path != "/" else "/"

    domain = event.get("requestContext", {}).get("domainName") or ""
    base_url = f"https://{domain}" if domain else "http://localhost:8000"

    headers = {
        "Content-Type": "application/json",
        "Access-Control-Allow-Origin": "*",
        "Access-Control-Allow-Methods": "GET, POST, OPTIONS",
        "Access-Control-Allow-Headers": "Content-Type, Authorization"
    }

    if method == "OPTIONS":
        return {
            "statusCode": 200,
            "headers": headers,
            "body": ""
        }

    # 1. Handle AgentCard Discovery
    if method == "GET" and (path in ["/.well-known/agent-card.json", "/agent-card.json"] or path.endswith("/.well-known/agent-card.json")):
        card = get_agent_card(base_url)
        return {
            "statusCode": 200,
            "headers": headers,
            "body": json.dumps(card, indent=2)
        }

    # 2. HTTP Method Validation for /rpc
    if path in ["/rpc", "/rpc/"] or path.endswith("/rpc"):
        if method != "POST":
            return {
                "statusCode": 405,
                "headers": headers,
                "body": json.dumps({"error": f"Method {method} not allowed on /rpc; use POST for JSON-RPC 2.0"})
            }

        raw_body = event.get("body", "{}")
        if event.get("isBase64Encoded"):
            try:
                raw_body = base64.b64decode(raw_body).decode("utf-8")
            except Exception as e:
                return {
                    "statusCode": 200,
                    "headers": headers,
                    "body": json.dumps(make_rpc_error(None, -32700, f"Parse error: invalid base64 body ({str(e)})"))
                }

        try:
            body = json.loads(raw_body) if isinstance(raw_body, str) else (raw_body or {})
        except json.JSONDecodeError as e:
            return {
                "statusCode": 200,
                "headers": headers,
                "body": json.dumps(make_rpc_error(None, -32700, f"Parse error: invalid JSON ({str(e)})"))
            }

        req_id = body.get("id")
        jsonrpc = body.get("jsonrpc")
        rpc_method = body.get("method")

        if jsonrpc != "2.0":
            return {
                "statusCode": 200,
                "headers": headers,
                "body": json.dumps(make_rpc_error(req_id, -32600, "Invalid Request: jsonrpc must be '2.0'"))
            }

        if rpc_method not in ["message/send", "send_message"]:
            return {
                "statusCode": 200,
                "headers": headers,
                "body": json.dumps(make_rpc_error(req_id, -32601, f"Method not found: '{rpc_method}' is not supported (expected 'message/send' or 'send_message')"))
            }

        params = body.get("params")
        if not isinstance(params, dict):
            return {
                "statusCode": 200,
                "headers": headers,
                "body": json.dumps(make_rpc_error(req_id, -32602, "Invalid params: 'params' must be an object"))
            }

        message = params.get("message")
        if not isinstance(message, dict):
            return {
                "statusCode": 200,
                "headers": headers,
                "body": json.dumps(make_rpc_error(req_id, -32602, "Invalid params: missing required 'message' object"))
            }

        parts = message.get("parts")
        if not isinstance(parts, list) or len(parts) == 0:
            return {
                "statusCode": 200,
                "headers": headers,
                "body": json.dumps(make_rpc_error(req_id, -32602, "Invalid params: 'message.parts' must be a non-empty list"))
            }

        # Extract contextId from camelCase or snake_case
        context_data = params.get("context", {}) if isinstance(params.get("context"), dict) else {}
        raw_context_id = (
            message.get("contextId")
            or message.get("context_id")
            or params.get("contextId")
            or params.get("context_id")
            or context_data.get("contextId")
            or context_data.get("context_id")
            or ""
        )
        session_id = normalize_session_id(raw_context_id)

        user_prompt = ""
        for part in parts:
            if isinstance(part, dict) and "text" in part:
                user_prompt += part["text"] + " "
            elif isinstance(part, str):
                user_prompt += part + " "
        user_prompt = user_prompt.strip()

        if not user_prompt:
            return {
                "statusCode": 200,
                "headers": headers,
                "body": json.dumps(make_rpc_error(req_id, -32602, "Invalid params: message text parts cannot be empty"))
            }

        try:
            agent_response_text = invoke_agent_reasoning(user_prompt, session_id)
        except Exception as e:
            return {
                "statusCode": 200,
                "headers": headers,
                "body": json.dumps(make_rpc_error(req_id, -32603, f"Internal error: {str(e)}"))
            }

        response_payload = {
            "jsonrpc": "2.0",
            "result": {
                "role": "agent",
                "parts": [{"text": agent_response_text}],
                "messageId": f"msg-{uuid.uuid4().hex[:12]}",
                "contextId": raw_context_id or session_id,
                "timestamp": datetime.datetime.now(datetime.timezone.utc).isoformat()
            },
            "id": req_id
        }

        return {
            "statusCode": 200,
            "headers": headers,
            "body": json.dumps(response_payload)
        }

    # 3. Handle default info root
    if path == "/":
        return {
            "statusCode": 200,
            "headers": headers,
            "body": json.dumps({
                "service": "A2A Harness Agent",
                "agent": AGENT_NAME,
                "agent_card": f"{base_url}/.well-known/agent-card.json",
                "rpc_endpoint": f"{base_url}/rpc",
                "status": "online"
            })
        }

    # 4. Unknown routes -> 404
    return {
        "statusCode": 404,
        "headers": headers,
        "body": json.dumps({"error": f"Path '{raw_path}' not found"})
    }
