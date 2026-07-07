package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/app"
)

var configPath = flag.String("config", "", "Use to specify path to config file")

func main() {
	flag.Parse()
	if err := run(*configPath); err != nil {
		fail(err)
	}
}

func run(configPath string) error {
	user, err := app.New(configPath)
	if err != nil {
		return err
	}
	return user.Run()
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "user-service failed: %v\n", err)
	os.Exit(1)
}
