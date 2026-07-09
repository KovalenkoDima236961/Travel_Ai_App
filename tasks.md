# 5. AI Trip Discovery v1: create a beautiful AI-powered trip creation flow where users can describe their desired trip, get destination suggestions based on preferences and previous trips, use a smart “Surprise me” button, refine bad suggestions, and create a trip from the selected destination.

You are a senior full-stack engineer and product-minded AI architect. Continue building the web-based AI travel planning application.

Your task:
Implement AI Trip Discovery v1: create a beautiful AI-powered trip creation flow where users can describe their desired trip, get destination suggestions based on preferences and previous trips, use a smart “Surprise me” button, refine bad suggestions, and create a trip from the selected destination.

Context:
We already have a microservices-based AI travel planning app.

Existing services:

- Auth Service:
  - Go microservice
  - issues JWT access tokens and refresh tokens
- User/Profile Service:
  - Go microservice
  - stores user profile/preferences
  - stores preferredLanguage
  - owns workspace membership and workspace roles
- Trip Service:
  - Go microservice
  - owns trips, workspace trips, trip creation, itinerary generation jobs, AI template adaptation jobs, AI repair jobs, budgets, workspace budgets, cost analytics, cost splitting, approval workflow, workspace policies, approval risk scoring, comments, activity, version history, templates, conflict detection, notifications integration, and permissions
  - supports personal trips and workspace trips
  - calls AI Planning Service for generation/regeneration/optimization/adaptation/repair
- Worker Service:
  - Go microservice
  - processes RabbitMQ-backed jobs
- Notification Service:
  - Go microservice
  - owns in-app/email/web-push/SSE notifications and notification preferences
- External Integrations Service:
  - Go microservice
  - owns places, routes, weather, calendar, exchange rates, prices, availability provider adapters, quota/rate limits
- AI Planning Service:
  - Python FastAPI service
  - supports itinerary generation, partial regeneration, budget optimization, template adaptation, policy-aware repair, destination context/RAG, validation/repair, multilingual output, and Ollama/mock modes
- Web App:
  - Next.js app under apps/web
  - supports auth, trips, workspace switcher, workspace pages, templates, AI template adaptation, budgets, workspace budgets, cost analytics, cost splitting, approval workflow, workspace policies, approval risk scoring, AI repair proposals, availability cards, exports, offline mode, PWA install, notifications, internationalization, etc.
- Infra:
  - Postgres
  - RabbitMQ
  - Prometheus/Grafana

Current create trip behavior:

- The app supports a normal form-based create trip flow.
- This works when the user already knows the destination.
- The page feels boring and does not help users who do not know where they want to go.

Problem:
Many users do not start with a fixed destination. They start with a vague idea:

- “I want a cheap weekend trip.”
- “I want something warm with good food.”
- “I want mountains and not too much walking.”
- “Surprise me.”
- “Something like my Prague trip, but new.”
- “Find me a 4-day trip in September under €700.”

Goal:
Add an AI-powered discovery flow:

- User can choose between:
  1. “I know where I want to go” → existing form
  2. “Help me choose” → new AI discovery experience
- User can write a natural-language prompt.
- User can use quick chips.
- User can press a smart “Surprise me” button.
- Trip Service combines user prompt, preferences, previous trips, language, budget, season, workspace policy, and existing trips.
- AI Planning Service suggests 3–5 destination ideas.
- Each suggestion explains why it fits, possible downsides, estimated budget, best duration, and trip preview.
- User can refine suggestions:
  - cheaper
  - warmer
  - more nature
  - more city
  - less walking
  - different country
  - similar places
  - not this vibe
- User selects a destination.
- App creates a draft trip from the suggestion and optionally starts itinerary generation.
- Do not automatically create a trip from “Surprise me” without user confirmation.

Do NOT add:

- real flight search
- real hotel booking
- automatic booking
- visa/legal guarantees
- health/safety guarantees
- payments
- full ML ranking model
- destination marketplace
- public destination database admin
- external travel recommendation APIs in v1
- complex conversational memory engine
- native mobile
- Kubernetes
- new backend service

For v1:

- Implement AI destination suggestions using existing AI Planning Service.
- Implement backend orchestration in Trip Service.
- Use existing user profile/preferences and previous trips.
- Use mock mode for deterministic tests.
- Store discovery sessions/suggestions if useful.
- Build a polished Web App experience.
- Keep existing create trip form working.
- English fallback must work.
- Support existing selected language/outputLanguage.
- Workspace policy should guide suggestions when creating workspace trips.
- User must confirm before a trip is created.

Important codebase consistency requirement:
Before implementing, inspect existing services and follow the same patterns exactly:

- apps/web
- services/trip-service
- services/user-service
- services/ai-planning-service
- services/worker-service
- services/external-integrations-service
- services/notification-service

Do not invent a different architecture if the repository already has conventions.

Match existing patterns for:

- Next.js routing/app router
- layouts/providers
- i18n
- frontend API clients/hooks
- TanStack Query
- forms/validation
- toasts/error handling
- trip creation
- trip generation jobs
- workspace permission checks
- workspace policy constraints
- user profile/preference fetching
- Go service modules
- Uber Fx modules
- Zap logging
- config loading
- HTTP middleware
- auth/JWT middleware
- response/error helpers
- pgxpool/sqlc
- migrations
- sqlc queries
- AI request clients
- FastAPI route structure
- Pydantic schemas
- Ollama/mock mode
- AI prompt builder
- tests
- smoke scripts
- docs

Part 1: Product flow

1. Update Create Trip page.

The create trip page should offer two modes:

Mode A:
“I know where I want to go”

- Existing form-based flow.

Mode B:
“Help me choose”

- New AI Trip Discovery flow.

Recommended hero copy:

- “Where should we go next?”
- “Describe your ideal trip and we’ll suggest destinations that fit you.”

Input examples:

- “A cheap 3-day trip with good food and warm weather.”
- “Mountains, nature, and not too much walking.”
- “Something romantic for a long weekend.”
- “A city break similar to Prague but new.”

2. Discovery actions.

Support:

- Prompt-based discovery.
- Smart Surprise Me.
- Refine suggestions.
- Create trip from selected suggestion.

3. User confirmation.

Never create a trip immediately after pressing “Surprise me.”
Flow:
Surprise me → show destination suggestion(s) → user confirms → create trip.

Part 2: AI Planning Service endpoint

4. Add endpoint:

POST /suggest-destinations

Request:

{
"prompt": "I want a cheap 3-day trip somewhere warm with good food.",
"mode": "prompt" | "surprise" | "refine",
"outputLanguage": "en" | "es" | "uk" | "fr",
"userContext": {
"homeCity": "Bratislava",
"homeCountry": "Slovakia",
"preferredCurrency": "EUR",
"preferredLanguage": "uk",
"preferences": {
"travelStyles": ["food", "city_break"],
"pace": "balanced",
"maxWalkingKmPerDay": 10,
"foodPreferences": ["local food"],
"avoid": ["nightclubs"],
"preferredTransport": ["train", "public_transport"]
}
},
"tripContext": {
"durationDays": 3,
"startDate": "2026-09-10",
"dateFlexibility": "flexible",
"budget": {
"amount": 700,
"currency": "EUR"
},
"travelers": 2,
"origin": "Bratislava, Slovakia",
"scope": "personal" | "workspace"
},
"previousTrips": [
{
"destination": "Prague",
"country": "Czechia",
"durationDays": 3,
"budget": {
"amount": 450,
"currency": "EUR"
},
"tags": ["city", "food", "architecture"],
"likedSignals": ["walkable city", "good food"],
"createdAt": "2026-05-12"
}
],
"workspacePolicyConstraints": {
"summary": "Keep total budget under 700 EUR. Avoid late activities after 22:00.",
"rules": {}
},
"refinement": {
"previousSuggestions": [],
"selectedSuggestionId": "optional",
"instruction": "Cheaper and more nature."
},
"constraints": {
"suggestionCount": 5,
"avoidPreviouslyVisited": true,
"preferNovelty": true,
"includeReasoning": true,
"maxTravelComplexity": "medium"
}
}

