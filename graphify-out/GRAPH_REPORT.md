# Graph Report - .  (2026-06-23)

## Corpus Check
- 218 files · ~91,084 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 1232 nodes · 2413 edges · 44 communities detected
- Extraction: 95% EXTRACTED · 5% INFERRED · 0% AMBIGUOUS · INFERRED: 130 edges (avg confidence: 0.5)
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
- `Raised when the Ollama embeddings API cannot provide a usable embedding.` --uses--> `Settings`  [INFERRED]
  services/ai-planning-service/app/services/ollama_embedding_client.py → services/ai-planning-service/app/config.py
- `KnowledgeSearchService` --uses--> `Settings`  [INFERRED]
  services/ai-planning-service/app/services/knowledge_search.py → services/ai-planning-service/app/config.py
- `FakeEmbeddingClient` --uses--> `Settings`  [INFERRED]
  services/ai-planning-service/tests/test_knowledge_search.py → services/ai-planning-service/app/config.py
- `FakeCollection` --uses--> `Settings`  [INFERRED]
  services/ai-planning-service/tests/test_knowledge_search.py → services/ai-planning-service/app/config.py
- `FakeChromaClient` --uses--> `Settings`  [INFERRED]
  services/ai-planning-service/tests/test_knowledge_search.py → services/ai-planning-service/app/config.py

## Communities

### Community 0 - "Community 0"
Cohesion: 0.03
Nodes (68): Itinerary, ItineraryDay, ItineraryItem, PlaceRef, BaseModel, Settings, DestinationContext, DestinationContextListResponse (+60 more)

### Community 1 - "Community 1"
Cohesion: 0.03
Nodes (32): Place, User, addDay(), addItem(), attachPlace(), getMoveTargetDayIndex(), moveItem(), moveItemAcrossDays() (+24 more)

### Community 2 - "Community 2"
Cohesion: 0.03
Nodes (61): CreateTripInput, RegenerateItineraryPartInput, UpdateItineraryInput, Status, Trip, rowQuerier, CreateTrip, RegenerateItineraryPart (+53 more)

### Community 3 - "Community 3"
Cohesion: 0.03
Nodes (41): AuthApiError, authFetch(), bearerToken(), buildAuthUrl(), login(), logout(), me(), NewAuth() (+33 more)

### Community 4 - "Community 4"
Cohesion: 0.04
Nodes (37): App, container, New(), warnWeakDevelopmentSecret(), GenerateItineraryInput, ItineraryGenerator, RegenerateDayInput, RegenerateItemInput (+29 more)

### Community 5 - "Community 5"
Cohesion: 0.13
Nodes (51): mockGenerator, mockUserContextProvider, assertInvalidInput(), authContext(), authContextWithToken(), decodeItinerary(), itineraryWithMutatedPlaceRaw(), newTestService() (+43 more)

### Community 6 - "Community 6"
Cohesion: 0.08
Nodes (25): decodeJSON(), writeError(), writeJSON(), OptionalFloat64, PatchPreferencesInput, UpdateProfileInput, Profile, errorBody (+17 more)

### Community 7 - "Community 7"
Cohesion: 0.07
Nodes (31): memoryRepository, mockRepo, assertInvalidInput(), assertStrings(), authContext(), defaultProfile(), newMemoryRepository(), newTestService() (+23 more)

### Community 8 - "Community 8"
Cohesion: 0.05
Nodes (25): AuthConfig, Config, CORSConfig, _env_bool(), _env_float(), _env_int(), _env_string(), get_settings() (+17 more)

### Community 9 - "Community 9"
Cohesion: 0.05
Nodes (31): MiddlewareConfig, corsMiddleware(), splitCSVSet(), aiPlanningGenerateRequest, AIPlanningHTTPGenerator, aiPlanningRegenerateDayRequest, aiPlanningRegenerateDayResponse, aiPlanningRegenerateItemRequest (+23 more)

### Community 10 - "Community 10"
Cohesion: 0.05
Nodes (12): newTestRouter(), TestHandlerInvalidJSONErrorShape(), TestHandlerInvalidLoginReturnsUnauthorized(), TestHandlerMeReturnsCurrentUser(), TestHandlerRegisterDuplicateReturnsConflict(), SearchPlacesResponse, stubAuthService, Repository (+4 more)

### Community 11 - "Community 11"
Cohesion: 0.1
Nodes (20): routeTestGenerator, routeTestRepo, newAuthTestRouter(), routeTestNextVersionNumber(), signAccessToken(), TestAuthDisabledUsesDevUserID(), TestHealthAndReadyRemainPublic(), TestItineraryVersionHistoryOwnerCanPreviewRestoreAndNonOwnerReceives404() (+12 more)

