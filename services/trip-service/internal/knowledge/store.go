package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	storage "github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/storage/postgres"
)

// Store persists normalized grounding records. It deliberately accepts only
// CuratedDataset, which has already passed private-content and provenance
// validation at the ingestion boundary.
type Store struct{ db *storage.DB }

func NewStore(db *storage.DB) *Store { return &Store{db: db} }

type IngestionResult struct {
	SourcesUpserted      int `json:"sourcesUpserted"`
	DestinationsUpserted int `json:"destinationsUpserted"`
	PlacesUpserted       int `json:"placesUpserted"`
	DocumentsUpserted    int `json:"documentsUpserted"`
	ChunksUpserted       int `json:"chunksUpserted"`
}

type Status struct {
	Sources      int `json:"sources"`
	Destinations int `json:"destinations"`
	Places       int `json:"places"`
	Documents    int `json:"documents"`
	Chunks       int `json:"chunks"`
}

func (s *Store) UpsertCurated(ctx context.Context, dataset *CuratedDataset, dryRun bool) (IngestionResult, error) {
	if s == nil || s.db == nil {
		return IngestionResult{}, fmt.Errorf("knowledge store is required")
	}
	if dataset == nil {
		return IngestionResult{}, fmt.Errorf("curated dataset is required")
	}
	result := IngestionResult{}
	if dryRun {
		result.SourcesUpserted = len(dataset.Sources)
		result.DestinationsUpserted = len(dataset.Destinations)
		for _, destination := range dataset.Destinations {
			result.PlacesUpserted += len(destination.Places)
			if _, ok := dataset.Documents[NormalizeName(destination.CanonicalName)]; ok {
				result.DocumentsUpserted++
			}
		}
		return result, nil
	}

	sourceIDs := make(map[string]uuid.UUID, len(dataset.Sources))
	for key, source := range dataset.Sources {
		id, err := s.upsertSource(ctx, source)
		if err != nil {
			return IngestionResult{}, err
		}
		sourceIDs[key] = id
		result.SourcesUpserted++
	}
	for _, destination := range dataset.Destinations {
		destinationID, err := s.upsertDestination(ctx, destination, sourceIDs[SourceTypeManualCurated])
		if err != nil {
			return IngestionResult{}, err
		}
		result.DestinationsUpserted++
		for _, place := range destination.Places {
			if err := s.upsertPlace(ctx, destinationID, place, sourceIDs[place.SourceKey]); err != nil {
				return IngestionResult{}, err
			}
			result.PlacesUpserted++
		}
		document, ok := dataset.Documents[NormalizeName(destination.CanonicalName)]
		if !ok {
			continue
		}
		documentID, err := s.upsertDocument(ctx, destinationID, document, sourceIDs[document.SourceKey])
		if err != nil {
			return IngestionResult{}, err
		}
		result.DocumentsUpserted++
		chunks := chunkDocument(document.Content, 900)
		for index, content := range chunks {
			if err := s.upsertChunk(ctx, documentID, destinationID, index, content); err != nil {
				return IngestionResult{}, err
			}
			result.ChunksUpserted++
		}
	}
	return result, nil
}

func (s *Store) Status(ctx context.Context) (Status, error) {
	if s == nil || s.db == nil {
		return Status{}, fmt.Errorf("knowledge store is required")
	}
	var status Status
	query := `SELECT
  (SELECT count(*) FROM travel_knowledge_sources),
  (SELECT count(*) FROM travel_destinations),
  (SELECT count(*) FROM travel_places),
  (SELECT count(*) FROM travel_knowledge_documents),
  (SELECT count(*) FROM travel_knowledge_chunks)`
	if err := s.db.QueryRow(ctx, query).Scan(&status.Sources, &status.Destinations, &status.Places, &status.Documents, &status.Chunks); err != nil {
		return Status{}, fmt.Errorf("read knowledge status: %w", err)
	}
	return status, nil
}

func (s *Store) upsertSource(ctx context.Context, source Source) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.db.QueryRow(ctx, `INSERT INTO travel_knowledge_sources
  (source_key, source_type, display_name, license_name, license_url, attribution, trust_level, enabled)
  VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
  ON CONFLICT (source_key) DO UPDATE SET source_type=EXCLUDED.source_type, display_name=EXCLUDED.display_name,
    license_name=EXCLUDED.license_name, license_url=EXCLUDED.license_url, attribution=EXCLUDED.attribution,
    trust_level=EXCLUDED.trust_level, enabled=EXCLUDED.enabled, updated_at=NOW()
  RETURNING id`, source.SourceKey, source.SourceType, source.DisplayName, nullText(source.LicenseName), nullText(source.LicenseURL), nullText(source.Attribution), source.TrustLevel, source.Enabled).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("upsert knowledge source %q: %w", source.SourceKey, err)
	}
	return id, nil
}

