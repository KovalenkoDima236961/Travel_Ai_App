# Auth In-Memory Repository

> 55 nodes · cohesion 0.07

## Key Concepts

- **service_test.go** (36 connections) — `services/auth-service/internal/application/service/service_test.go`
- **mockRepo** (15 connections) — `services/user-service/internal/application/service/service_test.go`
- **authContext()** (11 connections) — `services/user-service/internal/application/service/service_test.go`
- **newTestService()** (11 connections) — `services/auth-service/internal/application/service/service_test.go`
- **memoryRepository** (10 connections) — `services/auth-service/internal/application/service/service_test.go`
- **testUserID()** (10 connections) — `services/user-service/internal/application/service/service_test.go`
- **assertInvalidInput()** (5 connections) — `services/user-service/internal/application/service/service_test.go`
- **TestUpdateProfileRejectsInvalidCurrency()** (5 connections) — `services/user-service/internal/application/service/service_test.go`
- **TestUpdateProfileRejectsTooLongDisplayName()** (5 connections) — `services/user-service/internal/application/service/service_test.go`
- **TestPatchPreferencesClearsMaxWalkingWhenExplicitlyNull()** (4 connections) — `services/user-service/internal/application/service/service_test.go`
- **TestPatchPreferencesMergesProvidedFieldsAndSanitizesArrays()** (4 connections) — `services/user-service/internal/application/service/service_test.go`
- **TestPatchPreferencesRejectsInvalidPace()** (4 connections) — `services/user-service/internal/application/service/service_test.go`
- **TestPatchPreferencesRejectsMaxWalkingOver50()** (4 connections) — `services/user-service/internal/application/service/service_test.go`
- **TestServiceRegisterSuccess()** (4 connections) — `services/auth-service/internal/application/service/service_test.go`
- **TestUpdateProfileUsesAuthenticatedUserID()** (4 connections) — `services/user-service/internal/application/service/service_test.go`
- **validProfileInput()** (4 connections) — `services/user-service/internal/application/service/service_test.go`
- **.tokenByRaw()** (3 connections) — `services/auth-service/internal/application/service/service_test.go`
- **assertStrings()** (3 connections) — `services/user-service/internal/application/service/service_test.go`
- **TestGetPreferencesCreatesDefaultWhenMissing()** (3 connections) — `services/user-service/internal/application/service/service_test.go`
- **TestGetProfileCreatesDefaultWhenMissing()** (3 connections) — `services/user-service/internal/application/service/service_test.go`
- **TestServiceRefreshInvalidStates()** (3 connections) — `services/auth-service/internal/application/service/service_test.go`
- **TestServiceRefreshRotatesToken()** (3 connections) — `services/auth-service/internal/application/service/service_test.go`
- **testHasher** (3 connections) — `services/auth-service/internal/application/service/service_test.go`
- **.expireRefreshToken()** (2 connections) — `services/auth-service/internal/application/service/service_test.go`
- **.GetUserByEmail()** (2 connections) — `services/auth-service/internal/application/service/service_test.go`
- *... and 30 more nodes in this community*

## Relationships

- No strong cross-community connections detected

## Source Files

- `services/auth-service/internal/application/service/service_test.go`
- `services/trip-service/internal/application/service/trip_test.go`
- `services/user-service/internal/application/service/service_test.go`

## Audit Trail

- EXTRACTED: 203 (100%)
- INFERRED: 0 (0%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*