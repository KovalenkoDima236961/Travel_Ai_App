package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/app"
)

var configPath = flag.String("config", "", "Use to specify path to config file")

func main() {
	flag.Parse()
	auth, err := app.New(*configPath)
	if err != nil {
		fail(err)
	}
	if err := auth.Run(); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "auth-service failed: %v\n", err)
	os.Exit(1)
}
