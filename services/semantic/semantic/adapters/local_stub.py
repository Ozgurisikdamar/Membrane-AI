"""Deterministic stub for the tier-2 local model.

This is explicitly a STAND-IN until the vLLM-backed local model lands
(ROADMAP P2): a transparent heuristic that scores risk from simple signals in
the masked diff. It exists so the gate, the API and the orchestrator
integration are fully exercisable end-to-end today. It makes no model claims.
"""

from __future__ import annotations

from semantic.domain.models import EvaluationInput, Finding, LocalAssessment, Severity

# Touching these areas warrants a closer look — each adds to the risk score.
_RISK_MARKERS: tuple[tuple[str, float, str], ...] = (
    ("auth", 0.3, "change touches authentication logic"),
    ("crypto", 0.3, "change touches cryptographic code"),
    ("password", 0.3, "change handles passwords"),
    ("[masked:", 0.4, "a secret was masked out of this diff upstream"),
    ("eval(", 0.4, "dynamic evaluation of generated input"),
    ("subprocess", 0.2, "spawns external processes"),
)


class HeuristicLocalModel:
    """Implements ports.LocalModel with deterministic, explainable signals."""

    async def assess(self, item: EvaluationInput) -> LocalAssessment:
        lowered = item.masked_diff.lower()
        findings: list[Finding] = []
        score = 0.0
        for marker, weight, why in _RISK_MARKERS:
            if marker in lowered:
                score += weight
                findings.append(
                    Finding(
                        rule=f"local-heuristic:{marker.strip('[(:')}",
                        severity=Severity.INFO,
                        message=f"{why}; flagged for closer review",
                    )
                )
        # Large changes are inherently riskier to wave through.
        if len(item.masked_diff) > 4000:
            score += 0.2
            findings.append(
                Finding(
                    rule="local-heuristic:large-change",
                    severity=Severity.INFO,
                    message="large diff; semantic review depth is limited",
                )
            )
        return LocalAssessment(findings=tuple(findings), risk_score=min(score, 1.0))