func (s *Store) upsertDestination(ctx context.Context, destination DestinationKnowledge, sourceID uuid.UUID) (uuid.UUID, error) {
	encoded, err := marshalJSON(destination.Aliases, destination.Tags)
	if err != nil {
		return uuid.Nil, err
	}
	aliases, tags := encoded[0], encoded[1]
	var id uuid.UUID
	err = s.db.QueryRow(ctx, `INSERT INTO travel_destinations
  (canonical_name,country_code,country_name,region_name,latitude,longitude,aliases,tags,source_id,confidence,last_verified_at)
  VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,0.95,NOW())
  ON CONFLICT (canonical_name,country_code) DO UPDATE SET country_name=EXCLUDED.country_name, region_name=EXCLUDED.region_name,
    latitude=EXCLUDED.latitude, longitude=EXCLUDED.longitude, aliases=EXCLUDED.aliases, tags=EXCLUDED.tags,
    source_id=EXCLUDED.source_id, confidence=EXCLUDED.confidence, last_verified_at=EXCLUDED.last_verified_at, updated_at=NOW()
  RETURNING id`, destination.CanonicalName, destination.CountryCode, destination.CountryName, nullText(destination.RegionName), destination.Latitude, destination.Longitude, aliases, tags, sourceID).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("upsert destination %q: %w", destination.CanonicalName, err)
	}
	return id, nil
}

func (s *Store) upsertPlace(ctx context.Context, destinationID uuid.UUID, place PlaceKnowledge, sourceID uuid.UUID) error {
	encoded, err := marshalJSON(place.Aliases, place.Tags, place.AvoidIf, place.BestTimeOfDay)
	if err != nil {
		return err
	}
	aliases, tags, avoid, best := encoded[0], encoded[1], encoded[2], encoded[3]
	_, err = s.db.Exec(ctx, `INSERT INTO travel_places
  (destination_id,canonical_name,category,subcategory,latitude,longitude,address,neighborhood,aliases,tags,typical_duration_minutes,price_level,source_id,source_url,license_name,confidence,family_friendly,rain_friendly,outdoor,avoid_if,best_time_of_day,last_verified_at,status)
  VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,NOW(),'active')
  ON CONFLICT (destination_id,canonical_name) DO UPDATE SET category=EXCLUDED.category, subcategory=EXCLUDED.subcategory,
    latitude=EXCLUDED.latitude, longitude=EXCLUDED.longitude, address=EXCLUDED.address, neighborhood=EXCLUDED.neighborhood,
    aliases=EXCLUDED.aliases, tags=EXCLUDED.tags, typical_duration_minutes=EXCLUDED.typical_duration_minutes,
    price_level=EXCLUDED.price_level, source_id=EXCLUDED.source_id, source_url=EXCLUDED.source_url, license_name=EXCLUDED.license_name,
    confidence=EXCLUDED.confidence, family_friendly=EXCLUDED.family_friendly, rain_friendly=EXCLUDED.rain_friendly, outdoor=EXCLUDED.outdoor,
    avoid_if=EXCLUDED.avoid_if, best_time_of_day=EXCLUDED.best_time_of_day, last_verified_at=EXCLUDED.last_verified_at, status='active', updated_at=NOW()`,
		destinationID, place.Name, place.Category, nullText(place.Subcategory), place.Latitude, place.Longitude, nullText(place.Address), nullText(place.Neighborhood), aliases, tags, place.TypicalDurationMinutes, nullText(place.PriceLevel), sourceID, nullText(place.SourceURL), nullText(place.LicenseName), place.Confidence, place.FamilyFriendly, place.RainFriendly, place.Outdoor, avoid, best)
	if err != nil {
		return fmt.Errorf("upsert place %q: %w", place.Name, err)
	}
	return nil
}

func (s *Store) upsertDocument(ctx context.Context, destinationID uuid.UUID, document KnowledgeDocument, sourceID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.db.QueryRow(ctx, `INSERT INTO travel_knowledge_documents
  (source_id,destination_id,title,content,content_type,language,checksum,confidence,status)
  VALUES ($1,$2,$3,$4,$5,$6,$7,0.95,'active')
  ON CONFLICT (destination_id,title,source_id) DO UPDATE SET content=EXCLUDED.content, content_type=EXCLUDED.content_type,
    language=EXCLUDED.language, checksum=EXCLUDED.checksum, confidence=EXCLUDED.confidence, status='active', updated_at=NOW()
  RETURNING id`, sourceID, destinationID, document.Title, document.Content, document.ContentType, document.Language, document.Checksum).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("upsert document %q: %w", document.Title, err)
	}
	return id, nil
}

func (s *Store) upsertChunk(ctx context.Context, documentID, destinationID uuid.UUID, index int, content string) error {
	_, err := s.db.Exec(ctx, `INSERT INTO travel_knowledge_chunks (document_id,destination_id,chunk_index,content,checksum)
  VALUES ($1,$2,$3,$4,$5)
  ON CONFLICT (document_id,chunk_index) DO UPDATE SET content=EXCLUDED.content, checksum=EXCLUDED.checksum, embedding_id=NULL, updated_at=NOW()`, documentID, destinationID, index, content, Checksum(content))
	if err != nil {
		return fmt.Errorf("upsert document chunk %d: %w", index, err)
	}
	return nil
}

func marshalJSON(values ...[]string) ([][]byte, error) {
	result := make([][]byte, 0, len(values))
	for _, value := range values {
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("marshal knowledge json: %w", err)
		}
		result = append(result, encoded)
	}
	return result, nil
}

func nullText(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return strings.TrimSpace(value)
}

func chunkDocument(content string, maxChars int) []string {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil
	}
	if len(content) <= maxChars {
		return []string{content}
	}
	var chunks []string
	for len(content) > maxChars {
		boundary := strings.LastIndex(content[:maxChars], " ")
		if boundary < maxChars/2 {
			boundary = maxChars
		}
		chunks = append(chunks, strings.TrimSpace(content[:boundary]))
		content = strings.TrimSpace(content[boundary:])
	}
	if content != "" {
		chunks = append(chunks, content)
	}
	return chunks
}
