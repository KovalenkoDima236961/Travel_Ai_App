# Graph Report - /Users/dimon228/Desktop/Travel_Ai_App  (2026-06-23)

## Corpus Check
- Large corpus: 234 files · ~80,209 words. Semantic extraction will be expensive (many Claude tokens). Consider running on a subfolder, or use --no-semantic to run AST-only.

## Summary
- 1310 nodes · 2514 edges · 47 communities detected
- Extraction: 94% EXTRACTED · 6% INFERRED · 0% AMBIGUOUS · INFERRED: 141 edges (avg confidence: 0.52)
- Token cost: 0 input · 0 output

## God Nodes (most connected - your core abstractions)
1. `OllamaItineraryGenerator` - 44 edges
2. `newTestService()` - 32 edges
3. `authContext()` - 32 edges
4. `MockItineraryGenerator` - 31 edges
5. `Service` - 30 edges
6. `Repository` - 27 edges
7. `Handler` - 26 edges
8. `LLMResponseParseError` - 23 edges
9. `_settings()` - 22 edges
10. `OllamaClientError` - 21 edges

## Surprising Connections (you probably didn't know these)
- `Full-Stack Smoke Test` --semantically_similar_to--> `Manual Web Browser Smoke Flow`  [INFERRED] [semantically similar]
  README.md → scripts/web-smoke-test.md
- `Monorepo Project Layout` --semantically_similar_to--> `Hexagonal DDD layering (domain/application/infrastructure)`  [INFERRED] [semantically similar]
  AGENTS.md → services/trip-service/README.md
- `Auth Service` --implements--> `register()`  [EXTRACTED]
  README.md → apps/web/src/lib/api/auth.ts
- `Monorepo Project Layout` --conceptually_related_to--> `Travel AI App`  [INFERRED]
  AGENTS.md → README.md
- `Postgres init script (auth_service DB creation)` --rationale_for--> `Auth Service`  [EXTRACTED]
  infra/README.md → README.md

## Hyperedges (group relationships)
- **Itinerary generation request flow** — readme_web_app, readme_trip_service, readme_user_service, readme_ai_planning_service, ai_ollama [EXTRACTED 0.95]
- **Shared JWT_ACCESS_SECRET trust** — readme_auth_service, readme_trip_service, readme_user_service, readme_jwt_shared_secret [EXTRACTED 0.95]
- **Local RAG knowledge pipeline** — ai_rag_v1, ai_chromadb, ai_nomic_embed_model, ai_knowledge_dir, infra_index_knowledge [EXTRACTED 0.90]

## Communities

### Community 0 - "Itinerary Domain Models"
Cohesion: 0.03
Nodes (69): Itinerary, ItineraryDay, ItineraryItem, PlaceRef, BaseModel, Settings, DestinationContext, DestinationContextListResponse (+61 more)

### Community 1 - "Web UI Components"
Cohesion: 0.03
Nodes (32): Place, User, addDay(), addItem(), attachPlace(), getMoveTargetDayIndex(), moveItem(), moveItemAcrossDays() (+24 more)

### Community 2 - "Trip & Integrations Repositories"
Cohesion: 0.03
Nodes (61): CreateTripInput, RegenerateItineraryPartInput, UpdateItineraryInput, Status, Trip, rowQuerier, CreateTrip, RegenerateItineraryPart (+53 more)

### Community 3 - "Generator Orchestration & Auth Context"
Cohesion: 0.03
Nodes (58): container, GenerateItineraryInput, ItineraryGenerator, RegenerateDayInput, RegenerateItemInput, AuthenticatedUser, contextKey, newTestRouter() (+50 more)

### Community 4 - "Project Documentation & Architecture"
Cohesion: 0.03
Nodes (85): Coding Style Conventions, Conventional Commit Style, Graphify Knowledge Graph Workflow, Monorepo Project Layout, Repository Guidelines (AGENTS.md), ANONYMIZED_TELEMETRY=false rationale, Business validation rules (pace items, budget sanity), ChromaDB vector store (+77 more)

