"""Tier-3 premium consensus adapters (httpx MockTransport, no network)."""

import asyncio
import json

import httpx
import pytest

from semantic.adapters.premium_consensus import (
    AnthropicReviewer,
    DualModelConsensus,
    GeminiReviewer,
    PremiumConsensusError,
    Reviewer,
)
from semantic.domain.models import EvaluationInput, Finding, Severity


def _input() -> EvaluationInput:
    return EvaluationInput(submission_id="s", organization_id="o", language="go", masked_diff="+x")


def _swap(reviewer, handler):  # type: ignore[no-untyped-def]
    reviewer._client = httpx.AsyncClient(transport=httpx.MockTransport(handler))
    return reviewer


def _anthropic_body(content: str) -> dict:
    return {"content": [{"type": "text", "text": content}]}


def _gemini_body(content: str) -> dict:
    return {"candidates": [{"content": {"parts": [{"text": content}]}}]}


def test_anthropic_reviewer_parses_and_tags():
    captured = {}

    def handler(request: httpx.Request) -> httpx.Response:
        captured["url"] = str(request.url)
        captured["key"] = request.headers.get("x-api-key")
        captured["version"] = request.headers.get("anthropic-version")
        captured["body"] = json.loads(request.content)
        return httpx.Response(
            200,
            json=_anthropic_body(
                '{"findings":[{"rule":"sqli","severity":"blocking","message":"m"}]}'
            ),
        )

    reviewer = _swap(AnthropicReviewer("sk-test", "claude-sonnet-4-6", 5.0), handler)
    findings = asyncio.run(reviewer.review(_input()))

    assert captured["url"] == "https://api.anthropic.com/v1/messages"
    assert captured["key"] == "sk-test"
    assert captured["version"] == "2023-06-01"
    assert captured["body"]["model"] == "claude-sonnet-4-6"
    assert findings[0].rule == "premium:claude:sqli"
    assert findings[0].severity is Severity.BLOCKING


def test_gemini_reviewer_parses_and_tags():
    captured = {}

    def handler(request: httpx.Request) -> httpx.Response:
        captured["url"] = str(request.url)
        captured["key"] = request.headers.get("x-goog-api-key")
        return httpx.Response(
            200,
            json=_gemini_body('{"findings":[{"rule":"xss","severity":"warning","message":"m"}]}'),
        )

    reviewer = _swap(GeminiReviewer("g-test", "gemini-2.5-pro", 5.0), handler)
    findings = asyncio.run(reviewer.review(_input()))

    assert "gemini-2.5-pro:generateContent" in captured["url"]
    assert captured["key"] == "g-test"
    assert "key=" not in captured["url"], "the API key must not travel in the URL"
    assert findings[0].rule == "premium:gemini:xss"
    assert findings[0].severity is Severity.WARNING


def test_parse_tolerates_fences_and_coerces_severity():
    def handler(_: httpx.Request) -> httpx.Response:
        return httpx.Response(
            200,
            json=_anthropic_body(
                "Sure:\n```json\n"
                '{"findings":[{"rule":"x","severity":"CRITICAL","message":"m"},"junk"]}\n```'
            ),
        )

    reviewer = _swap(AnthropicReviewer("k", "claude-sonnet-4-6", 5.0), handler)
    findings = asyncio.run(reviewer.review(_input()))
    assert len(findings) == 1, "malformed entries are skipped"
    assert findings[0].severity is Severity.WARNING, "unknown severity fails safe"


def test_reviewer_http_error_propagates():
    def handler(_: httpx.Request) -> httpx.Response:
        return httpx.Response(429, text="rate limited")

    reviewer = _swap(AnthropicReviewer("k", "claude-sonnet-4-6", 5.0), handler)
    with pytest.raises(httpx.HTTPStatusError):
        asyncio.run(reviewer.review(_input()))


def test_anthropic_text_block_missing_text_key_degrades():
    # A {"type":"text"} block with no "text" key must not crash with KeyError.
    def handler(_: httpx.Request) -> httpx.Response:
        return httpx.Response(200, json={"content": [{"type": "text"}]})

    reviewer = _swap(AnthropicReviewer("k", "claude-sonnet-4-6", 5.0), handler)
    with pytest.raises(PremiumConsensusError):
        asyncio.run(reviewer.review(_input()))


