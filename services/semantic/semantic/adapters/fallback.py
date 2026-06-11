"""FallbackLocalModel: keep the tier-2 stage alive when the primary model
(e.g. vLLM) is down — degrade to the deterministic heuristic instead of
failing the evaluation (Decorator over ports.LocalModel, D-028)."""

from __future__ import annotations

from semantic.domain.models import EvaluationInput, LocalAssessment
from semantic.ports import LocalModel


class FallbackLocalModel:
    """Try the primary model; on ANY failure, answer from the backup."""

    def __init__(self, primary: LocalModel, backup: LocalModel) -> None:
        self._primary = primary
        self._backup = backup
        self.last_primary_error: str | None = None  # observable for tests/metrics

    async def assess(self, item: EvaluationInput) -> LocalAssessment:
        try:
            result = await self._primary.assess(item)
        except Exception as exc:
            self.last_primary_error = str(exc)
            return await self._backup.assess(item)
        self.last_primary_error = None
        return result
