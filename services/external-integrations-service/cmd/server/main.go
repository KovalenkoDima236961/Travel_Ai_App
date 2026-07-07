package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/app"
)

var configPath = flag.String("config", "", "Use to specify path to config file")

func main() {
	flag.Parse()
	service, err := app.New(*configPath)
	if err != nil {
		fail(err)
	}
	if err := service.Run(); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "external-integrations-service failed: %v\n", err)
	os.Exit(1)
}
