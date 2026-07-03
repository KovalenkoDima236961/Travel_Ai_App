package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/pkg/storage/postgres"
)

var configPath = flag.String("config", "", "Use to specify path to config file")

func main() {
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fail(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := postgres.New(ctx, cfg.Postgres)
	if err != nil {
		fail(err)
	}
	db.Close()
	fmt.Println("auth-service migrations applied")
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "migration failed: %v\n", err)
	os.Exit(1)
}