### Community 5 - "Auth Service & Web Auth Integration"
Cohesion: 0.03
Nodes (43): AuthApiError, authFetch(), buildAuthUrl(), login(), logout(), me(), NewAuth(), NewUser() (+35 more)

### Community 6 - "User Profile & Preferences"
Cohesion: 0.08
Nodes (26): bearerToken(), decodeJSON(), writeError(), writeJSON(), OptionalFloat64, PatchPreferencesInput, UpdateProfileInput, Profile (+18 more)

### Community 7 - "Trip Service Unit Tests"
Cohesion: 0.13
Nodes (51): mockGenerator, mockUserContextProvider, assertInvalidInput(), authContext(), authContextWithToken(), decodeItinerary(), itineraryWithMutatedPlaceRaw(), newTestService() (+43 more)

### Community 8 - "Auth In-Memory Repository"
Cohesion: 0.07
Nodes (31): memoryRepository, mockRepo, assertInvalidInput(), assertStrings(), authContext(), defaultProfile(), newMemoryRepository(), newTestService() (+23 more)

### Community 9 - "Service Configuration"
Cohesion: 0.05
Nodes (27): AuthConfig, Config, CORSConfig, _env_bool(), _env_float(), _env_int(), _env_string(), get_settings() (+19 more)

### Community 10 - "Service Bootstrap & Schema Checks"
Cohesion: 0.06
Nodes (18): App, New(), warnWeakDevelopmentSecret(), Add(), CloseAll(), closeFn, closeFunc, closer (+10 more)

### Community 11 - "Trip Service Route Tests"
Cohesion: 0.1
Nodes (20): routeTestGenerator, routeTestRepo, newAuthTestRouter(), routeTestNextVersionNumber(), signAccessToken(), TestAuthDisabledUsesDevUserID(), TestHealthAndReadyRemainPublic(), TestItineraryVersionHistoryOwnerCanPreviewRestoreAndNonOwnerReceives404() (+12 more)

### Community 12 - "Places Search & Details"
Cohesion: 0.12
Nodes (21): PlacesHandler, getPlaceDetails(), placeFetch(), searchPlaces(), writeError(), writeJSON(), newTestRouter(), performRequest() (+13 more)

### Community 13 - "Ollama Generator Tests"
Cohesion: 0.18
Nodes (27): _itinerary_body(), _request(), _settings(), test_generated_response_keeps_exact_itinerary_response_shape(), test_initial_invalid_json_and_repair_disabled_falls_back_when_fallback_enabled(), test_initial_invalid_json_and_repair_enabled_triggers_repair(), test_log_llm_payloads_false_does_not_log_full_prompt_or_response(), test_log_llm_payloads_true_in_development_allows_payload_logging() (+19 more)

### Community 14 - "Itinerary Generation Tests"
Cohesion: 0.1
Nodes (7): partial_payload(), test_regenerate_day_instruction_too_long_returns_400(), test_regenerate_day_invalid_day_number_returns_400(), test_regenerate_day_success_returns_replacement_day_only(), test_regenerate_item_accepts_optional_user_context(), test_regenerate_item_invalid_item_index_returns_400(), test_regenerate_item_success_returns_replacement_item_only()

### Community 15 - "Repository Interfaces"
Cohesion: 0.09
Nodes (2): Repository, New()

### Community 16 - "Web API Client"
Cohesion: 0.12
Nodes (13): ApiError, apiFetch(), apiFetchInternal(), buildApiUrl(), isMissing(), NewClient(), normalizeBaseURL(), notifySessionExpired() (+5 more)

### Community 17 - "Itinerary Validator Tests"
Cohesion: 0.28
Nodes (22): _assert_validation_code(), _itinerary(), _itinerary_body(), _request(), test_avoid_term_warning_does_not_fail_validation(), test_budget_exceeded_by_more_than_thirty_percent_fails_with_budget_exceeded(), test_budget_slightly_above_requested_amount_within_thirty_percent_passes(), test_dietary_restriction_warning_does_not_fail_validation() (+14 more)

### Community 18 - "HTTP Middleware & Readiness Probes"
Cohesion: 0.15
Nodes (14): MiddlewareConfig, bearerToken(), Middleware(), writeUnauthorized(), _check_chroma(), _check_ollama(), NewRouter(), _parse_partial_request() (+6 more)