Response:

{
"sessionTitle": "Warm budget food trips",
"suggestions": [
{
"id": "stable-id-or-generated",
"destination": "Valencia, Spain",
"city": "Valencia",
"country": "Spain",
"region": "Valencian Community",
"matchScore": 87,
"recommendedDurationDays": 4,
"bestFor": ["food", "architecture", "warm weather"],
"estimatedBudget": {
"amount": 520,
"currency": "EUR",
"confidence": "medium"
},
"bestTimeToGo": "Spring or early autumn",
"whyItFits": "You like walkable city trips with strong food culture and moderate budgets.",
"possibleDownsides": [
"Can be hot in August.",
"Flights or long train connections may affect budget."
],
"tripPreview": {
"title": "Valencia food and architecture escape",
"summary": "A relaxed city break with markets, old town walks, paella, and beach time.",
"sampleDay": [
"Central Market and old town walk",
"Turia Gardens",
"Paella dinner"
]
},
"tags": ["food", "city_break", "warm", "architecture"],
"suggestedPromptForItinerary": "Create a 4-day Valencia food and architecture trip with relaxed pace and a 520 EUR budget.",
"concerns": [
{
"type": "budget_uncertainty",
"message": "Transport cost from your origin is not verified."
}
]
}
],
"followUpQuestions": [
"Do you prefer beach cities or historic cities?"
],
"warnings": [
"Budgets are rough estimates and do not include live flight/hotel prices."
]
}

5. Pydantic schemas.

Create:

- DestinationSuggestionRequest
- DestinationSuggestionMode
- DestinationUserContext
- DestinationTripContext
- PreviousTripSummary
- DestinationRefinementContext
- DestinationSuggestionResponse
- DestinationSuggestion
- DestinationBudgetEstimate
- DestinationTripPreview
- DestinationConcern

6. Modes.

Support:

- prompt
- surprise
- refine

Prompt mode:

- Use user prompt heavily.

Surprise mode:

- If prompt is empty, use user preferences, previous trips, and novelty.
- Should produce smart-random suggestions, not random city names.
- Add some variety while staying plausible.

Refine mode:

- Use previous suggestions and refinement instruction.
- Avoid repeating rejected suggestion unless asking for similar places.

7. Mock mode.

Mock mode must be deterministic and language-aware.

For prompt mode:

- Return fixed suggestions based on prompt keywords:
  - warm/food → Valencia, Naples, Lisbon
  - mountains/nature → Salzburg, Ljubljana, Innsbruck
  - cheap/weekend → Kraków, Budapest, Brno
  - museums/culture → Vienna, Paris, Florence
  - beach → Valencia, Nice, Split

For surprise mode:

- Use user preferences and previous trips:
  - If previous trip includes Prague, suggest Vienna/Kraków/Ljubljana but avoid Prague.
  - If user likes food, include Valencia/Naples/Lisbon.
  - If user likes nature, include Salzburg/Ljubljana.
- Deterministic ordering.

For refine mode:

- If instruction includes cheaper, return cheaper alternatives.
- If warmer, return warmer alternatives.
- If nature, return more nature-heavy alternatives.
- If city, return city-break alternatives.

8. Ollama mode.

Add prompt builder for destination suggestions.

The prompt must instruct:

- Return strict JSON only.
- Do not claim real-time prices or availability.
- Use rough estimates only.
- Consider user preferences and previous trips.
- Avoid suggesting the same destination if avoidPreviouslyVisited is true.
- Explain why each suggestion fits.
- Include possible downsides.
- Include suggestedPromptForItinerary.
- Keep JSON keys/enums in English.
- Localize user-facing text values to outputLanguage.
- Avoid unsafe or illegal travel suggestions.
- Do not provide visa/legal/health guarantees.

9. Language behavior.

Use outputLanguage:

- User-facing text values should be localized.
- destination/city/country names can use common names for that language where natural.
- JSON keys/enums stay English.

Part 3: Trip Service data model

10. Decide whether to persist discovery sessions.

Recommended v1:
Persist sessions and suggestions so user can refine, revisit, and create from suggestion.

Add migration:

trip_discovery_sessions:

- id UUID primary key
- user_id UUID not null
- workspace_id UUID null
- mode TEXT not null
- prompt TEXT null
- output_language TEXT not null default 'en'
- status TEXT not null default 'completed'
- request_json JSONB not null
- response_json JSONB not null
- created_trip_id UUID null
- created_at TIMESTAMP not null default NOW()
- updated_at TIMESTAMP not null default NOW()

Constraints:

- mode in ('prompt', 'surprise', 'refine')
- status in ('completed', 'failed', 'created_trip')
- output_language in ('en','es','uk','fr')

Indexes:

- user_id, created_at desc
- workspace_id, created_at desc
- created_trip_id

Optional:
trip_discovery_feedback:

- id UUID primary key
- session_id UUID not null
- suggestion_id TEXT not null
- user_id UUID not null
- feedback_type TEXT not null
- feedback_text TEXT null
- created_at TIMESTAMP not null default NOW()

feedback_type:

- not_for_me
- too_expensive
- too_far
- too_much_walking
- warmer
- colder
- more_nature
- more_city
- similar
- accepted

If feedback table is too much for v1:

- store refinement history in request_json.

11. SQL/sqlc queries.

Add:

- CreateTripDiscoverySession
- GetTripDiscoverySessionByID
- ListTripDiscoverySessionsByUser
- MarkTripDiscoverySessionCreatedTrip
- CreateTripDiscoveryFeedback optional
- ListRecentDiscoverySessions optional

Part 4: Trip Service discovery module

12. Add module.

Create:

services/trip-service/internal/tripdiscovery/

Suggested files:

- types.go
- dto.go
- service.go
- handler.go
- repository.go
- ai_client.go
- context_builder.go
- previous_trips.go
- module.go
- errors.go

Adjust to repo conventions.

13. Endpoints.

Add:

POST /trip-discovery/suggestions
POST /trip-discovery/surprise-me
POST /trip-discovery/{sessionId}/refine
POST /trip-discovery/{sessionId}/suggestions/{suggestionId}/create-trip
GET /trip-discovery/sessions
GET /trip-discovery/sessions/{sessionId}

Alternative:
Use one endpoint with mode.
But explicit endpoints are easier for frontend.

14. POST /trip-discovery/suggestions.

Request:

{
"prompt": "I want a cheap 3-day trip somewhere warm with good food.",
"scope": "personal" | "workspace",
"workspaceId": "uuid-or-null",
"durationDays": 3,
"startDate": "2026-09-10",
"dateFlexibility": "flexible",
"budget": {
"amount": 700,
"currency": "EUR"
},
"travelers": 2,
"origin": "Bratislava, Slovakia",
"quickChips": ["warm", "food", "low_budget"],
"outputLanguage": "uk",
"avoidPreviouslyVisited": true,
"preferNovelty": true
}

Validation:

- prompt optional but required for suggestions endpoint unless quickChips present.
- prompt max 1000.
- durationDays optional 1–30.
- startDate optional valid date.
- budget optional amount >= 0 currency 3 letters.
- travelers optional 1–50.
- quickChips max 20.
- outputLanguage supported.
- workspaceId required if scope=workspace.

