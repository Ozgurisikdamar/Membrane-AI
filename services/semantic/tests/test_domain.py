"""Domain model invariants and the escalation gate policy."""

import pytest

from semantic.domain.models import (
    EvaluationInput,
    Finding,
    LocalAssessment,
    Severity,
    should_escalate,
)


def _input(diff: str = "+x := 1") -> EvaluationInput:
    return EvaluationInput(submission_id="s", organization_id="o", language="go", masked_diff=diff)


def test_evaluation_input_requires_diff():
    with pytest.raises(ValueError, match="masked_diff"):
        EvaluationInput(submission_id="s", organization_id="o", language="go", masked_diff="  ")


def test_local_assessment_validates_score_bounds():
    with pytest.raises(ValueError, match="risk_score"):
        LocalAssessment(findings=(), risk_score=1.5)
    with pytest.raises(ValueError, match="risk_score"):
        LocalAssessment(findings=(), risk_score=-0.1)


@pytest.mark.parametrize(
    ("findings", "score", "threshold", "want"),
    [
        ((), 0.4, 0.5, False),  # below threshold
        ((), 0.5, 0.5, True),  # at threshold
        ((), 0.9, 0.5, True),  # above threshold
        (
            (Finding(rule="x", severity=Severity.BLOCKING, message="m"),),
            0.0,
            0.5,
            True,
        ),  # blocking always escalates
        (
            (Finding(rule="x", severity=Severity.INFO, message="m"),),
            0.1,
            0.5,
            False,
        ),  # info alone does not
    ],
)
def test_should_escalate(findings, score, threshold, want):
    assessment = LocalAssessment(findings=findings, risk_score=score)
    assert should_escalate(assessment, threshold) is want
