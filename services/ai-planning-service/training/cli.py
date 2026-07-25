from __future__ import annotations

import argparse
import json

from training.artifacts import write_experiment_manifest
from training.config import load_config
from training.dataset_loader import load_training_dataset


def main() -> int:
    parser = argparse.ArgumentParser(description="Run a local AI fine-tuning experiment.")
    parser.add_argument("--config", required=True, help="Training config JSON")
    parser.add_argument("--dataset-version", help="Override dataset version label")
    parser.add_argument("--dataset-path", help="Override dataset export path")
    parser.add_argument("--method", choices=["lora", "qlora"], help="Override training method")
    parser.add_argument("--dry-run", action="store_true", help="Validate without training")
    parser.add_argument(
        "--cpu-smoke",
        action="store_true",
        help="Write a manifest without GPU training",
    )
    parser.add_argument("--resume", action="store_true", help="Resume from latest checkpoint")
    parser.add_argument(
        "--evaluate-after",
        action="store_true",
        help="Print post-training eval reminder",
    )
    args = parser.parse_args()

    config = load_config(args.config)
    updates = {}
    if args.dataset_version:
        updates["dataset_version"] = args.dataset_version
    if args.dataset_path:
        updates["dataset_path"] = args.dataset_path
    if args.method:
        updates["method"] = args.method
    if args.dry_run:
        updates["dry_run"] = True
    if args.cpu_smoke:
        updates["cpu_smoke"] = True
    if updates:
        config = config.model_copy(update=updates)

    bundle = load_training_dataset(config)
    if config.dry_run or config.cpu_smoke:
        manifest_path = write_experiment_manifest(
            config,
            bundle,
            status="validated" if config.dry_run else "cpu_smoke_completed",
        )
        print(
            json.dumps(
                {
                    "status": "ok",
                    "experimentKey": config.experiment_key,
                    "method": config.method,
                    "datasetVersion": config.dataset_version,
                    "distribution": bundle.distribution,
                    "manifestPath": str(manifest_path),
                },
                indent=2,
                sort_keys=True,
            )
        )
        return 0

    if config.method == "lora":
        from training.train_lora import run_lora_training

        adapter_dir = run_lora_training(config, bundle, resume=args.resume)
    else:
        from training.train_qlora import run_qlora_training

        adapter_dir = run_qlora_training(config, bundle, resume=args.resume)
    if args.evaluate_after:
        print("Run scripts/ai/evaluate-model-variants.sh before any promotion decision.")
    print(json.dumps({"status": "trained", "adapterDir": str(adapter_dir)}, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
