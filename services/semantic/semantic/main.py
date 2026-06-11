"""Composition root for the semantic service: wire adapters into the use-case
and serve the HTTP app (D-023)."""

from __future__ import annotations

import uvicorn
from fastapi import FastAPI

from semantic.adapters.fallback import FallbackLocalModel
from semantic.adapters.http_api import create_app
from semantic.adapters.local_stub import HeuristicLocalModel
from semantic.adapters.premium_stub import DisabledPremiumConsensus
from semantic.adapters.vllm_local import VLLMLocalModel
from semantic.app.evaluate import EvaluateDiff
from semantic.config import Config
from semantic.ports import LocalModel


def build() -> tuple[FastAPI, Config]:
    """Wire the service from configuration (used by main and by tests)."""
    cfg = Config.load()
    local: LocalModel = HeuristicLocalModel()
    if cfg.vllm_url:
        # Real tier-2 model with graceful degradation to the heuristic (D-028).
        local = FallbackLocalModel(
            primary=VLLMLocalModel(cfg.vllm_url, cfg.vllm_model, cfg.vllm_timeout_seconds),
            backup=HeuristicLocalModel(),
        )
    evaluate = EvaluateDiff(
        local=local,
        premium=DisabledPremiumConsensus(),
        premium_enabled=cfg.premium_enabled,
        escalation_threshold=cfg.escalation_threshold,
    )
    return create_app(evaluate), cfg


def main() -> None:
    """Entrypoint: `python -m semantic.main`."""
    app, cfg = build()
    uvicorn.run(app, host=cfg.host, port=cfg.port, log_level="info")


if __name__ == "__main__":
    main()
