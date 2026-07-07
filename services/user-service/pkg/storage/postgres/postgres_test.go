package postgres

import (
	"net/url"
	"testing"
)

func TestPostgresURLEncodesCredentialsAndPoolParams(t *testing.T) {
	raw := postgresURL(Config{
		Database: "user_service",
		Username: "travel user",
		Password: "p@ss:word/with?chars",
		Host:     "localhost",
		Port:     5432,
		MinConns: 2,
		MaxConns: 10,
	}, true)

	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}

	if parsed.Scheme != "postgres" {
		t.Fatalf("unexpected scheme %q", parsed.Scheme)
	}
	if parsed.User.Username() != "travel user" {
		t.Fatalf("unexpected username %q", parsed.User.Username())
	}
	password, ok := parsed.User.Password()
	if !ok || password != "p@ss:word/with?chars" {
		t.Fatalf("unexpected password %q", password)
	}
	if parsed.Query().Get("pool_max_conns") != "10" {
		t.Fatalf("missing max pool param in %q", raw)
	}
	if parsed.Query().Get("pool_min_conns") != "2" {
		t.Fatalf("missing min pool param in %q", raw)
	}
	if parsed.Query().Get("sslmode") != "disable" {
		t.Fatalf("missing sslmode in %q", raw)
	}
}

func TestPostgresURLCanOmitPoolParamsForMigrations(t *testing.T) {
	raw := postgresURL(Config{
		Database: "user_service",
		Username: "postgres",
		Password: "postgres",
		Host:     "localhost",
		Port:     5432,
		MinConns: 2,
		MaxConns: 10,
	}, false)

	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	if parsed.Query().Get("pool_max_conns") != "" || parsed.Query().Get("pool_min_conns") != "" {
		t.Fatalf("migration url should omit pool params: %q", raw)
	}
}