15. POST /trip-discovery/surprise-me.

Request:

{
"scope": "personal" | "workspace",
"workspaceId": "uuid-or-null",
"durationDays": 3,
"startDate": null,
"budget": {
"amount": 500,
"currency": "EUR"
},
"travelers": 1,
"origin": "Bratislava, Slovakia",
"outputLanguage": "en",
"noveltyLevel": "balanced"
}

noveltyLevel:

- safe
- balanced
- adventurous

Behavior:

- No prompt required.
- Build suggestions from profile/preferences/previous trips.
- Return suggestions; do not create trip.

16. POST /trip-discovery/{sessionId}/refine.

Request:

{
"instruction": "Make it cheaper and more nature-focused.",
"selectedSuggestionId": "valencia-spain",
"feedbackType": "too_expensive",
"outputLanguage": "uk"
}

Validation:

- instruction required, max 1000.
- feedbackType optional enum.
- selectedSuggestionId optional.

Behavior:

- Load previous session.
- Check owner/access.
- Build refine request using previous suggestions.
- Create new discovery session linked to previous session if schema supports.
- Return new suggestions.

Optional DB:

- add parent_session_id UUID null to trip_discovery_sessions.

Recommended:

- add parent_session_id.

17. POST /trip-discovery/{sessionId}/suggestions/{suggestionId}/create-trip.

Request:

{
"title": "Valencia food weekend",
"startDate": "2026-09-10",
"durationDays": 4,
"budget": {
"amount": 520,
"currency": "EUR"
},
"travelers": 2,
"workspaceId": null,
"autoGenerateItinerary": true
}

Behavior:

1. Load session and suggestion.
2. Check user owns session.
3. Check workspace permission if workspaceId provided.
4. Create draft trip with:
   - destination from suggestion
   - title from request or suggestion tripPreview title
   - startDate/duration/budget/travelers
   - language from session/outputLanguage/user preference
   - source metadata:
     {
     "source": "trip_discovery",
     "sessionId": "...",
     "suggestionId": "...",
     "matchScore": 87
     }
5. If autoGenerateItinerary:
   - create generation job using suggestion.suggestedPromptForItinerary
   - include destination and trip context
   - return trip + job
6. Mark session status created_trip and created_trip_id.

Response:

{
"trip": {...},
"generationJob": {...}
}

18. GET sessions.

GET /trip-discovery/sessions?limit=20

Return recent sessions for current user.

19. Permissions.

Personal discovery:

- authenticated user only.

Workspace discovery:

- user must be active workspace member.
- to create workspace trip from suggestion, user must have role owner/admin/member.
- viewer can maybe generate suggestions but cannot create workspace trip.
  Recommended:
- viewer can view/use discovery read-only? Simpler:
  - viewer cannot create discovery for workspace.
  - member/admin/owner can.

20. User context builder.

Trip Service should gather:

- user profile
- user preferences
- preferred language/currency
- recent trips
- previous destinations
- previous trip durations/budgets/tags
- liked templates if available
- workspace policy if workspace scope
- origin/homeCity

Limit previous trips:

- last 10–20 trips.
- Do not send full itineraries.
- Send summaries only.

21. Previous trip summary.

Build:
{
"destination": "Prague",
"country": "Czechia",
"durationDays": 3,
"budget": {"amount": 450, "currency": "EUR"},
"tags": ["city", "food", "architecture"],
"pace": "balanced",
"createdAt": "..."
}

Do not send:

- comments
- collaborators
- share tokens
- calendar sync IDs
- raw provider data
- private notes
- full itinerary unless summarized.

22. Workspace policy constraints.

For workspace scope:

- fetch active workspace policy.
- convert to AI constraints using existing policy helper.
- include in AI request.

23. Output language.

Determine:

1. request.outputLanguage if provided.
2. user preferredLanguage.
3. en.

Created trip should store language if trip.language exists.

Part 5: Trip Service AI client

24. Extend AI client.

Add:
SuggestDestinations(ctx, request) response.

Use existing:

- base URL
- timeout
- logging
- error handling
- retries if any

Config:

- TRIP_DISCOVERY_ENABLED=true
- TRIP_DISCOVERY_AI_TIMEOUT_SECONDS=120
- TRIP_DISCOVERY_MAX_PREVIOUS_TRIPS=15
- TRIP_DISCOVERY_DEFAULT_SUGGESTION_COUNT=5

25. Error handling.

If AI fails:

- return controlled error:
  - trip_discovery_failed
- In mock/local mode, should not fail.

Do not create session with empty invalid suggestions unless status failed is useful.
Recommended:

- Store failed session only if existing pattern stores failed job/session.

Part 6: Create trip from suggestion

26. Trip source metadata.

Add to trip metadata or existing column:
{
"creationSource": "trip_discovery",
"discoverySessionId": "uuid",
"discoverySuggestionId": "string",
"discoveryMatchScore": 87,
"discoveryPrompt": "I want a cheap 3-day trip..."
}

Do not expose full previous trip context in trip metadata.

27. Generation prompt.

When autoGenerateItinerary is true:

- Use suggestion.suggestedPromptForItinerary.
- Include original user prompt/refinement as additional context.
- Include destination, duration, budget, travelers, language.
- Include workspace policy if workspace trip.

28. Activity events.

Add:

- trip_discovery_suggestions_created
- trip_created_from_discovery
- trip_discovery_refined

Activity metadata should be safe:

- sessionId
- suggestionId
- destination
- matchScore
- mode

Do not store full prompt in activity if it may include sensitive data; either omit or truncate.

29. Notifications.

No notifications needed for personal discovery.
For workspace trip created from discovery, use existing trip created notifications if any.

Part 7: Web App route and UI

30. Update route.

Existing:
apps/web/app/trips/new/page.tsx

Refactor to show two creation modes:

- Known destination
- Help me choose

Do not remove existing form.

31. New components.

Create:

apps/web/components/trip-discovery/TripCreateModeSelector.tsx
apps/web/components/trip-discovery/TripDiscoveryHero.tsx
apps/web/components/trip-discovery/TripDiscoveryPromptBox.tsx
apps/web/components/trip-discovery/TripDiscoveryQuickChips.tsx
apps/web/components/trip-discovery/SurpriseMeButton.tsx
apps/web/components/trip-discovery/DestinationSuggestionCard.tsx
apps/web/components/trip-discovery/DestinationSuggestionsGrid.tsx
apps/web/components/trip-discovery/TripDiscoveryRefineBar.tsx
apps/web/components/trip-discovery/CreateTripFromSuggestionDialog.tsx
apps/web/components/trip-discovery/DiscoverySessionHistory.tsx optional

32. Visual design.

The Help Me Choose screen should feel like an inspiration page, not a form.

Layout:

- large hero card
- prompt input
- quick chips
- surprise button
- suggestions as rich cards
- refine bar after suggestions

Example UI:

Title:
“Where should we go next?”

Subtitle:
“Describe your ideal trip, or let AI surprise you based on your preferences.”

Prompt placeholder:
“E.g. A cheap 3-day trip with warm weather, good food, and not too much walking…”

Quick chips:

- Weekend
- Warm weather
- Mountains
- Food trip
- Museums
- Low budget
- No flights
- Hidden gem
- Nature
- City break
- Romantic
- Family friendly
- Less walking

Buttons:

- Get suggestions
- Surprise me

33. Suggestion card content.

Each card should show:

- destination
- country
- match score
- tags
- estimated budget
- recommended duration
- why it fits
- possible downsides
- sample day/trip preview
- buttons:
  - Use this destination
  - Show similar
  - Not this vibe