### Community 12 - "Community 12"
Cohesion: 0.12
Nodes (21): PlacesHandler, getPlaceDetails(), placeFetch(), searchPlaces(), writeError(), writeJSON(), newTestRouter(), performRequest() (+13 more)

### Community 13 - "Community 13"
Cohesion: 0.14
Nodes (16): _clean_metadata(), _distance_to_similarity(), _first_result_list(), KnowledgeSearchService, _normalize_destination(), _resolve_service_path(), FakeChromaClient, FakeCollection (+8 more)

### Community 14 - "Community 14"
Cohesion: 0.18
Nodes (27): _itinerary_body(), _request(), _settings(), test_generated_response_keeps_exact_itinerary_response_shape(), test_initial_invalid_json_and_repair_disabled_falls_back_when_fallback_enabled(), test_initial_invalid_json_and_repair_enabled_triggers_repair(), test_log_llm_payloads_false_does_not_log_full_prompt_or_response(), test_log_llm_payloads_true_in_development_allows_payload_logging() (+19 more)

### Community 15 - "Community 15"
Cohesion: 0.14
Nodes (18): readinessDB, ReadinessHandler, assertCapturedPayload(), newTestHTTPGenerator(), TestAIPlanningHTTPGeneratorGenerate_DefaultsRequestPayload(), TestAIPlanningHTTPGeneratorGenerate_EmptyDaysReturnsError(), TestAIPlanningHTTPGeneratorGenerate_InvalidJSONReturnsError(), TestAIPlanningHTTPGeneratorGenerate_Non2xxReturnsError() (+10 more)

### Community 16 - "Community 16"
Cohesion: 0.1
Nodes (7): partial_payload(), test_regenerate_day_instruction_too_long_returns_400(), test_regenerate_day_invalid_day_number_returns_400(), test_regenerate_day_success_returns_replacement_day_only(), test_regenerate_item_accepts_optional_user_context(), test_regenerate_item_invalid_item_index_returns_400(), test_regenerate_item_success_returns_replacement_item_only()

### Community 17 - "Community 17"
Cohesion: 0.12
Nodes (13): ApiError, apiFetch(), apiFetchInternal(), buildApiUrl(), isMissing(), NewClient(), normalizeBaseURL(), notifySessionExpired() (+5 more)

### Community 18 - "Community 18"
Cohesion: 0.28
Nodes (22): _assert_validation_code(), _itinerary(), _itinerary_body(), _request(), test_avoid_term_warning_does_not_fail_validation(), test_budget_exceeded_by_more_than_thirty_percent_fails_with_budget_exceeded(), test_budget_slightly_above_requested_amount_within_thirty_percent_passes(), test_dietary_restriction_warning_does_not_fail_validation() (+14 more)

### Community 19 - "Community 19"
Cohesion: 0.27
Nodes (19): _append_optional_line(), build_itinerary_prompt(), build_regenerate_day_prompt(), build_regenerate_day_repair_prompt(), build_regenerate_item_prompt(), build_regenerate_item_repair_prompt(), build_repair_prompt(), _compact_content() (+11 more)

### Community 20 - "Community 20"
Cohesion: 0.18
Nodes (11): MockItineraryGenerator, floatPtr(), intPtr(), mock(), mockPlaceMatches(), mockPlaces(), NewMockPlaceProvider(), normalizeSearchText() (+3 more)

### Community 21 - "Community 21"
Cohesion: 0.32
Nodes (12): _client(), test_destination_context_disabled_preview_prompt_returns_prompt_without_context(), test_destination_context_disabled_returns_404_for_destination(), test_destination_context_disabled_returns_empty_list(), test_generate_itinerary_still_works_after_adding_destination_context_routes(), test_get_destination_context_by_destination_returns_404_when_missing(), test_get_destination_context_by_destination_returns_context_when_found(), test_get_destination_context_returns_list_response() (+4 more)

### Community 22 - "Community 22"
Cohesion: 0.38
Nodes (13): _ensure_exact_day_response_shape(), _ensure_exact_item_response_shape(), _ensure_exact_item_shape(), _ensure_exact_response_shape(), _ensure_item_values_valid(), _extract_first_json_object(), LLMResponseParseError, parse_itinerary_response() (+5 more)

### Community 23 - "Community 23"
Cohesion: 0.22
Nodes (3): canMoveItemToDay(), moveItemToDay(), normalizeItineraryDays()

### Community 24 - "Community 24"
Cohesion: 0.18
Nodes (0): 

