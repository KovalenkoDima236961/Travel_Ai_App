package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	tripconfig "github.com/KovalenkoDima236961/Travel_Ai_App/internal/config"
	tripknowledge "github.com/KovalenkoDima236961/Travel_Ai_App/internal/knowledge"
	storage "github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/storage/postgres"
	workerknowledge "github.com/KovalenkoDima236961/Travel_Ai_App/services/worker-service/internal/knowledge"
)

func main() {
	dataDir := flag.String("data-dir", "", "path to data/travel-knowledge")
	destination := flag.String("destination", "", "optional canonical destination filter")
	dryRun := flag.Bool("dry-run", false, "validate and count without writing")
	status := flag.Bool("status", false, "print persisted knowledge counts")
	reindex := flag.Bool("reindex", false, "mark this ingestion for separate embedding reindex work")
	flag.Parse()
	if *dataDir == "" && !*status {
		fmt.Fprintln(os.Stderr, "--data-dir is required unless --status is used")
		os.Exit(2)
	}
	if *dryRun {
		dataset, err := tripknowledge.LoadCurated(*dataDir, *destination)
		if err != nil {
			fail(err)
		}
		result := tripknowledge.IngestionResult{
			SourcesUpserted:      len(dataset.Sources),
			DestinationsUpserted: len(dataset.Destinations),
		}
		for _, destination := range dataset.Destinations {
			result.PlacesUpserted += len(destination.Places)
			if _, ok := dataset.Documents[tripknowledge.NormalizeName(destination.CanonicalName)]; ok {
				result.DocumentsUpserted++
			}
		}
		writeJSON(struct {
			tripknowledge.IngestionResult
			DryRun bool `json:"dryRun"`
		}{IngestionResult: result, DryRun: true})
		return
	}

	cfg, err := tripconfig.Load("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "load trip configuration: %v\n", err)
		os.Exit(1)
	}
	db, err := storage.New(context.Background(), cfg.Postgres)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open knowledge store: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()
	store := tripknowledge.NewStore(db)
	if *status {
		result, err := store.Status(context.Background())
		if err != nil {
			fail(err)
		}
		writeJSON(result)
		return
	}
	result, err := workerknowledge.NewRunner(store).Run(context.Background(), workerknowledge.Request{
		DataDir: *dataDir, Destination: *destination, DryRun: *dryRun, Reindex: *reindex,
	})
	if err != nil {
		fail(err)
	}
	writeJSON(struct {
		tripknowledge.IngestionResult
		ReindexRequested bool `json:"reindexRequested"`
	}{IngestionResult: result, ReindexRequested: *reindex})
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
