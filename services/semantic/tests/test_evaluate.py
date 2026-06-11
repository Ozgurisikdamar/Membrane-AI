"""The EvaluateDiff use-case: tier selection and the cost gate."""

import asyncio

import pytest

from semantic.adapters.premium_stub import DisabledPremiumConsensus
from semantic.app.evaluate import EvaluateDiff
from semantic.domain.models import (
    EvaluationInput,
    Finding,
    LocalAssessment,
    Severity,
    Tier,
)


class FakeLocal:
    def __init__(self, assessment: LocalAssessment):
        self._a = assessment
        self.calls = 0

    async def assess(self, item: EvaluationInput) -> LocalAssessment:
        self.calls += 1
        return self._a


class FakePremium:
    def __init__(self, findings: tuple[Finding, ...] = ()):
        self._f = findings
        self.calls = 0

    async def review(self, item: EvaluationInput) -> tuple[Finding, ...]:
        self.calls += 1
        return self._f


def _input() -> EvaluationInput:
    return EvaluationInput(submission_id="s", organization_id="o", language="go", masked_diff="+x")


def test_low_risk_stays_local():
    local = FakeLocal(LocalAssessment(findings=(), risk_score=0.1))
    premium = FakePremium()
    uc = EvaluateDiff(local=local, premium=premium, premium_enabled=True, escalation_threshold=0.5)

    result = asyncio.run(uc.handle(_input()))

    assert result.tier is Tier.LOCAL
    assert result.escalated is False
    assert premium.calls == 0, "cost gate must not invoke premium on low risk"


def test_high_risk_escalates_and_merges():
    local_finding = Finding(rule="l", severity=Severity.INFO, message="m")
    premium_finding = Finding(rule="p", severity=Severity.BLOCKING, message="m")
    local = FakeLocal(LocalAssessment(findings=(local_finding,), risk_score=0.9))
    premium = FakePremium((premium_finding,))
    uc = EvaluateDiff(local=local, premium=premium, premium_enabled=True, escalation_threshold=0.5)

    result = asyncio.run(uc.handle(_input()))

    assert result.tier is Tier.PREMIUM
    assert result.escalated is True
    assert result.findings == (local_finding, premium_finding), "premium augments local"
    assert premium.calls == 1


def test_flag_off_never_calls_premium_even_on_high_risk():
    local = FakeLocal(LocalAssessment(findings=(), risk_score=1.0))
    premium = FakePremium()
    uc = EvaluateDiff(local=local, premium=premium, premium_enabled=False, escalation_threshold=0.5)

    result = asyncio.run(uc.handle(_input()))

    assert result.tier is Tier.LOCAL
    assert premium.calls == 0


def test_disabled_premium_stub_refuses():
    with pytest.raises(RuntimeError, match="premium consensus is not configured"):
        asyncio.run(DisabledPremiumConsensus().review(_input()))
