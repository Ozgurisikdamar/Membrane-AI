"""Tier-2 local model adapter against an OpenAI-compatible vLLM endpoint.

The local model pool (report: DeepSeek-Coder/CodeLlama on vLLM) exposes the
OpenAI chat-completions API; this adapter prompts it for a STRICT-JSON review
of the masked diff and parses the result into the domain. Any transport or
parse failure raises VLLMError — the composition root wraps this adapter with
FallbackLocalModel so the heuristic stub keeps the tier alive (D-028).
"""

from __future__ import annotations

import json
from typing import Any

import httpx

from semantic.domain.models import EvaluationInput, Finding, LocalAssessment, Severity

_SYSTEM_PROMPT = (
    "You are MEMBRANE.AI's code-security and architecture reviewer. "
    "Review the given masked diff against the organization's gold-standard context. "
    "Respond with ONLY a JSON object, no prose, of the shape: "
    '{"findings": [{"rule": string, "severity": "info"|"warning"|"blocking", '
    '"message": string}], "risk_score": number between 0 and 1}. '
    "Report only genuine issues; false positives erode trust."
)

_VALID_SEVERITIES = {s.value for s in Severity}


class VLLMError(RuntimeError):
    """Raised when the vLLM endpoint fails or returns an unusable response."""


class VLLMLocalModel:
    """Implements ports.LocalModel against /v1/chat/completions."""

    def __init__(self, base_url: str, model: str, timeout_seconds: float) -> None:
        self._url = base_url.rstrip("/") + "/v1/chat/completions"
        self._model = model
        self._client = httpx.AsyncClient(timeout=timeout_seconds)

    async def assess(self, item: EvaluationInput) -> LocalAssessment:
        payload = {
            "model": self._model,
            "temperature": 0,
            "messages": [
                {"role": "system", "content": _SYSTEM_PROMPT},
                {"role": "user", "content": _render_user_prompt(item)},
            ],
        }
        try:
            resp = await self._client.post(self._url, json=payload)
            resp.raise_for_status()
            body: dict[str, Any] = resp.json()
            content = body["choices"][0]["message"]["content"]
        except (httpx.HTTPError, KeyError, IndexError, TypeError, ValueError) as exc:
            msg = f"vllm call failed: {exc}"
            raise VLLMError(msg) from exc
        return _parse_assessment(str(content))

    async def aclose(self) -> None:
        """Release the underlying HTTP client."""
        await self._client.aclose()


def _render_user_prompt(item: EvaluationInput) -> str:
    parts = [f"Language: {item.language or 'unknown'}"]
    if item.gold_context:
        parts.append("Gold-standard implementations from this organization:")
        for g in item.gold_context:
            parts.append(f"--- {g.file_path}\n{g.code}\n({g.architectural_context})")
    parts.append("Masked diff under review (secrets already redacted):")
    parts.append(item.masked_diff)
    return "\n\n".join(parts)


def _parse_assessment(content: str) -> LocalAssessment:
    """Parse the model's JSON (tolerating markdown fences / surrounding prose)."""
    start, end = content.find("{"), content.rfind("}")
    if start < 0 or end <= start:
        msg = "vllm response contains no JSON object"
        raise VLLMError(msg)
    try:
        data = json.loads(content[start : end + 1])
    except json.JSONDecodeError as exc:
        msg = f"vllm response is not valid JSON: {exc}"
        raise VLLMError(msg) from exc

    raw_findings = data.get("findings", [])
    if not isinstance(raw_findings, list):
        msg = "vllm findings must be a list"
        raise VLLMError(msg)
    findings: list[Finding] = []
    for raw in raw_findings:
        if not isinstance(raw, dict):
            continue  # skip malformed entries rather than failing the tier
        severity = str(raw.get("severity", "warning")).lower()
        if severity not in _VALID_SEVERITIES:
            severity = Severity.WARNING.value  # fail safe: surfaced, never dropped
        rule = str(raw.get("rule", "")).strip() or "vllm-unnamed-rule"
        findings.append(
            Finding(
                rule=f"vllm:{rule}",
                severity=Severity(severity),
                message=str(raw.get("message", "")).strip() or "no detail provided",
            )
        )

    try:
        score = float(data.get("risk_score", 0.0))
    except (TypeError, ValueError) as exc:
        msg = "vllm risk_score is not a number"
        raise VLLMError(msg) from exc
    score = min(max(score, 0.0), 1.0)

    return LocalAssessment(findings=tuple(findings), risk_score=score)
