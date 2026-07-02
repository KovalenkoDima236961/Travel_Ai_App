package main

import (
	"flag"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/worker-service/internal/app"
)

func main() {
	configPath := flag.String("config", "", "path to optional config file")
	flag.Parse()

	app.New(*configPath).Run()
}
