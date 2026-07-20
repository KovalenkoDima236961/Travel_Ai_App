# Playbook: add a travel data provider

Read [trusted travel data providers](../../ai/trusted-travel-data-providers.md) first. A provider may not be enabled until its licensing and limits are documented; unlicensed provenance is a policy failure, not a quality issue.

## Before writing code

1. Record the provider's license name, license URL, attribution requirement, and terms URL. If attribution is required, confirm where the app will display it.
2. Confirm the terms permit storing normalized facts. If they discourage retaining the original response, the adapter must set `LicenseInfo.AllowsRawPayload = false` — that flag overrides any environment setting.
3. Record the rate limits and pick the `rate_limit_category` bucket used by the existing provider quota management in External Integrations Service.
4. Confirm the data is facts (name, coordinates, category, hours), not expressive text. Copying descriptions requires a license that explicitly permits it.
5. Add the provider to the table in the strategy doc, including its trust level. An undocumented provider maps to `unknown` trust and cannot reach strong grounding.

## Implementation

6. Implement the adapter in `services/external-integrations-service/internal/providers/`, following the existing `places` provider pattern: config-driven selection, quota guard, cache, timeout, and mock fallback.
7. Satisfy `TravelKnowledgeProvider` (`services/trip-service/internal/knowledge/provider/provider.go`): `SearchPlaces`, `GetPlaceDetails`, `SupportsRefresh`, `ProviderName`, `LicenseInfo`.
8. Return a complete `LicenseInfo` from every record. `NormalizeProviderRecord` rejects a record whose license is missing when the run policy requires one.
9. Never put credentials in a `PlaceRecord`, including `RawPayload`. Build records from response bodies only. `scripts/ai/assert-no-real-providers.sh` fails the build if the knowledge module starts handling credentials.
10. Add the provider name to `trustLevelForProvider` in `services/trip-service/internal/knowledge/ingest.go` and to the selection switch in `services/worker-service/internal/knowledge/provider_runner.go` and `buildKnowledgeIngestor` in `services/trip-service/internal/app/di.go`.
11. Map the provider's categories in `providerCategoryAliases` (`normalize.go`). Every mapping must resolve to an existing `travel_places` category — a unit test enforces this.

## Testing and rollout

12. Add adapter unit tests with recorded fixtures. No test may make a network call; CI has no provider credentials and must stay that way.
13. Run `go test ./internal/knowledge/...` in trip-service and worker-service, plus `./scripts/ai/validate-knowledge-sources.sh`.
14. Ingest one destination with `--dry-run` first and inspect the scores before writing:
    ```bash
    go run ./cmd/knowledge-provider --job knowledge_provider_ingest_destination \
      --destination Rome --country-code IT --provider <name> --dry-run
    ```
15. Enable in staging with `KNOWLEDGE_PROVIDER_FALLBACK_TO_MOCK=true` and review the first ingestion in the Ops AI Knowledge Quality panel before enabling in production.
16. Leave `KNOWLEDGE_PROVIDER_STORE_RAW_PAYLOAD=false` in production unless a specific debugging need justifies it.

## Retention check

Observations persist in `travel_provider_place_observations`. Confirm the refresh window for the categories this provider supplies, and confirm raw payload retention is off or bounded before merge.