34. Refine actions.

Provide quick refine buttons:

- Cheaper
- Warmer
- More nature
- More city
- Less walking
- Different country
- Similar places
- More hidden gem
- Better for food
- Better for museums

And free text:
“Tell us what to change…”

35. Create trip dialog.

When user clicks “Use this destination”:

- show dialog:
  - title
  - destination prefilled
  - startDate
  - durationDays
  - budget
  - travelers
  - scope/workspace
  - autoGenerateItinerary checkbox default true
- confirm creates trip.

After success:

- navigate to trip detail.
- If generation job started, show existing generation status UI.

36. Surprise Me UX.

Button behavior:

- if no preferences exist, still works but asks optional lightweight context:
  - budget
  - duration
  - origin
- if preferences exist, call surprise endpoint.
- show loading state:
  “Finding places that fit your travel style…”

37. Empty states.

If no suggestions:

- show friendly message.
- suggest changing budget/duration/prompt.
- offer normal create form.

38. Error states.

If AI fails:

- localized error.
- retry button.
- fallback suggestions optional from mock/static list only in local/dev.
- do not create trip.

39. Internationalization.

Add translation keys for all new UI in:

- en
- es
- uk
- fr

Namespace:
tripDiscovery

Include:

- hero title
- subtitles
- buttons
- chips
- card labels
- refine labels
- errors
- loading states
- create dialog labels

40. Accessibility.

- Prompt textarea has label.
- Buttons keyboard accessible.
- Suggestion cards have semantic headings.
- Match score not color-only.
- Loading states announced if existing pattern supports.

Part 8: Web App API/types/hooks

41. Types.

Create:

apps/web/types/trip-discovery.ts

Types:

- TripDiscoveryMode
- TripDiscoverySuggestion
- TripDiscoverySession
- TripDiscoveryRequest
- SurpriseMeRequest
- RefineDiscoveryRequest
- CreateTripFromSuggestionRequest
- TripDiscoveryResponse

42. API client.

Create:

apps/web/lib/api/trip-discovery.ts

Functions:

- getTripDiscoverySuggestions(input)
- surpriseMe(input)
- refineTripDiscovery(sessionId, input)
- createTripFromSuggestion(sessionId, suggestionId, input)
- listTripDiscoverySessions()
- getTripDiscoverySession(sessionId)

43. Hooks.

Create:

apps/web/hooks/useTripDiscoverySuggestions.ts
apps/web/hooks/useSurpriseMe.ts
apps/web/hooks/useRefineTripDiscovery.ts
apps/web/hooks/useCreateTripFromSuggestion.ts
apps/web/hooks/useTripDiscoverySessions.ts

Use TanStack Query/mutations:

- suggestions as mutation
- surprise as mutation
- refine as mutation
- create trip as mutation
- sessions as query optional

Invalidate:

- trips list after create.
- discovery sessions after create/refine.

Part 9: AI ranking/product rules

44. Match score.

AI returns matchScore 0–100.
Backend should clamp to 0–100.
Do not treat it as scientific.

UI label:
“Match score”
Tooltip:
“Estimated fit based on your prompt, preferences, and past trips.”

45. Budget estimate.

Budget is rough.
UI disclaimer:
“Estimated budget does not include live flight or hotel prices.”

46. Avoid repeated destinations.

If avoidPreviouslyVisited true:

- previous destinations should be discouraged.
- AI may still suggest similar but different destinations.

47. Novelty.

Surprise mode should balance:

- preference fit
- novelty
- feasibility
- budget
- travel complexity

Novelty levels:

- safe: close to known preferences
- balanced: mix familiar and new
- adventurous: more unusual suggestions

48. Bad suggestion recovery.

Every card should allow:

- Not this vibe
- Similar
- Cheaper
- Warmer
- More nature
  This is critical for trust.

Part 10: Backend tests

49. AI Planning Service tests.

Test:

- prompt mode returns valid suggestions in mock mode.
- surprise mode avoids previous destination.
- refine mode changes suggestions based on instruction.
- outputLanguage uk returns Ukrainian user-facing text.
- JSON keys/enums remain English.
- unsupported mode rejected.
- unsupported language rejected.
- prompt builder includes user preferences and previous trip summary but not private data.

50. Trip Service endpoint tests.

Test:

- authenticated user can request prompt suggestions.
- prompt too long rejected.
- unsupported outputLanguage rejected.
- surprise-me works without prompt.
- refine requires session ownership.
- non-owner cannot access/refine session.
- create trip from suggestion works.
- create trip does not happen during surprise-me.
- workspace viewer cannot create workspace trip from suggestion.
- workspace member can create workspace trip from suggestion.
- created trip stores discovery metadata.
- autoGenerateItinerary creates generation job.
- no autoGenerateItinerary creates draft only.
- created trip language matches outputLanguage/user preference.

51. Context builder tests.

Test:

- previous trips summarized and limited.
- private fields not included.
- workspace policy constraints included for workspace scope.
- user preferences included.
- default language/currency applied.

52. Repository tests.

Test:

- create session.
- get session by owner.
- list sessions.
- mark created trip.
- parent/refine session if implemented.

Part 11: Frontend tests

53. Component tests.

TripCreateModeSelector:

- switches between known destination and help me choose.

TripDiscoveryPromptBox:

- validates prompt.
- submits quick chips.

SurpriseMeButton:

- calls mutation.
- loading state.

DestinationSuggestionCard:

- renders destination, score, budget, tags, why, downsides.
- actions call callbacks.

TripDiscoveryRefineBar:

- quick refine buttons send expected instruction.
- free text works.

CreateTripFromSuggestionDialog:

- prefilled from suggestion.
- validates fields.
- submits create request.

54. Hook/API tests.

- suggestions mutation calls endpoint.
- surprise mutation calls endpoint.
- refine mutation calls endpoint.
- create trip mutation invalidates trips and navigates.

55. i18n tests.

- trip discovery UI renders in English, Spanish, Ukrainian, French.
- missing key falls back to English.

Part 12: Smoke tests

56. Update scripts/smoke-test.sh.

API smoke:

1. Login user.
2. Update profile preferences.
3. Create one previous trip to Prague.
4. POST /trip-discovery/surprise-me.
5. Assert suggestions returned.
6. Assert Prague is not first suggestion when avoidPreviouslyVisited true.
7. POST /trip-discovery/suggestions with prompt “cheap warm food weekend.”
8. Assert suggestions returned.
9. POST refine with “cheaper and more nature.”
10. Assert new session/suggestions returned.
11. POST create-trip from suggestion with autoGenerateItinerary=false.
12. Assert trip created with discovery metadata.
13. POST create-trip from another suggestion with autoGenerateItinerary=true.
14. Assert generation job created.
15. Try accessing another user’s session and assert forbidden.

16. Update scripts/web-smoke-test.md.

Manual test:

1. Open /trips/new.
2. Confirm two modes: known destination and help me choose.
3. Use prompt: “cheap 3-day warm food trip.”
4. Confirm suggestions appear as cards.
5. Click “Not this vibe” or “Cheaper.”
6. Confirm refined suggestions appear.
7. Click Surprise me.
8. Confirm no trip is created automatically.
9. Choose a suggestion.
10. Create trip with auto-generate enabled.
11. Confirm trip page opens and generation starts.
12. Switch UI language to Ukrainian and repeat; confirm UI and AI suggestion text are Ukrainian.

Part 13: Documentation

58. Update AI Planning Service README.

Document:

- /suggest-destinations endpoint
- request/response schema
- modes
- mock behavior
- language behavior
- limitations

59. Update Trip Service README.

Document:

- trip discovery endpoints
- discovery session model
- previous trip context rules
- create trip from suggestion
- permissions
- workspace behavior
- metadata
- limitations

60. Update Web App README.

Document:

- new Create Trip page modes
- prompt-based discovery
- Surprise Me
- refinement loop
- create trip from suggestion
- i18n keys

61. Update root README.md.

Mention:

- AI Trip Discovery v1.

62. User-facing limitations.

Document:

- suggestions are AI-generated estimates.
- budgets do not include live flight/hotel prices.
- destinations may not always be perfect.
- user should review before creating/generating itinerary.
- no booking is performed.
- no visa/legal/health guarantee.
- previous trips are summarized for personalization, not fully analyzed.

Part 14: Security and privacy

- Backend must enforce user/session ownership.
- Workspace permissions must be enforced.
- Do not send private comments/collaborators/share tokens/calendar IDs to AI.
- Do not send full previous itineraries unless summarized and sanitized.
- Do not log full prompts in production if they may contain sensitive data.
- Do not create trips automatically from Surprise Me.
- Do not claim real-time availability/prices.
- Do not call external travel booking APIs in v1.
- Do not expose discovery sessions to other users.
- Existing create trip flow must not regress.
- Existing AI generation flow must not regress.
- Existing i18n behavior must not regress.
- Keep code consistent with existing service patterns.
- Keep tests and docs updated.

Expected output:
The Create Trip page has a polished “Help me choose” AI discovery mode in addition to the existing known-destination form.
Users can enter a natural-language prompt, use quick chips, press Surprise Me, refine bad suggestions, and create a trip from a selected suggestion.
Trip Service orchestrates discovery using user preferences, previous trips, workspace policy, language, and AI Planning Service.
AI Planning Service exposes `/suggest-destinations` with prompt/surprise/refine modes and deterministic mock behavior.
Suggestions include destination cards with match score, budget estimate, why it fits, downsides, preview, and creation prompt.
No trip is created until the user confirms a suggestion.
Created trips store discovery metadata and can optionally start itinerary generation.
Docs, tests, and smoke tests are updated.

# 6. Add Multi-Destination & Multi-Modal Travel Planning v1: support trips with multiple stops, transfer legs between destinations, transport modes such as car/train/bus/flight/boat/bike/hiking, route builder UI, AI generation for transfer days, budget/route estimates, and map display

You are a senior full-stack engineer and product-minded AI architect. Continue building the web-based AI travel planning application.

Your task:
Implement Multi-Destination & Multi-Modal Travel Planning v1: support trips with multiple stops, transfer legs between destinations, transport modes such as car/train/bus/flight/boat/bike/hiking, route builder UI, AI generation for transfer days, budget/route estimates, and map display.

Context:
We already have a microservices-based AI travel planning app.

Existing services:

- Auth Service:
  - Go microservice
  - issues JWT access tokens and refresh tokens
- User/Profile Service:
  - Go microservice
  - stores user profile/preferences
  - stores preferredLanguage
  - owns workspace membership and workspace roles
- Trip Service:
  - Go microservice
  - owns trips, workspace trips, trip creation, itinerary generation jobs, AI trip discovery, AI template adaptation jobs, AI repair jobs, budgets, workspace budgets, cost analytics, cost splitting, approval workflow, workspace policies, approval risk scoring, comments, activity, version history, templates, conflict detection, notifications integration, and permissions
  - supports personal trips and workspace trips
  - calls AI Planning Service for generation/regeneration/optimization/adaptation/repair/discovery
- Worker Service:
  - Go microservice
  - processes RabbitMQ-backed jobs
- Notification Service:
  - Go microservice
  - owns in-app/email/web-push/SSE notifications and notification preferences
- External Integrations Service:
  - Go microservice
  - owns places, routes, weather, calendar, exchange rates, prices, availability provider adapters, quota/rate limits
- AI Planning Service:
  - Python FastAPI service
  - supports itinerary generation, partial regeneration, budget optimization, template adaptation, policy-aware repair, trip discovery, destination context/RAG, validation/repair, multilingual output, and Ollama/mock modes
- Web App:
  - Next.js app under apps/web
  - supports auth, trips, workspace switcher, workspace pages, trip discovery, templates, AI template adaptation, budgets, workspace budgets, cost analytics, cost splitting, approval workflow, workspace policies, approval risk scoring, AI repair proposals, availability cards, exports, offline mode, PWA install, notifications, internationalization, etc.
- Infra:
  - Postgres
  - RabbitMQ
  - Prometheus/Grafana

Current behavior:

- Trips are mostly modeled as one destination/city.
- Itinerary days contain activities inside one city.
- Route/walking estimates mostly assume walking inside a single destination.
- Create trip flow assumes the user provides one destination.
- AI Trip Discovery can suggest destinations but should now be able to suggest route-style trips too.

Problem:
Real travel is often multi-destination and multi-modal:

- Bratislava → Vienna → Salzburg → Hallstatt
- Barcelona → Valencia → Madrid
- Paris → Brussels → Amsterdam
- Tokyo → Kyoto → Osaka
- Road trips with several towns
- Train/backpacking routes
- Camping/hiking trips
- Island hopping by ferry/boat

The app needs to support:

- multiple stops/towns/cities in one trip
- transfer days between stops
- transport modes beyond walking
- car/train/bus/flight/boat/bike/hiking/public transport
- camping/hiking/adventure trip styles
- route-level budget and timing estimates

Goal:
Add Multi-Destination & Multi-Modal Travel Planning v1:

- A trip can have one or more route stops.
- Backward compatibility: old/single-city trips still work.
- Multi-destination trips include transfer legs between stops.
- Users can build/reorder route stops in the create trip page.
- Users can specify transport preferences and trip style.
- AI generation understands route stops and transfer days.
- Itinerary can contain transfer items/legs.
- Budget summary includes transfer costs.
- Map view shows route stops and transfer lines.
- Route validation warns about unrealistic routes.
- AI Trip Discovery can suggest multi-city routes, not only single destinations.

Do NOT add:

- real ticket booking
- train ticket purchase
- live flight search/prices
- hotel booking
- car rental checkout
- boat rental checkout
- camping permit booking
- advanced GPS hiking route generation
- turn-by-turn navigation
- offline navigation
- complex route optimization across countries
- legal/visa guarantees
- new backend service
- Kubernetes

For v1:

- Support structured route stops and transfer legs.
- Support planned transport modes.
- Use mock/estimated transfer durations and costs if real route provider does not support mode.
- Keep real provider integration optional/fail-open.
- Use AI to plan realistic transfer days.
- Keep user in control.
- Existing trips must not break.
- Single-destination trips should be represented as route with one stop internally if practical.
- Multi-destination trips should work with budgets, exports, map, approval/risk/policy as much as practical.

Important codebase consistency requirement:
Before implementing, inspect existing services and follow the same patterns exactly:

- services/trip-service
- services/ai-planning-service
- services/external-integrations-service
- services/worker-service
- services/user-service
- services/notification-service
- apps/web

Do not invent a different architecture if the repository already has conventions.

Match existing patterns for:

- migrations
- sqlc
- pgxpool
- Go modules
- Uber Fx
- Zap logging
- config/env
- HTTP handlers
- middleware
- JWT/trip/workspace permission checks
- response/error helpers
- itinerary schema validation
- version history
- activity events
- budget summary
- map view
- route estimate client
- AI request building
- FastAPI schemas/routes
- Ollama/mock mode
- i18n
- frontend API clients/hooks
- TanStack Query
- forms
- UI components
- tests
- smoke scripts
- docs

Part 1: Core domain model

1. Add route model to trips.

A trip should support:

{
"route": {
"origin": {
"name": "Bratislava",
"country": "Slovakia",
"coordinates": {
"lat": 48.1486,
"lng": 17.1077
}
},
"returnToOrigin": false,
"stops": [
{
"id": "stop_1",
"destination": "Vienna",
"city": "Vienna",
"country": "Austria",
"arrivalDate": "2026-09-10",
"departureDate": "2026-09-12",
"nights": 2,
"coordinates": {
"lat": 48.2082,
"lng": 16.3738
},
"accommodationHint": "hotel",
"notes": null
},
{
"id": "stop_2",
"destination": "Salzburg",
"city": "Salzburg",
"country": "Austria",
"arrivalDate": "2026-09-12",
"departureDate": "2026-09-14",
"nights": 2,
"coordinates": {
"lat": 47.8095,
"lng": 13.0550
},
"accommodationHint": "guesthouse",
"notes": null
}
],
"legs": [
{
"id": "leg_1",
"fromStopId": "origin",
"toStopId": "stop_1",
"fromName": "Bratislava",
"toName": "Vienna",
"mode": "train",
"departureDate": "2026-09-10",
"estimatedDurationMinutes": 70,
"estimatedDistanceKm": 80,
"estimatedCost": {
"amount": 18,
"currency": "EUR",
"category": "transport",
"confidence": "medium",
"source": "ai"
},
"notes": "Direct regional train recommended.",
"providerMetadata": null
}
],
"preferences": {
"preferredModes": ["train", "public_transport"],
"avoidModes": ["flight"],
"carAvailable": false,
"maxTransferHoursPerDay": 4,
"tripStyles": ["train_trip", "city_break"]
}
}
}

2. Backward compatibility.

Existing single-city trips:

- continue working with existing destination field.
- route can be null.
- OR route is auto-derived as one stop.
  Recommended:
- Add nullable route_json JSONB column to trips.
- Keep existing destination field.
- New multi-destination trips set route_json.
- Existing APIs include route if present.
- UI treats route null as single-destination.

3. Migration.

Trip Service migration:
ALTER TABLE trips ADD COLUMN route_json JSONB NULL;

Optional:
ALTER TABLE trips ADD COLUMN trip_type TEXT NOT NULL DEFAULT 'single_destination';

Allowed trip_type:

- single_destination
- multi_destination

If adding trip_type:

- existing trips default single_destination.
- multi-stop trips set multi_destination.

4. Route validation.

Validate:

- stops length 1–20.
- stop destination required.
- arrivalDate/departureDate valid if present.
- departureDate >= arrivalDate.
- nights >= 0.
- legs connect valid stop IDs or origin.
- supported transport mode.
- maxTransferHoursPerDay reasonable, 1–24.
- coordinates optional but if present lat/lng valid.
- tripStyles supported.
- route dates should not contradict trip start/end/duration if those exist.

5. Transport mode enum.

Supported v1 modes:

- walk
- car
- rental_car
- train
- bus
- flight
- boat
- ferry
- bike
- public_transport
- hiking
- other

6. Trip style enum.

Supported v1 trip styles:

- city_break
- road_trip
- train_trip
- backpacking
- camping
- hiking
- island_hopping
- nature
- beach
- food
- culture
- adventure
- family
- romantic
- low_budget
- luxury
- hidden_gem

7. Accommodation hint enum.

Supported:

- hotel
- hostel
- apartment
- guesthouse
- campsite
- cabin
- campervan
- home
- other
- unknown

Part 2: Itinerary schema updates

8. Support transfer itinerary items.

Add or extend itinerary item type:

- transfer

Transfer item shape:

{
"type": "transfer",
"name": "Train from Vienna to Salzburg",
"description": "Travel from Vienna to Salzburg by train, then check in and take a relaxed evening walk.",
"startTime": "09:30",
"endTime": "12:00",
"transfer": {
"legId": "leg_2",
"from": "Vienna",
"to": "Salzburg",
"mode": "train",
"estimatedDurationMinutes": 150,
"estimatedDistanceKm": 295,
"estimatedCost": {
"amount": 35,
"currency": "EUR",
"category": "transport",
"confidence": "medium",
"source": "ai"
},
"bookingRequired": false,
"notes": "Check train times before travel."
},
"estimatedCost": {
"amount": 35,
"currency": "EUR",
"category": "transport",
"confidence": "medium",
"source": "ai"
}
}

9. Day location.

Each itinerary day should optionally include:

- primaryStopId
- locationName
- transferDay boolean

Example:

{
"dayNumber": 3,
"date": "2026-09-12",
"title": "Transfer to Salzburg",
"primaryStopId": "stop_2",
"locationName": "Salzburg",
"transferDay": true,
"items": [...]
}

10. Existing itinerary validation.

Update validators to accept:

- transfer item type
- transfer object
- transport modes
- estimatedCost category transport
- day.primaryStopId/locationName/transferDay if present

Do not break old itinerary items.

Part 3: Trip Service API

11. Update trip create/update DTOs.

Trip creation should accept either:

- destination for single-city trip
- route for multi-destination trip

Request example:

{
"tripType": "multi_destination",
"title": "Austria train route",
"destination": "Austria",
"startDate": "2026-09-10",
"days": 5,
"budget": {
"amount": 900,
"currency": "EUR"
},
"travelers": 2,
"route": {...}
}

Rules:

- If tripType=single_destination, destination required.
- If tripType=multi_destination, route.stops required.
- destination can be derived as “Austria route” or first/primary stop.
- trips list should display route summary for multi-destination trips.

12. Add route endpoints.

Add endpoints:

GET /trips/{tripId}/route
PUT /trips/{tripId}/route

PUT request:
{
"expectedItineraryRevision": 12,
"route": {...}
}

Permissions:

- owner/editor can update route.
- viewer read-only.
- public share read-only if route included in public trip.

Behavior:

- Updating route is material if itinerary exists.
- Should increment itineraryRevision only if route is stored as part of itinerary? Recommended:
  - route update should increment routeRevision or update trip metadata but also mark itinerary potentially stale.
  - Simpler v1: route update increments itineraryRevision only if itinerary depends on route.
- If approval pending/approved, reset to draft if route materially changes.
- Create activity event route_updated.
- Expire stale repair proposals if needed.

13. Public share behavior.

Public trip share may include sanitized route:

- origin name
- stops
- legs
- transport modes
- durations/costs if itinerary exposes them

Do not expose:

- private notes
- internal provider metadata
- user IDs
- workspace policy metadata

14. Version history.

Itinerary versions should include route snapshot if route affects itinerary.
If existing version only stores itinerary JSON, add metadata routeSnapshot if useful.

Part 4: Transfer estimates / External Integrations

15. Extend route estimate endpoint.

Existing:
POST /routes/estimate

Extend request to support modes:

- car
- train
- bus
- flight
- boat/ferry
- bike
- hiking
- public_transport

Request:
{
"from": {
"name": "Vienna",
"lat": 48.2082,
"lng": 16.3738
},
"to": {
"name": "Salzburg",
"lat": 47.8095,
"lng": 13.0550
},
"mode": "train",
"date": "2026-09-12",
"currency": "EUR"
}

Response:
{
"mode": "train",
"estimatedDistanceKm": 295,
"estimatedDurationMinutes": 150,
"estimatedCost": {
"amount": 35,
"currency": "EUR",
"category": "transport",
"confidence": "low",
"source": "mock"
},
"provider": "mock",
"fallbackUsed": true,
"warnings": [
"This is an estimate, not a live schedule."
]
}

16. Mock estimator.

Implement deterministic estimates:

