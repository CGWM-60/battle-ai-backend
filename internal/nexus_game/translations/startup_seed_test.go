package translations

import (
	"context"
	"os"
	"testing"

	"cgwm/battle/internal/db"
	"cgwm/battle/internal/models"
	"gorm.io/gorm"
)

func TestInitialSeedContainsStartupKeys(t *testing.T) {
	rows, err := loadInitialSeedRows()
	if err != nil {
		t.Fatalf("load initial seed rows: %v", err)
	}

	keys := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		keys[row.Key] = struct{}{}
	}

	for _, required := range RequiredStartupKeys {
		if _, ok := keys[required]; !ok {
			t.Errorf("startup key missing from seed: %s", required)
		}
	}
}

func TestStartupSeedRowsPreviewImportValid(t *testing.T) {
	rows, err := loadInitialSeedRows()
	if err != nil {
		t.Fatalf("load initial seed rows: %v", err)
	}

	required := make(map[string]struct{}, len(RequiredStartupKeys))
	for _, key := range RequiredStartupKeys {
		required[key] = struct{}{}
	}

	startupRows := make([]models.TranslationImportRow, 0, len(RequiredStartupKeys))
	for _, row := range rows {
		if _, ok := required[row.Key]; !ok {
			continue
		}
		startupRows = append(startupRows, row)
	}

	if len(startupRows) != len(RequiredStartupKeys) {
		t.Fatalf("expected %d startup rows, got %d", len(RequiredStartupKeys), len(startupRows))
	}

	service := NewTranslationService(nil)
	preview, err := service.PreviewImport(context.Background(), startupRows)
	if err != nil {
		t.Fatalf("PreviewImport returned error: %v", err)
	}

	for _, row := range preview {
		if row.Status != "ok" {
			t.Errorf("startup row %s has status=%s error=%s", row.Key, row.Status, row.Error)
		}
	}
}

func TestCheckStartupKeysInDatabaseRequiresDatabase(t *testing.T) {
	_, err := CheckStartupKeysInDatabase(context.Background(), nil, "fr")
	if err == nil {
		t.Fatal("expected error for nil database")
	}
}

func TestSeedInitialImportProvidesStartupKeysInDatabase(t *testing.T) {
	if os.Getenv("RUN_TRANSLATION_DB_TESTS") != "1" {
		t.Skip("set RUN_TRANSLATION_DB_TESTS=1 to run database integration test")
	}

	var database *gorm.DB
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Skipf("database unavailable: %v", r)
			}
		}()
		database = db.DbConnect()
	}()
	if database == nil {
		t.Skip("database unavailable")
	}

	ctx := context.Background()
	if _, err := SeedInitialImport(ctx, database); err != nil {
		t.Fatalf("SeedInitialImport: %v", err)
	}

	missing, err := CheckStartupKeysInDatabase(ctx, database, "fr")
	if err != nil {
		t.Fatalf("CheckStartupKeysInDatabase: %v", err)
	}
	if len(missing) > 0 {
		t.Fatalf("missing startup keys after seed: %v", missing)
	}
}