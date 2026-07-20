// Command knowledge-provider runs provider-backed knowledge ingestion jobs.
// It mirrors cmd/knowledge-ingest, which ingests the curated corpus, and
// defaults to the deterministic mock provider so it is safe to run in CI.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	tripconfig "github.com/KovalenkoDima236961/Travel_Ai_App/internal/config"
	tripknowledge "github.com/KovalenkoDima236961/Travel_Ai_App/internal/knowledge"
	storage "github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/storage/postgres"
	workerknowledge "github.com/KovalenkoDima236961/Travel_Ai_App/services/worker-service/internal/knowledge"
)

func main() {
	jobType := flag.String("job", tripknowledge.JobIngestDestination, "job type: "+
		strings.Join([]string{
			tripknowledge.JobIngestDestination,
			tripknowledge.JobRefreshStalePlaces,
			tripknowledge.JobMatchObservations,
			tripknowledge.JobQualityScoreRecompute,
			tripknowledge.JobDuplicateDetection,
			tripknowledge.JobReindexAfterMerge,
		}, ", "))
	destination := flag.String("destination", "", "destination canonical name")
	destinationID := flag.String("destination-id", "", "destination UUID (optional)")
	countryCode := flag.String("country-code", "", "ISO alpha-2 country code")
	categories := flag.String("categories", "", "comma-separated category filter")
	providerName := flag.String("provider", "", "override KNOWLEDGE_PROVIDER for this run")
	limit := flag.Int("limit", 0, "maximum provider results")
	batchSize := flag.Int("batch-size", 0, "refresh batch size")
	dryRun := flag.Bool("dry-run", false, "score and report without writing")
	summary := flag.Bool("summary", false, "print the knowledge quality summary and exit")
	flag.Parse()

	cfg, err := tripconfig.Load("")
	if err != nil {
		fail(fmt.Errorf("load trip configuration: %w", err))
	}
	db, err := storage.New(context.Background(), cfg.Postgres)
	if err != nil {
		fail(fmt.Errorf("open knowledge store: %w", err))
	}
	defer db.Close()

	store := tripknowledge.NewStore(db)
	thresholds := tripknowledge.DefaultThresholds()

	if *summary {
		result, summaryErr := store.QualitySummary(context.Background(), thresholds.StrongMinQuality, thresholds.StaleAfterDays)
		if summaryErr != nil {
			fail(summaryErr)
		}
		writeJSON(result)
		return
	}

	providerConfig := workerknowledge.ProviderConfigFromEnv()
	if strings.TrimSpace(*providerName) != "" {
		providerConfig.Provider = *providerName
	}

	runner, err := workerknowledge.NewProviderRunner(store, providerConfig)
	if err != nil {
		fail(err)
	}

	result, err := runner.Run(context.Background(), workerknowledge.ProviderRequest{
		JobType:         *jobType,
		DestinationID:   *destinationID,
		DestinationName: *destination,
		CountryCode:     *countryCode,
		Categories:      splitCategories(*categories),
		Provider:        *providerName,
		Limit:           *limit,
		BatchSize:       *batchSize,
		DryRun:          *dryRun,
	})
	if err != nil {
		fail(err)
	}
	writeJSON(result)
}

func splitCategories(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	categories := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			categories = append(categories, trimmed)
		}
	}
	return categories
}

func writeJSON(value any) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(value)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
