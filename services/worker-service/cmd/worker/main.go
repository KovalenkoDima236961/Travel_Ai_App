package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/worker-service/internal/app"
)

func main() {
	configPath := flag.String("config", "", "path to optional config file")
	flag.Parse()

	if err := run(*configPath); err != nil {
		fmt.Fprintf(os.Stderr, "worker-service failed: %v\n", err)
		os.Exit(1)
	}
}

func run(configPath string) error {
	worker, err := app.New(configPath)
	if err != nil {
		return err
	}
	return worker.Run()
}
