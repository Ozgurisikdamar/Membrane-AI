"""Pure semantic domain: evaluation inputs, findings and tier outcomes.

No I/O and no framework imports here (ENGINEERING-STANDARDS §1/§8) — pydantic
lives at the adapter edge; the domain uses frozen dataclasses.
"""

from __future__ import annotations

from dataclasses import dataclass, field
from enum import StrEnum


class Severity(StrEnum):
    """Finding severity, aligned with the platform-wide scale."""

    INFO = "info"
    WARNING = "warning"
    BLOCKING = "blocking"


class Tier(StrEnum):
    """Which cost tier produced the result (D-008 three-tier gating).

    Tier 0 (cache) and tier 1 (context pruning) happen upstream in the
    orchestrator; this service implements tier 2 (local model) and tier 3
    (premium consensus).
    """

    LOCAL = "local"
    PREMIUM = "premium"


@dataclass(frozen=True, slots=True)
class GoldContext:
    """One gold-codebase snippet retrieved by the resolver (RAG)."""

    file_path: str
    code: str
    architectural_context: str


@dataclass(frozen=True, slots=True)
class EvaluationInput:
    """A masked diff plus its retrieval context, ready for model review.

    The diff MUST already be secret-masked by the analyzer ([MASKED:<rule>]
    placeholders) — this service never sees raw credentials by design.
    """

    submission_id: str
    organization_id: str
    language: str
    masked_diff: str
    gold_context: tuple[GoldContext, ...] = field(default=())

    def __post_init__(self) -> None:
        if not self.masked_diff.strip():
            msg = "masked_diff is required"
            raise ValueError(msg)


@dataclass(frozen=True, slots=True)
class Finding:
    """A single issue raised by a model tier."""

    rule: str
    severity: Severity
    message: str


@dataclass(frozen=True, slots=True)
class LocalAssessment:
    """Tier-2 output: findings plus an escalation risk score in [0, 1]."""

    findings: tuple[Finding, ...]
    risk_score: float

    def __post_init__(self) -> None:
        if not 0.0 <= self.risk_score <= 1.0:
            msg = f"risk_score must be within [0, 1], got {self.risk_score}"
            raise ValueError(msg)


@dataclass(frozen=True, slots=True)
class EvaluationResult:
    """The consolidated semantic verdict for one submission."""

    findings: tuple[Finding, ...]
    tier: Tier
    escalated: bool


def should_escalate(assessment: LocalAssessment, threshold: float) -> bool:
    """Gate policy (D-008): escalate to the premium tier only when the local
    model sees real risk — a blocking finding, or a risk score at/over the
    threshold."""
    if any(f.severity is Severity.BLOCKING for f in assessment.findings):
        return True
    return assessment.risk_score >= threshold
