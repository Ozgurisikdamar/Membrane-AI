"""Disabled stand-in for the tier-3 premium consensus.

The real adapter (Claude Sonnet 4.6 + Gemini dual consensus, D-009) lands when
API-key handling is designed; until then the premium tier stays behind the
feature flag and this stub guarantees the port is never silently exercised.
"""

from __future__ import annotations

from semantic.domain.models import EvaluationInput, Finding


class DisabledPremiumConsensus:
    """Implements ports.PremiumConsensus by refusing to run."""

    async def review(self, item: EvaluationInput) -> tuple[Finding, ...]:
        msg = (
            "premium consensus is not configured "
            f"(submission {item.submission_id}); enable it only with a real adapter"
        )
        raise RuntimeError(msg)
