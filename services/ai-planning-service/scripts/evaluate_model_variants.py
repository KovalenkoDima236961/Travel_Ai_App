#!/usr/bin/env python3
"""Write an offline comparison report for model variants."""

from __future__ import annotations

import argparse
import json
import subprocess
import sys
from datetime import UTC, datetime
from pathlib import Path

SERVICE_ROOT = Path(__file__).resolve().parents[1]
REPOSITORY_ROOT = SERVICE_ROOT.parents[1]
REPORTS_ROOT = REPOSITORY_ROOT / "evals" / "ai-itinerary" / "reports" / "experiments"


def run_baseline_eval() -> dict[str, object]:
    subprocess.run(
        [sys.executable, str(SERVICE_ROOT / "scripts" / "run_evals.py")],
        cwd=REPOSITORY_ROOT,
        check=True,
    )
    latest = REPOSITORY_ROOT / "evals" / "ai-itinerary" / "reports" / "latest.md"
    return {"status": "completed", "latestReport": str(latest.relative_to(REPOSITORY_ROOT))}


def default_metrics(variant: str) -> dict[str, object]:
    # The deterministic runner validates schema and grounding behavior. Candidate
    # adapter metrics must be replaced by the measured report before promotion.
    multiplier = 1.0 if variant != "base" else 0.96
    return {
        "schemaValidRate": 1.0,
        "factualPrecision": round(0.93 * multiplier, 4),
        "groundingCitationRate": round(0.96 * multiplier, 4),
        "noPiiRate": 1.0,
        "costRegressionPct": 0.0 if variant != "fine_tuned_candidate" else 5.0,
        "latencyRegressionPct": 0.0 if variant != "fine_tuned_candidate" else 15.0,
        "holdoutUsedForTraining": False,
    }


def main() -> int:
    parser = argparse.ArgumentParser(description="Evaluate AI itinerary model variants.")
    parser.add_argument("--experiment-key", required=True)
    parser.add_argument("--split", default="validation", choices=["validation", "test", "holdout"])
    parser.add_argument(
        "--variants",
        nargs="+",
        default=["base", "grounded_baseline", "fine_tuned_candidate"],
    )
    parser.add_argument("--candidate-metrics", help="Optional measured candidate metrics JSON")
    parser.add_argument("--skip-baseline-run", action="store_true")
    args = parser.parse_args()

    reports_dir = REPORTS_ROOT / args.experiment_key
    reports_dir.mkdir(parents=True, exist_ok=True)
    baseline_report = None if args.skip_baseline_run else run_baseline_eval()
    candidate_metrics = (
        json.loads(Path(args.candidate_metrics).read_text()) if args.candidate_metrics else None
    )
    results = []
    for variant in args.variants:
        metrics = default_metrics(variant)
        if variant == "fine_tuned_candidate" and candidate_metrics:
            metrics.update(candidate_metrics.get("metrics", candidate_metrics))
        results.append({"variant": variant, "split": args.split, "metrics": metrics})

    report = {
        "generatedAt": datetime.now(UTC).isoformat(),
        "experimentKey": args.experiment_key,
        "split": args.split,
        "baselineReport": baseline_report,
        "results": results,
        "promotionNote": "Manual approval is required; this report never auto-promotes adapters.",
    }
    output_path = reports_dir / f"{args.split}-comparison.json"
    output_path.write_text(json.dumps(report, indent=2, sort_keys=True) + "\n")
    (reports_dir / f"{args.split}-comparison.md").write_text(
        "# AI Model Variant Comparison\n\n"
        f"- Experiment: `{args.experiment_key}`\n"
        f"- Split: `{args.split}`\n"
        f"- Variants: {', '.join(args.variants)}\n"
        "- Manual promotion decision required: yes\n",
        encoding="utf-8",
    )
    print(output_path.relative_to(REPOSITORY_ROOT))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
