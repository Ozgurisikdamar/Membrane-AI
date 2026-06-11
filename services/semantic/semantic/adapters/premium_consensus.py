"""Tier-3 premium consensus adapter (D-009/D-031).

Two independent frontier models review the masked diff and their findings are
merged into one dual-model consensus: Claude (Anthropic Messages API) and
Gemini (Google generateContent). Both speak their NATIVE wire format here —
unlike the tier-2 vLLM adapter (OpenAI-compatible), Anthropic and Gemini are
called directly via httpx with a strict-JSON instruction.

API keys come from the environment (D-031), matching the service's 12-factor
config and the Helm credentials Secret. The tier runs ONLY when the cost gate
escalates AND the premium flag is on (D-008); the composition root keeps the
DisabledPremiumConsensus stub otherwise, so an enabled-but-unconfigured tier
still fails loud.

Resilience: reviewers run concurrently; a single provider outage degrades to
the survivor's findings (surfaced as an info note), and only an all-providers
failure raises PremiumConsensusError — which the use-case maps to HTTP 503 and
the orchestrator's app.Optional turns into an advisory warning (mirrors D-024).
"""

from __future__ import annotations

import asyncio
import json
from typing import Any, Protocol

import httpx

from semantic.domain.models import EvaluationInput, Finding, Severity

_SYSTEM_PROMPT = (
    "You are a senior security and software-architecture reviewer for MEMBRANE.AI. "
    "Review the given masked diff against the organization's gold-standard context. "
    "Respond with ONLY a JSON object, no prose, of the shape: "
    '{"findings": [{"rule": string, "severity": "info"|"warning"|"blocking", '
    '"message": string}]}. '
    "Report only genuine, high-confidence issues; false positives erode trust."
)

_VALID_SEVERITIES = {s.value for s in Severity}

# Bound the premium response; reviews are short structured JSON.
_MAX_TOKENS = 1024


class PremiumConsensusError(RuntimeError):
    """Raised when EVERY premium reviewer fails (the tier cannot run)."""


class Reviewer(Protocol):
    """One premium model that reviews a diff and returns findings."""

    name: str

    async def review(self, item: EvaluationInput) -> tuple[Finding, ...]:
        """Return this model's findings (already source-tagged)."""
        ...


class AnthropicReviewer:
    """Claude reviewer over the Anthropic Messages API (D-009: Sonnet 4.6)."""

    name = "claude"

    def __init__(
        self,
        api_key: str,
        model: str,
        timeout_seconds: float,
        base_url: str = "https://api.anthropic.com",
    ) -> None:
        self._url = base_url.rstrip("/") + "/v1/messages"
        self._model = model
        self._key = api_key
        self._client = httpx.AsyncClient(timeout=timeout_seconds)

    async def review(self, item: EvaluationInput) -> tuple[Finding, ...]:
        payload = {
            "model": self._model,
            "max_tokens": _MAX_TOKENS,
            "system": _SYSTEM_PROMPT,
            "messages": [{"role": "user", "content": _render_user_prompt(item)}],
        }
        headers = {
            "x-api-key": self._key,
            "anthropic-version": "2023-06-01",
            "content-type": "application/json",
        }
        resp = await self._client.post(self._url, json=payload, headers=headers)
        resp.raise_for_status()
        return _parse_findings(_anthropic_text(_decode(resp, self.name)), self.name)

    async def aclose(self) -> None:
        await self._client.aclose()


class GeminiReviewer:
    """Gemini reviewer over the Google generateContent API (D-009)."""

    name = "gemini"

    def __init__(
        self,
        api_key: str,
        model: str,
        timeout_seconds: float,
        base_url: str = "https://generativelanguage.googleapis.com",
    ) -> None:
        self._base = base_url.rstrip("/")
        self._model = model
        self._key = api_key
        self._client = httpx.AsyncClient(timeout=timeout_seconds)

    async def review(self, item: EvaluationInput) -> tuple[Finding, ...]:
        url = f"{self._base}/v1beta/models/{self._model}:generateContent"
        payload = {
            "system_instruction": {"parts": [{"text": _SYSTEM_PROMPT}]},
            "contents": [{"parts": [{"text": _render_user_prompt(item)}]}],
            # Ask Gemini for raw JSON and deterministic output.
            "generationConfig": {"temperature": 0, "responseMimeType": "application/json"},
        }
        # Key travels in a header, never the URL/query (keeps it out of logs).
        resp = await self._client.post(url, json=payload, headers={"x-goog-api-key": self._key})
        resp.raise_for_status()
        return _parse_findings(_gemini_text(_decode(resp, self.name)), self.name)

    async def aclose(self) -> None:
        await self._client.aclose()


