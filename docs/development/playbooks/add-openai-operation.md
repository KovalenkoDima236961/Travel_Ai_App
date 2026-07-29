# Add OpenAI Operation

Use this when adding another AI Planning operation to the OpenAI provider.

## Checklist

1. Add the operation to `AIModelProvider` in `app/providers/base.py`.
2. Add or reuse the Pydantic request/response schemas.
3. Reuse an existing prompt builder or add one under `app/services`.
4. Add an operation-specific model alias in `Settings`.
5. Map the operation in `Settings.openai_model_for_operation`.
6. Implement `OpenAIProvider.<operation>` using `_structured_response`.
7. Re-run Pydantic and business validation after the provider response.
8. Add fallback wrapper support only for allowed transient failures.
9. Attach safe provider metadata to private responses only.
10. Add mocked SDK tests. CI must not perform real OpenAI calls.

Do not return SDK objects, raw provider response bodies, prompts, or stack traces
outside `app/providers`.

