// Package version exposes build metadata that is safe to publish from /version.
package version

import "os"

const ServiceName = "notification-service"

var (
	Version   = "dev"
	GitSHA    = "unknown"
	BuildTime = "unknown"
)

type Metadata struct {
	Service            string `json:"service"`
	Version            string `json:"version"`
	GitSHA             string `json:"gitSha"`
	BuildTime          string `json:"buildTime"`
	Environment        string `json:"environment"`
	APIContractVersion string `json:"apiContractVersion"`
}

func Info() Metadata {
	environment := os.Getenv("APP_ENV")
	if environment == "" {
		environment = "local"
	}
	return Metadata{Service: ServiceName, Version: Version, GitSHA: GitSHA, BuildTime: BuildTime, Environment: environment, APIContractVersion: Version}
}
