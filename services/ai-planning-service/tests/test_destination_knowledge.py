import json

import pytest

from app.schemas.destination_context import DestinationContext
from app.services.destination_knowledge import FileDestinationKnowledgeProvider


def test_file_destination_knowledge_provider_list_contexts_returns_all_valid_contexts(
    tmp_path,
) -> None:
    (tmp_path / "rome.json").write_text(
        json.dumps({"destination": "Rome", "aliases": ["Roma"]}),
        encoding="utf-8",
    )
    (tmp_path / "paris.json").write_text(
        json.dumps({"destination": "Paris", "aliases": ["Paris, France"]}),
        encoding="utf-8",
    )

    provider = FileDestinationKnowledgeProvider(tmp_path)

    summaries = provider.list_contexts()

    assert [summary.destination for summary in summaries] == ["Paris", "Rome"]
    assert summaries[0].aliases == ["Paris, France"]
    assert all(summary.source == "file" for summary in summaries)


def test_file_destination_knowledge_provider_list_contexts_sorts_alphabetically(tmp_path) -> None:
    (tmp_path / "vienna.json").write_text(
        json.dumps({"destination": "Vienna"}),
        encoding="utf-8",
    )
    (tmp_path / "paris.json").write_text(
        json.dumps({"destination": "paris"}),
        encoding="utf-8",
    )
    (tmp_path / "rome.json").write_text(
        json.dumps({"destination": "Rome"}),
        encoding="utf-8",
    )

    provider = FileDestinationKnowledgeProvider(tmp_path)

    assert [summary.destination for summary in provider.list_contexts()] == [
        "paris",
        "Rome",
        "Vienna",
    ]


def test_file_destination_knowledge_provider_list_contexts_skips_invalid_json(
    tmp_path,
    caplog: pytest.LogCaptureFixture,
) -> None:
    caplog.set_level("WARNING", logger="app.services.destination_knowledge")
    (tmp_path / "broken.json").write_text("{not valid json", encoding="utf-8")
    (tmp_path / "rome.json").write_text(
        json.dumps({"destination": "Rome"}),
        encoding="utf-8",
    )

    provider = FileDestinationKnowledgeProvider(tmp_path)

    assert [summary.destination for summary in provider.list_contexts()] == ["Rome"]
    assert any(
        "Skipping invalid destination context JSON" in record.message for record in caplog.records
    )


def test_file_destination_knowledge_provider_list_contexts_skips_invalid_schema(
    tmp_path,
    caplog: pytest.LogCaptureFixture,
) -> None:
    caplog.set_level("WARNING", logger="app.services.destination_knowledge")
    (tmp_path / "missing-destination.json").write_text(
        json.dumps({"localTips": ["No destination."]}),
        encoding="utf-8",
    )
    (tmp_path / "rome.json").write_text(
        json.dumps({"destination": "Rome"}),
        encoding="utf-8",
    )

    provider = FileDestinationKnowledgeProvider(tmp_path)

    assert [summary.destination for summary in provider.list_contexts()] == ["Rome"]
    assert any(
        "Skipping invalid destination context data" in record.message for record in caplog.records
    )


def test_file_destination_knowledge_provider_matches_exact_destination(tmp_path) -> None:
    (tmp_path / "rome.json").write_text(
        json.dumps(
            {
                "destination": "Rome",
                "aliases": ["Roma"],
                "localTips": ["Start early."],
            }
        ),
        encoding="utf-8",
    )

    provider = FileDestinationKnowledgeProvider(tmp_path)

    context = provider.get_context("Rome")

    assert context == DestinationContext(
        destination="Rome",
        aliases=["Roma"],
        localTips=["Start early."],
    )


def test_file_destination_knowledge_provider_matches_destination_case_insensitively(
    tmp_path,
) -> None:
    (tmp_path / "rome.json").write_text(
        json.dumps({"destination": "Rome", "aliases": ["Roma"]}),
        encoding="utf-8",
    )

    provider = FileDestinationKnowledgeProvider(tmp_path)

    assert provider.get_context("rOmE").destination == "Rome"


def test_file_destination_knowledge_provider_matches_alias_case_insensitively(tmp_path) -> None:
    (tmp_path / "paris.json").write_text(
        json.dumps(
            {
                "destination": "Paris",
                "aliases": ["Paris, France"],
                "localTips": ["Cluster neighborhoods."],
            }
        ),
        encoding="utf-8",
    )

    provider = FileDestinationKnowledgeProvider(tmp_path)

    assert provider.get_context("PARIS, FRANCE").destination == "Paris"


def test_file_destination_knowledge_provider_trims_destination_input(tmp_path) -> None:
    (tmp_path / "rome.json").write_text(
        json.dumps({"destination": "Rome", "aliases": ["Roma"]}),
        encoding="utf-8",
    )

    provider = FileDestinationKnowledgeProvider(tmp_path)

    assert provider.get_context("  Rome  ").destination == "Rome"


def test_file_destination_knowledge_provider_skips_invalid_files(
    tmp_path,
    caplog: pytest.LogCaptureFixture,
) -> None:
    caplog.set_level("WARNING", logger="app.services.destination_knowledge")
    (tmp_path / "broken.json").write_text("{not valid json", encoding="utf-8")
    (tmp_path / "missing-destination.json").write_text(
        json.dumps({"localTips": ["No destination."]}),
        encoding="utf-8",
    )
    (tmp_path / "vienna.json").write_text(
        json.dumps({"destination": "Vienna", "aliases": ["Wien"]}),
        encoding="utf-8",
    )

    provider = FileDestinationKnowledgeProvider(tmp_path)

    assert provider.get_context("wien").destination == "Vienna"
    assert any(
        "Skipping invalid destination context JSON" in record.message for record in caplog.records
    )
    assert any(
        "Skipping invalid destination context data" in record.message for record in caplog.records
    )


def test_file_destination_knowledge_provider_returns_none_for_missing_directory(tmp_path) -> None:
    provider = FileDestinationKnowledgeProvider(tmp_path / "missing")

    assert provider.get_context("Rome") is None
    assert provider.get_context("Rome") is None
