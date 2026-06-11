"""EvaluateDiff use-case: the tier-2/tier-3 slice of the cost gate (D-008)."""

from __future__ import annotations

from dataclasses import dataclass

from semantic.domain.models import (
    EvaluationInput,
    EvaluationResult,
    Tier,
    should_escalate,
)
from semantic.ports import LocalModel, PremiumConsensus


@dataclass(frozen=True, slots=True)
class EvaluateDiff:
    """Run the local tier always; escalate to premium only on real risk and
    only when the premium tier is enabled (cost gate)."""

    local: LocalModel
    premium: PremiumConsensus
    premium_enabled: bool
    escalation_threshold: float

    async def handle(self, item: EvaluationInput) -> EvaluationResult:
        assessment = await self.local.assess(item)

        escalate = should_escalate(assessment, self.escalation_threshold)
        if not (escalate and self.premium_enabled):
            return EvaluationResult(
                findings=assessment.findings,
                tier=Tier.LOCAL,
                escalated=False,
            )

        premium_findings = await self.premium.review(item)
        # Premium review augments — never silently discards — the local tier.
        merged = assessment.findings + premium_findings
        return EvaluationResult(findings=merged, tier=Tier.PREMIUM, escalated=True)
