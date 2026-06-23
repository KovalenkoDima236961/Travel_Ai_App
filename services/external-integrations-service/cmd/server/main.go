package main

import (
	"flag"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/app"
)

var configPath = flag.String("config", "", "Use to specify path to config file")

func main() {
	flag.Parse()
	service := app.New(*configPath)
	service.Run()
}