def test_non_json_200_body_raises_premium_error():
    # An HTML maintenance page served with 200 (raise_for_status won't catch it).
    def handler(_: httpx.Request) -> httpx.Response:
        return httpx.Response(200, text="<html>down for maintenance</html>")

    reviewer = _swap(GeminiReviewer("k", "gemini-2.5-pro", 5.0), handler)
    with pytest.raises(PremiumConsensusError, match="not valid JSON"):
        asyncio.run(reviewer.review(_input()))


class _StubReviewer:
    """In-memory Reviewer for consensus tests (no HTTP)."""

    def __init__(self, name: str, findings: tuple[Finding, ...] | None, error: Exception | None):
        self.name = name
        self._findings = findings
        self._error = error

    async def review(self, item: EvaluationInput) -> tuple[Finding, ...]:
        if self._error is not None:
            raise self._error
        assert self._findings is not None
        return self._findings


def _finding(rule: str) -> Finding:
    return Finding(rule=rule, severity=Severity.WARNING, message="m")


def test_consensus_merges_all_reviewers():
    reviewers: tuple[Reviewer, ...] = (
        _StubReviewer("claude", (_finding("premium:claude:a"),), None),
        _StubReviewer("gemini", (_finding("premium:gemini:b"),), None),
    )
    findings = asyncio.run(DualModelConsensus(reviewers).review(_input()))
    rules = {f.rule for f in findings}
    assert rules == {"premium:claude:a", "premium:gemini:b"}


def test_consensus_degrades_on_partial_failure():
    reviewers: tuple[Reviewer, ...] = (
        _StubReviewer("claude", (_finding("premium:claude:a"),), None),
        _StubReviewer("gemini", None, RuntimeError("boom")),
    )
    findings = asyncio.run(DualModelConsensus(reviewers).review(_input()))
    rules = {f.rule for f in findings}
    assert "premium:claude:a" in rules
    # the survivor answers AND the outage is surfaced, never silently dropped
    assert "premium:gemini:unavailable" in rules
    unavailable = next(f for f in findings if f.rule == "premium:gemini:unavailable")
    assert unavailable.severity is Severity.INFO


def test_consensus_raises_when_all_fail():
    reviewers: tuple[Reviewer, ...] = (
        _StubReviewer("claude", None, RuntimeError("a-down")),
        _StubReviewer("gemini", None, RuntimeError("b-down")),
    )
    with pytest.raises(PremiumConsensusError, match="all reviewers failed"):
        asyncio.run(DualModelConsensus(reviewers).review(_input()))


def test_consensus_requires_a_reviewer():
    with pytest.raises(ValueError, match="at least one reviewer"):
        DualModelConsensus(())


def test_consensus_bounds_a_slow_reviewer():
    class _SlowReviewer:
        name = "slow"

        async def review(self, item: EvaluationInput) -> tuple[Finding, ...]:
            await asyncio.sleep(10)
            return ()

    reviewers: tuple[Reviewer, ...] = (
        _StubReviewer("fast", (_finding("premium:fast:a"),), None),
        _SlowReviewer(),
    )
    # A 0.05s overall deadline: the slow reviewer is dropped, the fast one answers.
    findings = asyncio.run(DualModelConsensus(reviewers, timeout_seconds=0.05).review(_input()))
    rules = {f.rule for f in findings}
    assert "premium:fast:a" in rules
    unavailable = next(f for f in findings if f.rule == "premium:slow:unavailable")
    assert unavailable.message.endswith("(timeout)"), "slow provider summarized as a timeout"


def test_unavailable_message_does_not_echo_request_url():
    # An httpx error carries the full request URL; the surfaced finding must not.
    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(429, text="rate limited")

    reviewer = _swap(GeminiReviewer("secret-key", "gemini-2.5-pro", 5.0), handler)
    findings = asyncio.run(
        DualModelConsensus(
            (_StubReviewer("claude", (_finding("premium:claude:a"),), None), reviewer)
        ).review(_input())
    )
    note = next(f for f in findings if f.rule == "premium:gemini:unavailable")
    assert "googleapis.com" not in note.message, "must not echo the provider URL"
    assert "secret-key" not in note.message, "must never echo the key"
    assert note.message.endswith("(HTTP 429)"), "summarized to a stable status"
