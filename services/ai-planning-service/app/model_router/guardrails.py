from __future__ import annotations


def latency_increase_percent(baseline_ms: int, candidate_ms: int) -> float:
    if baseline_ms <= 0 or candidate_ms <= baseline_ms:
        return 0
    return ((candidate_ms - baseline_ms) / baseline_ms) * 100