class DualModelConsensus:
    """Implements ports.PremiumConsensus by merging several reviewers.

    Reviewers run concurrently. Partial failure degrades to the survivors'
    findings (each outage surfaced as an info finding); only when ALL reviewers
    fail does the tier raise, so a single provider blip never blocks a verdict.
    """

    def __init__(self, reviewers: tuple[Reviewer, ...], timeout_seconds: float = 30.0) -> None:
        if not reviewers:
            msg = "DualModelConsensus needs at least one reviewer"
            raise ValueError(msg)
        self._reviewers = reviewers
        self._timeout = timeout_seconds

    async def review(self, item: EvaluationInput) -> tuple[Finding, ...]:
        async def bounded(reviewer: Reviewer) -> tuple[Finding, ...]:
            # An overall per-reviewer deadline so a slow-but-not-stalled
            # provider (httpx's read timeout is only per-chunk) can never
            # stretch the consensus past a fixed SLA.
            return await asyncio.wait_for(reviewer.review(item), self._timeout)

        results = await asyncio.gather(
            *(bounded(r) for r in self._reviewers),
            return_exceptions=True,
        )

        findings: list[Finding] = []
        failures: list[tuple[str, BaseException]] = []
        for reviewer, result in zip(self._reviewers, results, strict=True):
            if isinstance(result, asyncio.CancelledError):
                raise result  # cooperative cancellation must propagate, not degrade
            if isinstance(result, BaseException):
                failures.append((reviewer.name, result))
            else:
                findings.extend(result)

        if len(failures) == len(self._reviewers):
            detail = "; ".join(f"{name}: {_summarize_error(err)}" for name, err in failures)
            msg = f"premium consensus: all reviewers failed: {detail}"
            raise PremiumConsensusError(msg)

        for name, err in failures:
            findings.append(
                Finding(
                    rule=f"premium:{name}:unavailable",
                    severity=Severity.INFO,
                    message=f"premium reviewer '{name}' was unavailable ({_summarize_error(err)})",
                )
            )
        return tuple(findings)

    async def aclose(self) -> None:
        """Release every reviewer's HTTP client (called from the app lifespan)."""
        await asyncio.gather(
            *(r.aclose() for r in self._reviewers if hasattr(r, "aclose")),
            return_exceptions=True,
        )


def _render_user_prompt(item: EvaluationInput) -> str:
    parts = [f"Language: {item.language or 'unknown'}"]
    if item.gold_context:
        parts.append("Gold-standard implementations from this organization:")
        for g in item.gold_context:
            parts.append(f"--- {g.file_path}\n{g.code}\n({g.architectural_context})")
    parts.append("Masked diff under review (secrets already redacted):")
    parts.append(item.masked_diff)
    return "\n\n".join(parts)


def _decode(resp: httpx.Response, source: str) -> dict[str, Any]:
    """Decode a 2xx JSON body, converting a non-JSON 200 (an HTML maintenance/
    rate-limit page that raise_for_status won't catch) into PremiumConsensusError."""
    try:
        body = resp.json()
    except ValueError as exc:
        msg = f"{source} response is not valid JSON"
        raise PremiumConsensusError(msg) from exc
    if not isinstance(body, dict):
        msg = f"{source} response is not a JSON object"
        raise PremiumConsensusError(msg)
    return body


def _anthropic_text(body: dict[str, Any]) -> str:
    """Concatenate the text blocks of an Anthropic Messages response."""
    blocks = body.get("content", [])
    if not isinstance(blocks, list):
        msg = "anthropic response has no content blocks"
        raise PremiumConsensusError(msg)
    # Filter on key PRESENCE, not just type, so a `{"type":"text"}` block with
    # no "text" key degrades cleanly instead of raising a bare KeyError.
    texts = [
        b["text"] for b in blocks if isinstance(b, dict) and b.get("type") == "text" and "text" in b
    ]
    if not texts:
        msg = "anthropic response contains no text block"
        raise PremiumConsensusError(msg)
    return "\n".join(str(t) for t in texts)


def _gemini_text(body: dict[str, Any]) -> str:
    """Extract the first candidate's text from a Gemini generateContent response."""
    try:
        parts = body["candidates"][0]["content"]["parts"]
        texts = [p["text"] for p in parts if isinstance(p, dict) and "text" in p]
    except (KeyError, IndexError, TypeError) as exc:
        msg = f"gemini response shape unexpected: {exc}"
        raise PremiumConsensusError(msg) from exc
    if not texts:
        msg = "gemini response contains no text part"
        raise PremiumConsensusError(msg)
    return "\n".join(str(t) for t in texts)


def _summarize_error(err: BaseException) -> str:
    """Stable, non-reflective summary of a reviewer failure for downstream
    findings / the 503 detail — never echoes the request URL (which would name
    the provider/endpoint and, if a key ever moved into the query, leak it)."""
    if isinstance(err, httpx.HTTPStatusError):
        return f"HTTP {err.response.status_code}"
    if isinstance(err, TimeoutError | httpx.TimeoutException):
        return "timeout"
    if isinstance(err, PremiumConsensusError):
        return str(err)  # our own messages carry no URL/key
    if isinstance(err, httpx.HTTPError):
        return "connection error"
    return type(err).__name__


def _parse_findings(content: str, source: str) -> tuple[Finding, ...]:
    """Parse a model's strict-JSON findings, tolerating fences / prose.

    Mirrors the tier-2 parser's fail-safes: malformed entries are skipped,
    unknown severities coerce to warning (surfaced, never dropped), and a
    response with no JSON object raises (handled per-reviewer by the consensus).
    """
    start, end = content.find("{"), content.rfind("}")
    if start < 0 or end <= start:
        msg = f"{source} response contains no JSON object"
        raise PremiumConsensusError(msg)
    try:
        data = json.loads(content[start : end + 1])
    except json.JSONDecodeError as exc:
        msg = f"{source} response is not valid JSON: {exc}"
        raise PremiumConsensusError(msg) from exc

    raw_findings = data.get("findings", [])
    if not isinstance(raw_findings, list):
        msg = f"{source} findings must be a list"
        raise PremiumConsensusError(msg)

    findings: list[Finding] = []
    for raw in raw_findings:
        if not isinstance(raw, dict):
            continue  # skip malformed entries rather than failing the reviewer
        severity = str(raw.get("severity", "warning")).lower()
        if severity not in _VALID_SEVERITIES:
            severity = Severity.WARNING.value  # fail safe: surfaced, never dropped
        rule = str(raw.get("rule", "")).strip() or "unnamed-rule"
        findings.append(
            Finding(
                rule=f"premium:{source}:{rule}",
                severity=Severity(severity),
                message=str(raw.get("message", "")).strip() or "no detail provided",
            )
        )
    return tuple(findings)
