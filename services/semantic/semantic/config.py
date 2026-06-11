"""12-factor configuration for the semantic service (prefix MEMBRANE_SEMANTIC_)."""

from __future__ import annotations

import os
from dataclasses import dataclass

_PREFIX = "MEMBRANE_SEMANTIC_"


class ConfigError(ValueError):
    """Raised at startup when an environment value cannot be parsed."""


def _get(name: str, default: str) -> str:
    return os.environ.get(_PREFIX + name, default)


def _get_float(name: str, default: float) -> float:
    raw = _get(name, str(default))
    try:
        return float(raw)
    except ValueError as exc:
        msg = f"{_PREFIX}{name}: invalid float {raw!r}"
        raise ConfigError(msg) from exc


def _get_bool(name: str, *, default: bool) -> bool:
    raw = _get(name, "1" if default else "0").strip().lower()
    if raw in {"1", "t", "true", "yes", "on"}:
        return True
    if raw in {"0", "f", "false", "no", "off"}:
        return False
    msg = f"{_PREFIX}{name}: invalid bool {raw!r}"
    raise ConfigError(msg)


@dataclass(frozen=True, slots=True)
class Config:
    """Runtime settings; defaults match local development."""

    host: str
    port: int
    premium_enabled: bool
    escalation_threshold: float
    # vLLM tier-2 endpoint (empty = use the deterministic heuristic only).
    vllm_url: str
    vllm_model: str
    vllm_timeout_seconds: float

    @staticmethod
    def load() -> Config:
        port_raw = _get("PORT", "8005")
        try:
            port = int(port_raw)
        except ValueError as exc:
            msg = f"{_PREFIX}PORT: invalid int {port_raw!r}"
            raise ConfigError(msg) from exc
        return Config(
            host=_get("HOST", "0.0.0.0"),  # noqa: S104 — service binds all interfaces in containers
            port=port,
            premium_enabled=_get_bool("PREMIUM_ENABLED", default=False),
            escalation_threshold=_get_float("ESCALATION_THRESHOLD", 0.5),
            vllm_url=_get("VLLM_URL", ""),
            vllm_model=_get("VLLM_MODEL", "deepseek-coder"),
            vllm_timeout_seconds=_get_float("VLLM_TIMEOUT_SECONDS", 8.0),
        )
