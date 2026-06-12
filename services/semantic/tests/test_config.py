"""Config loader: env parsing + fail-fast validation."""

import pytest

from semantic.config import Config, ConfigError


def test_defaults_load(monkeypatch):
    for var in ("VLLM_TIMEOUT_SECONDS", "PREMIUM_TIMEOUT_SECONDS", "PREMIUM_ENABLED", "PORT"):
        monkeypatch.delenv(f"MEMBRANE_SEMANTIC_{var}", raising=False)
    cfg = Config.load()
    assert cfg.port == 8005
    assert cfg.vllm_timeout_seconds == 8.0
    assert cfg.premium_timeout_seconds == 30.0
    assert cfg.premium_enabled is False


@pytest.mark.parametrize(
    "var",
    ["MEMBRANE_SEMANTIC_VLLM_TIMEOUT_SECONDS", "MEMBRANE_SEMANTIC_PREMIUM_TIMEOUT_SECONDS"],
)
@pytest.mark.parametrize("bad", ["0", "-5", "-0.1"])
def test_non_positive_timeout_fails_fast(monkeypatch, var, bad):
    monkeypatch.setenv(var, bad)
    with pytest.raises(ConfigError, match="must be > 0"):
        Config.load()


def test_invalid_float_fails_fast(monkeypatch):
    monkeypatch.setenv("MEMBRANE_SEMANTIC_VLLM_TIMEOUT_SECONDS", "soon")
    with pytest.raises(ConfigError, match="invalid float"):
        Config.load()


def test_invalid_bool_fails_fast(monkeypatch):
    monkeypatch.setenv("MEMBRANE_SEMANTIC_PREMIUM_ENABLED", "maybe")
    with pytest.raises(ConfigError, match="invalid bool"):
        Config.load()
