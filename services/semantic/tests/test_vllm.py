"""VLLMLocalModel (httpx MockTransport) and the fallback decorator."""

import asyncio
import json

import httpx
import pytest

from semantic.adapters.fallback import FallbackLocalModel
from semantic.adapters.local_stub import HeuristicLocalModel
from semantic.adapters.vllm_local import VLLMError, VLLMLocalModel
from semantic.domain.models import EvaluationInput, LocalAssessment, Severity


def _input() -> EvaluationInput:
    return EvaluationInput(submission_id="s", organization_id="o", language="go", masked_diff="+x")


def make_model(handler) -> VLLMLocalModel:
    model = VLLMLocalModel("http://vllm.local", "deepseek-coder", 5.0)
    # Swap the transport so no network is touched.
    model._client = httpx.AsyncClient(transport=httpx.MockTransport(handler))
    return model


def chat_response(content: str) -> httpx.Response:
    return httpx.Response(200, json={"choices": [{"message": {"content": content}}]})


def test_assess_parses_strict_json():
    captured = {}

    def handler(request: httpx.Request) -> httpx.Response:
        captured["url"] = str(request.url)
        captured["body"] = json.loads(request.content)
        return chat_response(
            '{"findings":[{"rule":"sql-injection","severity":"blocking","message":"m"}],'
            '"risk_score":0.8}'
        )

    model = make_model(handler)
    result = asyncio.run(model.assess(_input()))

    assert captured["url"] == "http://vllm.local/v1/chat/completions"
    assert captured["body"]["model"] == "deepseek-coder"
    assert result.risk_score == 0.8
    assert result.findings[0].rule == "vllm:sql-injection"
    assert result.findings[0].severity is Severity.BLOCKING


def test_assess_tolerates_fences_and_coerces():
    def handler(_: httpx.Request) -> httpx.Response:
        return chat_response(
            "Here is my review:\n```json\n"
            '{"findings":[{"rule":"x","severity":"CRITICAL","message":"m"},"junk"],'
            '"risk_score":7}\n```'
        )

    result = asyncio.run(make_model(handler).assess(_input()))
    assert result.risk_score == 1.0, "score must clamp to [0,1]"
    assert len(result.findings) == 1, "malformed entries are skipped"
    assert result.findings[0].severity is Severity.WARNING, "unknown severity fails safe"


@pytest.mark.parametrize(
    "content",
    ["no json here at all", '{"findings": "not-a-list", "risk_score": 0}'],
)
def test_assess_unusable_response_raises(content: str):
    def handler(_: httpx.Request) -> httpx.Response:
        return chat_response(content)

    with pytest.raises(VLLMError):
        asyncio.run(make_model(handler).assess(_input()))


def test_assess_http_error_raises():
    def handler(_: httpx.Request) -> httpx.Response:
        return httpx.Response(503, text="overloaded")

    with pytest.raises(VLLMError, match="vllm call failed"):
        asyncio.run(make_model(handler).assess(_input()))


def test_fallback_degrades_to_heuristic():
    def handler(_: httpx.Request) -> httpx.Response:
        return httpx.Response(503)

    chain = FallbackLocalModel(primary=make_model(handler), backup=HeuristicLocalModel())
    result = asyncio.run(chain.assess(_input()))

    assert isinstance(result, LocalAssessment), "backup must answer"
    assert chain.last_primary_error is not None


def test_fallback_prefers_primary_when_healthy():
    def handler(_: httpx.Request) -> httpx.Response:
        return chat_response('{"findings":[],"risk_score":0.42}')

    chain = FallbackLocalModel(primary=make_model(handler), backup=HeuristicLocalModel())
    result = asyncio.run(chain.assess(_input()))

    assert result.risk_score == 0.42
    assert chain.last_primary_error is None
