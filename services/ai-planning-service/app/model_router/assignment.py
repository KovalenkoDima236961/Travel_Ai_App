from __future__ import annotations

import hashlib


def deterministic_bucket(assignment_salt: str, stable_key: str) -> int:
    salt = assignment_salt.strip() or "missing-salt"
    key = stable_key.strip() or "anonymous"
    digest = hashlib.sha256(f"{salt}:{key}".encode("utf-8")).digest()
    return int.from_bytes(digest[:8], "big") % 10000


def rollout_selected(bucket: int, percent: float) -> bool:
    clamped = min(max(percent, 0), 100)
    return bucket < int(clamped * 100)

