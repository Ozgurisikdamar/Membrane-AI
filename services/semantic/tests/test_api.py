"""HTTP adapter tests via FastAPI's TestClient (real wiring, stub adapters)."""

from fastapi.testclient import TestClient

from semantic.adapters.http_api import create_app
from semantic.adapters.local_stub import HeuristicLocalModel
from semantic.adapters.premium_stub import DisabledPremiumConsensus
from semantic.app.evaluate import EvaluateDiff


def make_client(premium_enabled: bool = False) -> TestClient:
    evaluate = EvaluateDiff(
        local=HeuristicLocalModel(),
        premium=DisabledPremiumConsensus(),
        premium_enabled=premium_enabled,
        escalation_threshold=0.5,
    )
    return TestClient(create_app(evaluate))


def test_health_endpoints():
    client = make_client()
    assert client.get("/livez").json() == {"status": "ok"}
    assert client.get("/readyz").json() == {"status": "ready"}


def test_evaluate_clean_diff_stays_local():
    client = make_client()
    resp = client.post(
        "/v1/semantic/evaluate",
        json={
            "submission_id": "s-1",
            "organization_id": "o-1",
            "language": "go",
            "masked_diff": "+x := 1",
        },
    )
    assert resp.status_code == 200
    body = resp.json()
    assert body["tier"] == "local"
    assert body["escalated"] is False


def test_evaluate_risky_diff_with_premium_off_stays_local():
    client = make_client(premium_enabled=False)
    resp = client.post(
        "/v1/semantic/evaluate",
        json={
            "submission_id": "s-2",
            "organization_id": "o-1",
            "language": "go",
            # markers: masked secret + auth + crypto → score ≥ threshold
            "masked_diff": '+pw = "[MASKED:generic-assigned-secret]"\n+auth.crypto.Verify()',
        },
    )
    assert resp.status_code == 200
    body = resp.json()
    assert body["tier"] == "local"
    assert len(body["findings"]) >= 2


def test_evaluate_escalation_with_disabled_premium_is_503():
    client = make_client(premium_enabled=True)
    resp = client.post(
        "/v1/semantic/evaluate",
        json={
            "submission_id": "s-3",
            "organization_id": "o-1",
            "language": "go",
            "masked_diff": '+pw = "[MASKED:x]"\n+auth password crypto',
        },
    )
    assert resp.status_code == 503, "flag on without a real adapter must fail loudly"


def test_evaluate_tolerates_null_gold_context():
    # Go's encoding/json renders a nil slice as `null`; the edge must accept it
    # as "no context" (regression: container E2E got a 422 here).
    client = make_client()
    resp = client.post(
        "/v1/semantic/evaluate",
        json={
            "submission_id": "s-4",
            "organization_id": "o-1",
            "language": "go",
            "masked_diff": "+x := 1",
            "gold_context": None,
        },
    )
    assert resp.status_code == 200
    assert resp.json()["tier"] == "local"


def test_evaluate_validation_errors():
    client = make_client()
    # pydantic rejects an empty diff at the edge
    resp = client.post(
        "/v1/semantic/evaluate",
        json={"submission_id": "s", "organization_id": "o", "masked_diff": ""},
    )
    assert resp.status_code == 422
    # whitespace-only diff passes pydantic length but fails the domain invariant
    resp = client.post(
        "/v1/semantic/evaluate",
        json={"submission_id": "s", "organization_id": "o", "masked_diff": "   "},
    )
    assert resp.status_code == 400