- walk: distance / 5 km/h
- bike: distance / 15 km/h
- hiking: distance / 3.5 km/h plus terrain warning
- car/rental_car: distance / 80 km/h plus 20 min buffer
- bus: distance / 60 km/h plus 30 min buffer
- train: distance / 100 km/h plus 20 min buffer
- flight: fixed airport overhead 180 min + flight distance / 700 km/h
- ferry/boat: distance / 35 km/h plus 30 min buffer
- public_transport: distance / 35 km/h plus 30 min buffer
- other: distance / 50 km/h

Cost estimates:

- walk: 0
- bike: low/0 if user-owned, else estimate
- hiking: 0
- car/rental_car: fuel estimate distance \* 0.18 EUR/km, rental_car add optional daily estimate if existing
- bus: distance \* 0.08 EUR/km
- train: distance \* 0.12 EUR/km
- flight: max(50, distance \* 0.15 EUR/km)
- ferry/boat: distance \* 0.20 EUR/km
- public_transport: distance \* 0.10 EUR/km
- boat rental should be an activity/accommodation-style estimate, not ordinary transfer booking in v1.

17. Real provider support.

If existing OpenRouteService supports only walking/driving/cycling:

- map car/rental_car to driving.
- bike to cycling.
- walk/hiking to walking if safe.
- train/bus/flight/ferry should use mock estimate in v1 with warning.
- Do not pretend live schedules exist.

18. Trip Service route estimation client.

Trip Service should be able to:

- estimate all route legs
- update route_json legs with estimates
- fail open with warnings if provider fails

Config:

- MULTI_DESTINATION_ENABLED=true
- ROUTE_LEG_ESTIMATION_ENABLED=true
- ROUTE_LEG_ESTIMATION_FAIL_OPEN=true
- ROUTE_LEG_MAX_STOPS=20
- ROUTE_LEG_TIMEOUT_SECONDS=8

Part 5: AI Planning Service updates

19. Update generation schemas.

Add route context to generation requests:

- route
- transport preferences
- trip styles
- max transfer time
- camping/hiking preferences

Request excerpt:
{
"route": {...},
"transportPreferences": {
"preferredModes": ["train"],
"avoidModes": ["flight"],
"carAvailable": false,
"maxTransferHoursPerDay": 4
},
"tripStyles": ["train_trip", "city_break"]
}

20. AI prompt rules.

Prompt should instruct:

- Plan across all route stops.
- Respect arrival/departure dates and nights per stop.
- Include transfer items on transfer days.
- Do not schedule dense sightseeing before/after long transfers.
- Use selected transport modes.
- Avoid disallowed transport modes.
- Add realistic rest after long travel.
- For camping trips, include campsite/accommodation-style notes but do not claim reservations.
- For hiking trips, include conservative day planning and safety notes, but do not generate technical GPS routes.
- For boat/ferry/island hopping, include transfer estimates as approximate and warn to verify schedules.
- Keep costs as estimates.
- Do not claim tickets/bookings are confirmed.
- Output in requested language.

21. Mock generation behavior.

For route with multiple stops:

- Generate days assigned to stops.
- Insert transfer item when day changes stop.
- Use route leg mode in transfer item.
- Produce deterministic simple itinerary.

Example:
Day 1: Arrival in Vienna
Day 2: Explore Vienna
Day 3: Transfer Vienna → Salzburg by train + Salzburg evening
Day 4: Explore Salzburg
Day 5: Return / relaxed final day

22. Template adaptation / repair / discovery.

Update AI Planning Service schemas/prompts:

- Template adaptation can adapt templates into route-style trips if route provided.
- Policy-aware repair can repair route-related issues.
- Trip discovery can suggest route suggestions, not only single destinations.

Part 6: AI Trip Discovery integration

23. Support route suggestions.

Extend destination suggestion response with optional route:

{
"suggestionType": "single_destination" | "route",
"destination": "Austria train route",
"route": {
"origin": {...},
"stops": [...],
"legs": [...],
"preferences": {...}
}
}

24. Discovery cards.

For route suggestions, cards should show:

- route title
- stops sequence
- transport mode
- estimated total route duration
- estimated transfer cost
- why it fits
- downsides
- “Use this route”

25. Create trip from route suggestion.

When user selects route suggestion:

- create multi_destination trip
- store route_json
- optionally auto-generate itinerary

Part 7: Budget integration

26. Transport cost category.

Ensure estimatedCost supports:

- category: transport

Transfer legs should contribute to:

- trip budget summary
- cost analytics
- workspace budget summary
- cost splitting if travelers > 1

27. Avoid double counting.

If a transfer leg has estimatedCost and itinerary transfer item also has estimatedCost:

- decide one source of truth.
  Recommended:
- Itinerary transfer item estimatedCost is included in budget summary.
- Route leg cost is used to prefill transfer item and route display.
- Avoid counting route leg separately if equivalent transfer item exists.

28. Accommodation/camping.

Camping as accommodation style:

- campsite accommodation cost may be stored in accommodation model or itinerary estimatedCost.
- Do not implement full campsite booking.
- Budget can include campsite cost as accommodation if user adds it.

Part 8: Policy/risk/repair integration

29. Workspace policy additions.

Extend workspace policy rules optionally:

- maxTransferHoursPerDay
- disallowedTransportModes
- preferredTransportModes already exists
- requireCarAvailableForRoadTrip optional
- maxTransportBudget optional

If too much for v1:

- add only:
  - maxTransferHoursPerDay
  - disallowedTransportModes

30. Policy evaluator.

Evaluate:

- transfer legs exceeding maxTransferHoursPerDay.
- disallowed transport modes used.
- flight used when disallowed.
- transport budget over threshold if implemented.

31. Approval risk scoring.

Add risk factors:

- too_many_stops_for_duration
- long_transfer_day
- disallowed_transport_mode
- route_estimate_missing
- high_transport_cost
- hiking_day_too_dense
- camping_accommodation_missing

32. AI repair.

Repair can suggest:

- remove a stop
- change transport mode
- add rest after transfer
- reduce route complexity
- replace flight with train if feasible
- split long transfer across days

Do not auto-apply.

Part 9: Web App route builder UI

33. Create trip page.

Update /trips/new.

Modes:

- Single destination
- Multi-destination route
- Help me choose

Multi-destination route builder:

- origin input
- stops list
- add stop
- remove stop
- reorder stops
- nights per stop
- arrival/departure date optional
- transport mode per leg
- route preferences

34. Components.

Create:

apps/web/components/routes/TripRouteBuilder.tsx
apps/web/components/routes/RouteStopCard.tsx
apps/web/components/routes/RouteLegCard.tsx
apps/web/components/routes/TransportModeSelector.tsx
apps/web/components/routes/TripStyleSelector.tsx
apps/web/components/routes/RouteSummaryCard.tsx
apps/web/components/routes/RouteValidationWarnings.tsx

35. Transport mode selector.

Show modes:

- Walking
- Car
- Rental car
- Train
- Bus
- Flight
- Ferry/boat
- Bike
- Hiking
- Public transport
- Other

Use icons if existing icon system supports it.

36. Trip style selector.

Chips:

- Road trip
- Train trip
- Backpacking
- Camping
- Hiking
- Island hopping
- Nature
- Beach
- Food
- Culture
- Adventure
- Family
- Romantic
- Low budget
- Hidden gem

37. Route validation UI.

Warn:

- “5 stops in 3 days may feel rushed.”
- “This transfer is longer than your max transfer time.”
- “Flight is selected but flights are in your avoid list.”
- “Camping selected, but no campsite/accommodation stop is configured.”
- “Hiking selected; routes are approximate and should be checked with local maps.”

