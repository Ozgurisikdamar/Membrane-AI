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


def _get_positive_float(name: str, default: float) -> float:
    """A float that must be > 0 — a non-positive timeout would make every call
    fail instantly (httpx/asyncio.wait_for), so fail fast at startup instead."""
    value = _get_float(name, default)
    if value <= 0:
        msg = f"{_PREFIX}{name}: must be > 0, got {value}"
        raise ConfigError(msg)
    return value


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
    # Tier-3 premium consensus credentials (D-031). Keys come from the env /
    # Helm Secret; an empty key drops that reviewer. With premium_enabled and
    # no key at all, the disabled stub stays wired and escalation fails loud.
    premium_anthropic_api_key: str
    premium_anthropic_model: str
    premium_gemini_api_key: str
    premium_gemini_model: str
    premium_timeout_seconds: float
    # Distributed tracing (D-032): OTLP/gRPC collector address (empty = off);
    # otlp_insecure sends over plaintext gRPC for a dev collector.
    otlp_endpoint: str
    otlp_insecure: bool

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
            vllm_timeout_seconds=_get_positive_float("VLLM_TIMEOUT_SECONDS", 8.0),
            premium_anthropic_api_key=_get("ANTHROPIC_API_KEY", ""),
            premium_anthropic_model=_get("PREMIUM_ANTHROPIC_MODEL", "claude-sonnet-4-6"),
            premium_gemini_api_key=_get("GEMINI_API_KEY", ""),
            premium_gemini_model=_get("PREMIUM_GEMINI_MODEL", "gemini-2.5-pro"),
            premium_timeout_seconds=_get_positive_float("PREMIUM_TIMEOUT_SECONDS", 30.0),
            otlp_endpoint=_get("OTLP_ENDPOINT", ""),
            otlp_insecure=_get_bool("OTLP_INSECURE", default=False),
        )
