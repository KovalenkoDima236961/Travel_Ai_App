from pathlib import Path


def service_root() -> Path:
    return Path(__file__).resolve().parents[2]


def resolve_service_path(raw_path: str | Path) -> Path:
    path = Path(raw_path)
    if path.is_absolute():
        return path

    cwd_path = Path.cwd() / path
    if cwd_path.exists():
        return cwd_path

    return service_root() / path