### Community 19 - "LLM Prompt Builder"
Cohesion: 0.27
Nodes (19): _append_optional_line(), build_itinerary_prompt(), build_regenerate_day_prompt(), build_regenerate_day_repair_prompt(), build_regenerate_item_prompt(), build_regenerate_item_repair_prompt(), build_repair_prompt(), _compact_content() (+11 more)

### Community 20 - "Mock Itinerary Generator"
Cohesion: 0.18
Nodes (11): MockItineraryGenerator, floatPtr(), intPtr(), mock(), mockPlaceMatches(), mockPlaces(), NewMockPlaceProvider(), normalizeSearchText() (+3 more)

### Community 21 - "Knowledge Search Tests"
Cohesion: 0.25
Nodes (10): FakeChromaClient, FakeCollection, FakeEmbeddingClient, _settings(), test_builds_search_text_from_destination_interests_and_query(), test_embedding_failure_returns_empty_list_non_fatally(), test_filters_by_destination_metadata_and_maps_results(), test_get_collection_disables_chroma_default_embedding_function() (+2 more)

### Community 22 - "Destination Context Route Tests"
Cohesion: 0.32
Nodes (12): _client(), test_destination_context_disabled_preview_prompt_returns_prompt_without_context(), test_destination_context_disabled_returns_404_for_destination(), test_destination_context_disabled_returns_empty_list(), test_generate_itinerary_still_works_after_adding_destination_context_routes(), test_get_destination_context_by_destination_returns_404_when_missing(), test_get_destination_context_by_destination_returns_context_when_found(), test_get_destination_context_returns_list_response() (+4 more)

### Community 23 - "LLM Response Parser"
Cohesion: 0.38
Nodes (13): _ensure_exact_day_response_shape(), _ensure_exact_item_response_shape(), _ensure_exact_item_shape(), _ensure_exact_response_shape(), _ensure_item_values_valid(), _extract_first_json_object(), LLMResponseParseError, parse_itinerary_response() (+5 more)

### Community 24 - "Itinerary Editor Utilities (Web)"
Cohesion: 0.22
Nodes (3): canMoveItemToDay(), moveItemToDay(), normalizeItineraryDays()

### Community 25 - "Destination Knowledge Provider Tests"
Cohesion: 0.18
Nodes (0): 

### Community 26 - "RAG Prompt Builder Tests"
Cohesion: 0.38
Nodes (7): _rag_chunks(), _request(), test_itinerary_prompt_includes_rag_context_when_chunks_exist(), test_itinerary_prompt_omits_user_context_when_not_provided(), test_itinerary_prompt_preserves_json_schema_instructions_with_rag(), test_itinerary_prompt_works_without_rag_chunks(), test_repair_prompt_includes_rag_context()

### Community 27 - "User Service DB Helpers"
Cohesion: 0.22
Nodes (2): abs32(), fromPgNumericPtr()

### Community 28 - "File Destination Knowledge Provider"
Cohesion: 0.46
Nodes (1): FileDestinationKnowledgeProvider

### Community 29 - "Knowledge Routes Tests"
Cohesion: 0.46
Nodes (5): _client(), FakeKnowledgeSearchService, test_knowledge_search_route_rejects_invalid_top_k(), test_knowledge_search_route_returns_empty_list_when_rag_disabled(), test_knowledge_search_route_returns_results_from_search_service()

### Community 30 - "Destination Context Routes"
Cohesion: 0.38
Nodes (3): get_destination_context(), _lookup_destination_context(), preview_destination_context_prompt()

### Community 31 - "Knowledge Chunker"
Cohesion: 0.52
Nodes (6): chunk_text(), _flush_paragraph(), _join_blocks(), _overlap_tail(), _split_blocks(), _split_long_block()

### Community 32 - "Service Config Tests"
Cohesion: 0.47
Nodes (3): TestLoadAppliesAIGenerationTimeoutDefaults(), TestLoadReadsCORSOverrides(), unsetEnv()

