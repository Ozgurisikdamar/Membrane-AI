"""Ports (Protocols) the semantic application depends on — defined on the
consumer side; adapters implement them (SOLID I/D)."""

from __future__ import annotations

from typing import Protocol

from semantic.domain.models import EvaluationInput, Finding, LocalAssessment


class LocalModel(Protocol):
    """Tier-2: a cheap model inside the trust boundary (vLLM pool in prod)."""

    async def assess(self, item: EvaluationInput) -> LocalAssessment:
        """Return findings plus an escalation risk score."""
        ...


class PremiumConsensus(Protocol):
    """Tier-3: the premium dual-model consensus (Claude + Gemini in prod).

    Implementations are called ONLY when the gate escalates (D-008) and the
    feature flag enables them.
    """

    async def review(self, item: EvaluationInput) -> tuple[Finding, ...]:
        """Return the premium tier's findings."""
        ...
