"""HTTP adapter: the FastAPI app exposing POST /v1/semantic/evaluate (D-023).

Pydantic models live here, at the boundary; they map to/from the pure domain.
"""

from __future__ import annotations

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, Field

from semantic.app.evaluate import EvaluateDiff
from semantic.domain.models import EvaluationInput, GoldContext, Severity, Tier


class GoldContextIn(BaseModel):
    """One resolver-provided gold snippet."""

    file_path: str = ""
    code: str = ""
    architectural_context: str = ""


class EvaluateRequest(BaseModel):
    """Wire request for /v1/semantic/evaluate."""

    submission_id: str = Field(min_length=1)
    organization_id: str = Field(min_length=1)
    language: str = ""
    masked_diff: str = Field(min_length=1)
    gold_context: list[GoldContextIn] = Field(default_factory=list)


class FindingOut(BaseModel):
    """One finding in the response."""

    rule: str
    severity: Severity
    message: str


class EvaluateResponse(BaseModel):
    """Wire response for /v1/semantic/evaluate."""

    findings: list[FindingOut]
    tier: Tier
    escalated: bool


def create_app(evaluate: EvaluateDiff) -> FastAPI:
    """Build the FastAPI app around the injected use-case (composition root
    passes the wired EvaluateDiff in — no globals)."""
    app = FastAPI(title="MEMBRANE.AI semantic service", version="0.1.0")

    @app.get("/livez")
    async def livez() -> dict[str, str]:
        return {"status": "ok"}

    @app.get("/readyz")
    async def readyz() -> dict[str, str]:
        return {"status": "ready"}

    @app.post("/v1/semantic/evaluate", response_model=EvaluateResponse)
    async def evaluate_endpoint(req: EvaluateRequest) -> EvaluateResponse:
        try:
            item = EvaluationInput(
                submission_id=req.submission_id,
                organization_id=req.organization_id,
                language=req.language.lower().strip(),
                masked_diff=req.masked_diff,
                gold_context=tuple(
                    GoldContext(
                        file_path=g.file_path,
                        code=g.code,
                        architectural_context=g.architectural_context,
                    )
                    for g in req.gold_context
                ),
            )
        except ValueError as exc:
            raise HTTPException(status_code=400, detail=str(exc)) from exc

        try:
            result = await evaluate.handle(item)
        except RuntimeError as exc:  # premium misconfiguration → 503, retryable
            raise HTTPException(status_code=503, detail=str(exc)) from exc

        return EvaluateResponse(
            findings=[
                FindingOut(rule=f.rule, severity=f.severity, message=f.message)
                for f in result.findings
            ],
            tier=result.tier,
            escalated=result.escalated,
        )

    return app