### Community 33 - "ChromaDB Client Tests"
Cohesion: 0.47
Nodes (4): FakeChromaSettings, _install_fake_chromadb(), test_create_persistent_chroma_client_can_enable_telemetry(), test_create_persistent_chroma_client_disables_telemetry_by_default()

### Community 34 - "Knowledge Chunker Tests"
Cohesion: 0.33
Nodes (0): 

### Community 35 - "Ollama Embedding Tests"
Cohesion: 0.6
Nodes (5): _settings(), test_embedding_connection_error_is_wrapped(), test_missing_embedding_response_raises_error(), test_non_2xx_embedding_response_raises_error(), test_successful_embedding_response_returns_float_list()

### Community 36 - "User Service Validation Tags"
Cohesion: 0.4
Nodes (1): TagOption

### Community 37 - "Knowledge Indexer Script"
Cohesion: 0.7
Nodes (4): _get_or_create_collection(), _iter_knowledge_files(), main(), _resolve_service_path()

### Community 38 - "Generator Factory"
Cohesion: 0.7
Nodes (4): get_destination_knowledge_provider(), get_itinerary_generator(), get_knowledge_search_service(), _resolve_destination_context_dir()

### Community 39 - "JWT Claims"
Cohesion: 0.5
Nodes (2): accessClaims, jwtPayload

### Community 40 - "Knowledge Search Routes"
Cohesion: 0.5
Nodes (0): 

### Community 41 - "AI Service Operational Scripts"
Cohesion: 1.0
Nodes (1): Operational scripts for the AI Planning Service.

### Community 42 - "Chroma Client Bootstrap"
Cohesion: 1.0
Nodes (0): 

### Community 43 - "Next.js Env Types"
Cohesion: 1.0
Nodes (0): 

### Community 44 - "Tailwind Config"
Cohesion: 1.0
Nodes (0): 

### Community 45 - "PostCSS Config"
Cohesion: 1.0
Nodes (0): 

### Community 46 - "AI Service Dev Dependencies"
Cohesion: 1.0
Nodes (1): AI service dev dependencies (pytest, ruff, httpx)

## Knowledge Gaps
- **116 isolated node(s):** `container`, `readinessBody`, `readinessDB`, `Trip`, `ListTrips` (+111 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `AI Service Operational Scripts`** (2 nodes): `__init__.py`, `Operational scripts for the AI Planning Service.`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Chroma Client Bootstrap`** (2 nodes): `chroma_client.py`, `create_persistent_chroma_client()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Next.js Env Types`** (1 nodes): `next-env.d.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Tailwind Config`** (1 nodes): `tailwind.config.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `PostCSS Config`** (1 nodes): `postcss.config.js`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `AI Service Dev Dependencies`** (1 nodes): `AI service dev dependencies (pytest, ruff, httpx)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Settings` connect `Itinerary Domain Models` to `Service Configuration`, `Knowledge Search Tests`, `Knowledge Routes Tests`, `ChromaDB Client Tests`?**
  _High betweenness centrality (0.134) - this node is a cross-community bridge._
- **Why does `ItineraryGenerationError` connect `Itinerary Domain Models` to `Service Bootstrap & Schema Checks`?**
  _High betweenness centrality (0.117) - this node is a cross-community bridge._
- **Why does `register()` connect `Project Documentation & Architecture` to `Auth Service & Web Auth Integration`?**
  _High betweenness centrality (0.103) - this node is a cross-community bridge._
- **Are the 20 inferred relationships involving `OllamaItineraryGenerator` (e.g. with `Settings` and `ItineraryGenerationError`) actually correct?**
  _`OllamaItineraryGenerator` has 20 INFERRED edges - model-reasoned connections that need verification._
- **Are the 15 inferred relationships involving `MockItineraryGenerator` (e.g. with `GenerateItineraryRequest` and `ItineraryDay`) actually correct?**
  _`MockItineraryGenerator` has 15 INFERRED edges - model-reasoned connections that need verification._
- **What connects `container`, `readinessBody`, `readinessDB` to the rest of the system?**
  _116 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Itinerary Domain Models` be split into smaller, more focused modules?**
  _Cohesion score 0.03 - nodes in this community are weakly interconnected._