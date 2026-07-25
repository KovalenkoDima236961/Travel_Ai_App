from __future__ import annotations

import argparse
import json
from dataclasses import asdict, dataclass
from pathlib import Path
from typing import Any

from training.config import PromotionGates


@dataclass(frozen=True)
class PromotionGateResult:
    approved: bool
    failed_gates: list[str]
    metrics_snapshot: dict[str, Any]


def evaluate_promotion_gates(
    metrics: dict[str, Any],
    gates: PromotionGates | None = None,
) -> PromotionGateResult:
    gates = gates or PromotionGates()
    metrics = _candidate_metrics(metrics)
    failed: list[str] = []
    checks = {
        "schema_valid_rate": _metric(metrics, "schema_valid_rate", "schemaValidRate"),
        "factual_precision": _metric(metrics, "factual_precision", "factualPrecision"),
        "grounding_citation_rate": _metric(
            metrics,
            "grounding_citation_rate",
            "groundingCitationRate",
        ),
        "no_pii_rate": _metric(metrics, "no_pii_rate", "noPiiRate"),
        "cost_regression_pct": _metric(metrics, "cost_regression_pct", "costRegressionPct"),
        "latency_regression_pct": _metric(
            metrics,
            "latency_regression_pct",
            "latencyRegressionPct",
        ),
        "holdout_used_for_training": bool(
            _metric(metrics, "holdout_used_for_training", "holdoutUsedForTraining", default=False)
        ),
    }
    if checks["schema_valid_rate"] < gates.min_schema_valid_rate:
        failed.append("schema_valid_rate")
    if checks["factual_precision"] < gates.min_factual_precision:
        failed.append("factual_precision")
    if checks["grounding_citation_rate"] < gates.min_grounding_citation_rate:
        failed.append("grounding_citation_rate")
    if checks["no_pii_rate"] < gates.min_no_pii_rate:
        failed.append("no_pii_rate")
    if checks["cost_regression_pct"] > gates.max_cost_regression_pct:
        failed.append("cost_regression_pct")
    if checks["latency_regression_pct"] > gates.max_latency_regression_pct:
        failed.append("latency_regression_pct")
    if checks["holdout_used_for_training"]:
        failed.append("holdout_used_for_training")
    return PromotionGateResult(
        approved=not failed,
        failed_gates=failed,
        metrics_snapshot=checks,
    )


def _metric(metrics: dict[str, Any], *keys: str, default: float | bool = 0.0) -> Any:
    for key in keys:
        if key in metrics:
            return metrics[key]
        if "." in key:
            value: Any = metrics
            for part in key.split("."):
                if not isinstance(value, dict) or part not in value:
                    value = None
                    break
                value = value[part]
            if value is not None:
                return value
    if isinstance(metrics.get("metrics"), dict):
        return _metric(metrics["metrics"], *keys, default=default)
    return default


def _candidate_metrics(payload: dict[str, Any]) -> dict[str, Any]:
    results = payload.get("results")
    if not isinstance(results, list):
        return payload
    for result in results:
        if not isinstance(result, dict):
            continue
        if result.get("variant") == "fine_tuned_candidate" and isinstance(
            result.get("metrics"),
            dict,
        ):
            return result["metrics"]
    for result in results:
        if isinstance(result, dict) and isinstance(result.get("metrics"), dict):
            return result["metrics"]
    return payload


def main() -> int:
    parser = argparse.ArgumentParser(description="Check AI model promotion gates.")
    parser.add_argument("--metrics", required=True, help="Metrics JSON file")
    args = parser.parse_args()
    metrics = json.loads(Path(args.metrics).read_text())
    result = evaluate_promotion_gates(metrics)
    print(json.dumps(asdict(result), indent=2, sort_keys=True))
    return 0 if result.approved else 1


if __name__ == "__main__":
    raise SystemExit(main())
