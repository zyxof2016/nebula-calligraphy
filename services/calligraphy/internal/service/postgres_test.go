package service

import (
	"strings"
	"testing"
)

func TestPostgresMigrationSQLIncludesCoreTables(t *testing.T) {
	for _, table := range []string{
		"calligraphy_auth_users",
		"calligraphy_auth_sessions",
		"calligraphy_artwork_drafts",
		"calligraphy_learning_favorites",
		"calligraphy_learning_practice",
	} {
		if !strings.Contains(PostgresMigrationSQL, table) {
			t.Fatalf("PostgresMigrationSQL missing %s", table)
		}
	}
}