### Community 25 - "Community 25"
Cohesion: 0.38
Nodes (7): _rag_chunks(), _request(), test_itinerary_prompt_includes_rag_context_when_chunks_exist(), test_itinerary_prompt_omits_user_context_when_not_provided(), test_itinerary_prompt_preserves_json_schema_instructions_with_rag(), test_itinerary_prompt_works_without_rag_chunks(), test_repair_prompt_includes_rag_context()

### Community 26 - "Community 26"
Cohesion: 0.22
Nodes (2): abs32(), fromPgNumericPtr()

### Community 27 - "Community 27"
Cohesion: 0.46
Nodes (1): FileDestinationKnowledgeProvider

### Community 28 - "Community 28"
Cohesion: 0.38
Nodes (3): get_destination_context(), _lookup_destination_context(), preview_destination_context_prompt()

### Community 29 - "Community 29"
Cohesion: 0.52
Nodes (6): chunk_text(), _flush_paragraph(), _join_blocks(), _overlap_tail(), _split_blocks(), _split_long_block()

### Community 30 - "Community 30"
Cohesion: 0.47
Nodes (3): TestLoadAppliesAIGenerationTimeoutDefaults(), TestLoadReadsCORSOverrides(), unsetEnv()

### Community 31 - "Community 31"
Cohesion: 0.47
Nodes (4): FakeChromaSettings, _install_fake_chromadb(), test_create_persistent_chroma_client_can_enable_telemetry(), test_create_persistent_chroma_client_disables_telemetry_by_default()

### Community 32 - "Community 32"
Cohesion: 0.33
Nodes (0): 

### Community 33 - "Community 33"
Cohesion: 0.6
Nodes (5): _settings(), test_embedding_connection_error_is_wrapped(), test_missing_embedding_response_raises_error(), test_non_2xx_embedding_response_raises_error(), test_successful_embedding_response_returns_float_list()

### Community 34 - "Community 34"
Cohesion: 0.4
Nodes (1): TagOption

### Community 35 - "Community 35"
Cohesion: 0.7
Nodes (4): _get_or_create_collection(), _iter_knowledge_files(), main(), _resolve_service_path()

### Community 36 - "Community 36"
Cohesion: 0.7
Nodes (4): get_destination_knowledge_provider(), get_itinerary_generator(), get_knowledge_search_service(), _resolve_destination_context_dir()

### Community 37 - "Community 37"
Cohesion: 0.5
Nodes (2): accessClaims, jwtPayload

### Community 38 - "Community 38"
Cohesion: 0.5
Nodes (0): 

### Community 39 - "Community 39"
Cohesion: 1.0
Nodes (1): Operational scripts for the AI Planning Service.

### Community 40 - "Community 40"
Cohesion: 1.0
Nodes (0): 

### Community 41 - "Community 41"
Cohesion: 1.0
Nodes (0): 

### Community 42 - "Community 42"
Cohesion: 1.0
Nodes (0): 

### Community 43 - "Community 43"
Cohesion: 1.0
Nodes (0): 

## Knowledge Gaps
- **88 isolated node(s):** `container`, `readinessBody`, `readinessDB`, `Trip`, `ListTrips` (+83 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Community 39`** (2 nodes): `__init__.py`, `Operational scripts for the AI Planning Service.`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 40`** (2 nodes): `chroma_client.py`, `create_persistent_chroma_client()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 41`** (1 nodes): `next-env.d.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 42`** (1 nodes): `tailwind.config.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 43`** (1 nodes): `postcss.config.js`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Settings` connect `Community 0` to `Community 8`, `Community 13`, `Community 31`?**
  _High betweenness centrality (0.136) - this node is a cross-community bridge._
- **Why does `ItineraryGenerationError` connect `Community 0` to `Community 4`?**
  _High betweenness centrality (0.126) - this node is a cross-community bridge._
- **Why does `OllamaItineraryGenerator` connect `Community 0` to `Community 13`, `Community 22`?**
  _High betweenness centrality (0.062) - this node is a cross-community bridge._
- **Are the 20 inferred relationships involving `OllamaItineraryGenerator` (e.g. with `Settings` and `ItineraryGenerationError`) actually correct?**
  _`OllamaItineraryGenerator` has 20 INFERRED edges - model-reasoned connections that need verification._
- **Are the 15 inferred relationships involving `MockItineraryGenerator` (e.g. with `GenerateItineraryRequest` and `ItineraryDay`) actually correct?**
  _`MockItineraryGenerator` has 15 INFERRED edges - model-reasoned connections that need verification._
- **What connects `container`, `readinessBody`, `readinessDB` to the rest of the system?**
  _88 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Community 0` be split into smaller, more focused modules?**
  _Cohesion score 0.03 - nodes in this community are weakly interconnected._