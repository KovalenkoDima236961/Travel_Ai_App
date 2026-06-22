from app.services.knowledge_chunker import chunk_text


def test_short_text_returns_one_chunk() -> None:
    text = "A concise note about local restaurants."

    assert chunk_text(text, max_chars=900, overlap_chars=120) == [text]


def test_long_text_splits_into_multiple_chunks() -> None:
    text = "\n\n".join(
        [
            "Alpha neighborhood notes with enough detail for a useful first chunk.",
            "Beta market notes with enough detail for a useful second chunk.",
            "Gamma transport notes with enough detail for a useful third chunk.",
        ]
    )

    chunks = chunk_text(text, max_chars=90, overlap_chars=20)

    assert len(chunks) > 1


def test_chunker_does_not_create_empty_chunks() -> None:
    text = "\n\n".join(["First useful paragraph.", "Second useful paragraph."])

    chunks = chunk_text(text, max_chars=25, overlap_chars=5)

    assert chunks
    assert all(chunk.strip() for chunk in chunks)


def test_overlap_is_applied_when_possible() -> None:
    first = "Alpha neighborhood notes with enough detail for overlap."
    second = "Beta market notes with enough detail for the next chunk."
    text = f"{first}\n\n{second}"

    chunks = chunk_text(text, max_chars=90, overlap_chars=24)

    assert len(chunks) == 2
    assert "detail for overlap" in chunks[1]


def test_bullet_boundaries_are_preserved_when_possible() -> None:
    text = """
Intro paragraph about the city.

- First bullet stays readable.
- Second bullet stays readable.
""".strip()

    chunks = chunk_text(text, max_chars=70, overlap_chars=10)

    assert any("- First bullet stays readable." in chunk for chunk in chunks)
    assert any("- Second bullet stays readable." in chunk for chunk in chunks)