38. Create trip dialog/submit.

When submitting:

- validate route.
- create trip with tripType=multi_destination.
- optionally auto-generate itinerary.

Part 10: Trip detail UI

39. Route overview panel.

Add route overview on trip detail:

- route sequence
- stops with dates/nights
- legs with mode/duration/cost
- warnings
- edit route button for editors

40. Itinerary UI.

Transfer items should render differently:

- transport icon
- from → to
- mode
- estimated duration
- estimated cost
- warning “Verify schedule before travel”

41. Map view.

Update map to show:

- markers for stops
- numbered stop order
- lines between stops
- existing activity markers if coordinates exist

If exact route geometry unavailable:

- draw straight lines with dashed/approx style.
- label as approximate.

42. Route editing.

Editors can update route.
If itinerary exists:

- show warning:
  “Changing the route may make the current itinerary outdated. Regenerate affected days after saving.”

V1 can update route without automatically rewriting itinerary.

Part 11: Frontend API/types/hooks

43. Types.

Create/update:

apps/web/types/route.ts

Types:

- TripRoute
- RouteStop
- RouteLeg
- TransportMode
- TripStyle
- AccommodationHint
- RouteValidationWarning

Update trip types to include:

- tripType
- route

44. API client.

Create/update:
apps/web/lib/api/trip-routes.ts

Functions:

- getTripRoute(tripId)
- updateTripRoute(tripId, input)

Update createTrip API to accept tripType/route.

45. Hooks.

Create:

- useTripRoute
- useUpdateTripRoute

Update:

- useCreateTrip
- useGenerateTrip
- useTrip
- useTripList if route summary displayed

Part 12: Exports/calendar/offline

46. PDF export.

Include:

- route overview
- stops
- transfer legs
- transport warnings

47. CSV export.

Include transfer items and transport mode/cost columns.

48. ICS export.

Transfer items should become calendar events if timed.
Title:
“Transfer: Vienna → Salzburg”
Description includes mode/duration/warnings.

49. Offline mode.

Cached trip should include route_json.
Offline editing route can be disabled in v1 unless existing offline mutation architecture supports it.

Part 13: Internationalization

50. Add translation keys.

Namespaces:

- routes
- transportModes
- tripStyles

Translate to:

- en
- es
- uk
- fr

Keys:

- Multi-destination route
- Add stop
- Remove stop
- Reorder
- Origin
- Stop
- Transfer
- Transport mode
- Road trip
- Train trip
- Camping
- Hiking
- Ferry/boat
- Estimated duration
- Estimated cost
- Verify schedules before travel
- Route may be unrealistic

Part 14: Backend tests

51. Trip Service tests.

Test:

- create single destination trip still works.
- create multi-destination trip with route works.
- invalid route rejected.
- unsupported transport mode rejected.
- get/update route permissions.
- viewer cannot update route.
- public share route sanitized.
- route update resets approval if needed.
- route update creates activity.
- generation request includes route.
- existing trip APIs do not break when route null.

52. Itinerary validation tests.

Test:

- transfer item accepted.
- invalid transfer mode rejected.
- transfer estimatedCost category transport accepted.
- old itinerary items still accepted.

53. Budget tests.

Test:

- transfer item costs included in budget.
- route leg cost not double-counted if matching transfer item exists.
- transport category appears in analytics.

54. Policy/risk tests.

Test:

- disallowed transport mode violation.
- max transfer hours violation.
- risk factor for long transfer.
- personal trip behavior unchanged.

55. External Integration tests.

Test:

- route estimate for car/train/bus/flight/ferry returns deterministic mock.
- unsupported mode rejected.
- ORS provider maps car/bike/walk where available.
- train/flight/ferry fallback to mock with warning.

56. AI Planning Service tests.

Test:

- mock generation with route produces transfer days.
- route prompt includes stops/legs/modes.
- no booking claims.
- multilingual output still works.
- old single-destination generation still works.

Part 15: Frontend tests

57. Route builder tests.

Test:

- add/remove/reorder stop.
- select transport mode.
- select trip style.
- route validation warnings.
- submit multi-destination create request.

58. Trip detail tests.

Test:

- route overview renders stops/legs.
- transfer item card renders mode/from/to/duration/cost.
- map receives route stops/lines.
- edit route button hidden for viewer.

59. Discovery tests.

Test:

- route suggestion card renders stop sequence.
- create trip from route suggestion sends route.

60. i18n tests.

Test route/transport/trip style labels in all four languages.

Part 16: Smoke tests

61. Update scripts/smoke-test.sh.

API smoke:

1. Login user.
2. Create multi-destination trip:
   - origin Bratislava
   - stops Vienna and Salzburg
   - train leg
3. Assert trip route exists.
4. Trigger itinerary generation in mock mode.
5. Assert itinerary has transfer item.
6. Assert budget summary includes transport cost.
7. Update route with car leg.
8. Assert activity created and route updated.
9. Create single-destination trip and assert old flow still works.
10. Request route estimate for train/bus/flight/ferry and assert response/warnings.

11. Update scripts/web-smoke-test.md.

Manual test:

1. Open /trips/new.
2. Select Multi-destination route.
3. Add origin and 3 stops.
4. Select train/car transport modes.
5. Select trip styles: train trip + hiking.
6. Create trip and auto-generate itinerary.
7. Confirm route overview appears.
8. Confirm transfer day appears.
9. Confirm map shows stop markers and route lines.
10. Confirm budget includes transfer cost.
11. Edit route and confirm warning about outdated itinerary.
12. Confirm single-destination create still works.

Part 17: Documentation

63. Update Trip Service README.

Document:

- route_json schema
- trip_type
- route endpoints
- transfer itinerary items
- budget behavior
- policy/risk integration
- public share sanitization
- limitations

64. Update External Integrations README.

Document:

- route estimate modes
- mock estimator rules
- provider fallback
- warnings for non-live schedules

65. Update AI Planning Service README.

Document:

- route context in generation requests
- transfer day behavior
- camping/hiking constraints
- no booking/live schedule claims

66. Update Web App README.

Document:

- multi-destination route builder
- transport mode selector
- trip style selector
- route overview/map
- limitations

67. Update root README.md.

Mention:

- Multi-Destination & Multi-Modal Travel Planning v1.

68. User-facing limitations.

Document:

- transport durations/costs are estimates.
- no live train/bus/flight/ferry schedules in v1.
- no booking or ticket purchase.
- hiking/camping suggestions require user verification.
- route lines on map may be approximate.
- changing route does not automatically rewrite the whole itinerary unless user regenerates.

Part 18: Security and quality requirements

- Existing single-destination trips must not break.
- Route update must enforce trip edit permissions.
- Public share must sanitize route metadata.
- Do not expose internal provider metadata.
- Do not claim live schedules/prices unless real provider explicitly supports them.
- Do not book transport/accommodation.
- Do not generate technical hiking navigation or safety guarantees.
- AI output must be validated before saving.
- Transport cost must avoid double counting.
- Approval/risk/policy behavior must remain consistent.
- Keep code consistent with existing service patterns.
- Keep tests and docs updated.

Expected output:
Trips can be single-destination or multi-destination.
Multi-destination trips store route stops, transfer legs, transport modes, and trip styles.
Create Trip page includes a route builder with stops, transport modes, and route validation warnings.
AI generation can create transfer days and multi-stop itineraries.
External Integrations can estimate transfer durations/costs for supported modes with mock/fallback behavior.
Trip detail shows route overview, transfer items, and route map.
Budget, policy, approval risk, exports, discovery, and public share support route data where practical.
Existing single-city trips continue working.
Docs, tests, and smoke tests are updated.
