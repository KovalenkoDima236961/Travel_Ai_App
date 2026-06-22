package tls

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"

	"golang.org/x/crypto/acme/autocert"
)

type CertManager interface {
	HTTPHandler(fallback http.Handler) http.Handler
	TLSConfig() *tls.Config
}

type ManagerConfig struct {
	EnableTLS     bool     `yaml:"enable_tls" env:"HTTP_ENABLE_TLS" env-default:"false"`
	CertsStoreDir string   `yaml:"certs_store_dir" env:"HTTP_CERTS_STORE_DIR" env-default:"./certs" validate:"required"`
	HostWhitelist []string `yaml:"host_whitelist" env:"HTTP_TLS_HOST_WHITELIST" env-separator:"," env-default:"localhost,0.0.0.0"`
	AcceptTOS     bool     `yaml:"accept_tos" env:"TLS_ACCEPT_TOS" env-default:"false"`
	Email         string   `yaml:"email" env:"TLS_EMAIL" validate:"omitempty,email"`
}

func GetCertManager(cfg *ManagerConfig) (CertManager, error) {
	if err := os.MkdirAll(cfg.CertsStoreDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to init cache certs directory: %w", err)
	}

	m := &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(cfg.HostWhitelist...),
		// Cache certificates to avoid issues with rate limits (https://letsencrypt.org/docs/rate-limits)
		Cache: autocert.DirCache(cfg.CertsStoreDir),
	}

	if cfg.Email != "" {
		m.Email = cfg.Email
	}

	return m, nil
}