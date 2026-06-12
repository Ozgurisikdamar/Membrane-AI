"""Composition root for the semantic service: wire adapters into the use-case
and serve the HTTP app (D-023)."""

from __future__ import annotations

import uvicorn
from fastapi import FastAPI

from semantic.adapters import tracing
from semantic.adapters.fallback import FallbackLocalModel
from semantic.adapters.http_api import create_app
from semantic.adapters.local_stub import HeuristicLocalModel
from semantic.adapters.premium_consensus import (
    AnthropicReviewer,
    DualModelConsensus,
    GeminiReviewer,
    Reviewer,
)
from semantic.adapters.premium_stub import DisabledPremiumConsensus
from semantic.adapters.vllm_local import VLLMLocalModel
from semantic.app.evaluate import EvaluateDiff
from semantic.config import Config
from semantic.ports import LocalModel, PremiumConsensus


def build() -> tuple[FastAPI, Config]:
    """Wire the service from configuration (used by main and by tests)."""
    cfg = Config.load()
    # Adapters holding network clients; their aclose() runs on app shutdown.
    closeables: list[object] = []

    local: LocalModel = HeuristicLocalModel()
    if cfg.vllm_url:
        # Real tier-2 model with graceful degradation to the heuristic (D-028).
        vllm = VLLMLocalModel(cfg.vllm_url, cfg.vllm_model, cfg.vllm_timeout_seconds)
        closeables.append(vllm)
        local = FallbackLocalModel(primary=vllm, backup=HeuristicLocalModel())

    premium = _build_premium(cfg)
    closeables.append(premium)  # DisabledPremiumConsensus has no aclose → lifespan skips it

    evaluate = EvaluateDiff(
        local=local,
        premium=premium,
        premium_enabled=cfg.premium_enabled,
        escalation_threshold=cfg.escalation_threshold,
    )
    app = create_app(evaluate, closeables)
    tracing.configure(app, cfg)  # distributed tracing, gated on OTLP endpoint (D-032)
    return app, cfg


def _build_premium(cfg: Config) -> PremiumConsensus:
    """Wire the tier-3 consensus from whichever provider keys are present
    (D-031). With the flag off, or on but no keys, the disabled stub stays —
    so an enabled-but-unconfigured premium tier fails loud instead of silently
    degrading."""
    if not cfg.premium_enabled:
        return DisabledPremiumConsensus()
    reviewers: list[Reviewer] = []
    if cfg.premium_anthropic_api_key:
        reviewers.append(
            AnthropicReviewer(
                cfg.premium_anthropic_api_key,
                cfg.premium_anthropic_model,
                cfg.premium_timeout_seconds,
            )
        )
    if cfg.premium_gemini_api_key:
        reviewers.append(
            GeminiReviewer(
                cfg.premium_gemini_api_key,
                cfg.premium_gemini_model,
                cfg.premium_timeout_seconds,
            )
        )
    if not reviewers:
        return DisabledPremiumConsensus()
    return DualModelConsensus(tuple(reviewers), timeout_seconds=cfg.premium_timeout_seconds)


def main() -> None:
    """Entrypoint: `python -m semantic.main`."""
    app, cfg = build()
    uvicorn.run(app, host=cfg.host, port=cfg.port, log_level="info")


if __name__ == "__main__":
    main()
