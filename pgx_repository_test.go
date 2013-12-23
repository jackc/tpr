package main

import (
	"github.com/JackC/pgx"
	"github.com/kylelemons/go-gypsy/yaml"
	"testing"
)

var sharedPgxRepository *pgxRepository

func getFreshPgxRepository(t testing.TB) *pgxRepository {
	var err error

	if sharedPgxRepository == nil {
		var connectionParameters pgx.ConnectionParameters
		var yf *yaml.File

		configPath := "config.test.yml"
		if yf, err = yaml.ReadFile(configPath); err != nil {
			t.Fatalf("Unable to read %v as YAML: %v", configPath, err)
		}

		if connectionParameters, err = extractConnectionOptions(yf); err != nil {
			t.Fatalf("Unable to read connection parameters from %v: %v", configPath, err)
		}

		connectionPoolOptions := pgx.ConnectionPoolOptions{MaxConnections: 1, AfterConnect: afterConnect}
		sharedPgxRepository, err = NewPgxRepository(connectionParameters, connectionPoolOptions)
		if err != nil {
			t.Fatalf("NewPgxRepository failed: %v", err)
		}
	}

	err = sharedPgxRepository.empty()
	if err != nil {
		t.Fatalf("Unable to empty pgxRepository: %v", err)
	}

	return sharedPgxRepository
}

func TestPgxRepositoryUsers(t *testing.T) {
	repo = getFreshPgxRepository(t)
	testRepositoryUsers(t, repo)
}
