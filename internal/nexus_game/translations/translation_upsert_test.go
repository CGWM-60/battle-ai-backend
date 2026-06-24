package translations

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"cgwm/battle/internal/db"
	"cgwm/battle/internal/models"
	"gorm.io/gorm"
)

func TestIsDuplicateEntryError(t *testing.T) {
	if isDuplicateEntryError(nil) {
		t.Fatal("nil error should not be duplicate")
	}
	if !isDuplicateEntryError(errors.New(`Error 1062 (23000): Duplicate entry 'sandbox' for key 'idx_nexus_translation_domains_code'`)) {
		t.Fatal("expected mysql duplicate entry detection")
	}
	if !isDuplicateEntryError(errors.New("duplicate entry for key")) {
		t.Fatal("expected duplicate entry detection")
	}
}

func TestFindOrCreateTranslationDomainRequiresCode(t *testing.T) {
	_, err := findOrCreateTranslationDomain(nil, " ")
	if err == nil {
		t.Fatal("expected error for empty domain code")
	}
}

func TestFindOrCreateTranslationKeyRequiresKey(t *testing.T) {
	_, err := findOrCreateTranslationKey(nil, 1, " ", models.TranslationKey{})
	if err == nil {
		t.Fatal("expected error for empty key")
	}
}

func connectTranslationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
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
	return database
}

func TestFindOrCreateTranslationDomainRestoresSoftDeleted(t *testing.T) {
	database := connectTranslationTestDB(t)
	ctx := context.Background()

	code := "sandbox_test_restore"
	database.Unscoped().Where("code = ?", code).Delete(&models.TranslationDomain{})

	domain := models.TranslationDomain{Code: code, Name: code}
	if err := database.Create(&domain).Error; err != nil {
		t.Fatalf("create domain: %v", err)
	}
	originalID := domain.ID
	if err := database.Delete(&domain).Error; err != nil {
		t.Fatalf("soft delete domain: %v", err)
	}

	var restored *models.TranslationDomain
	if err := database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		restored, err = findOrCreateTranslationDomain(tx, code)
		return err
	}); err != nil {
		t.Fatalf("findOrCreateTranslationDomain: %v", err)
	}

	if restored.ID != originalID {
		t.Fatalf("expected restored domain id=%d, got %d", originalID, restored.ID)
	}
	if restored.DeletedAt.Valid {
		t.Fatalf("expected domain to be restored, deleted_at=%v", restored.DeletedAt)
	}

	database.Unscoped().Where("code = ?", code).Delete(&models.TranslationDomain{})
}

func TestUpsertTranslationRowIsIdempotent(t *testing.T) {
	database := connectTranslationTestDB(t)
	ctx := context.Background()

	row := models.TranslationImportRow{
		Domain: "sandbox",
		Key:    "sandbox.test.idempotent.key",
		Locale: "fr",
		Value:  "Valeur test idempotente",
		Status: "ok",
	}

	for i := 0; i < 3; i++ {
		if err := database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			return upsertTranslationRow(tx, row)
		}); err != nil {
			t.Fatalf("upsertTranslationRow attempt %d: %v", i+1, err)
		}
	}

	var domainCount int64
	if err := database.Model(&models.TranslationDomain{}).
		Where("code = ?", "sandbox").
		Count(&domainCount).Error; err != nil {
		t.Fatalf("count domains: %v", err)
	}
	if domainCount != 1 {
		t.Fatalf("expected 1 sandbox domain, got %d", domainCount)
	}

	var keyCount int64
	if err := database.Model(&models.TranslationKey{}).
		Where("`key` = ?", row.Key).
		Count(&keyCount).Error; err != nil {
		t.Fatalf("count keys: %v", err)
	}
	if keyCount != 1 {
		t.Fatalf("expected 1 translation key, got %d", keyCount)
	}

	var key models.TranslationKey
	if err := database.Where("`key` = ?", row.Key).First(&key).Error; err != nil {
		t.Fatalf("load key: %v", err)
	}
	if key.TagsJSON != "[]" && normalizeTagsJSON(key.TagsJSON) != key.TagsJSON {
		t.Fatalf("unexpected tags_json=%q", key.TagsJSON)
	}
	if strings.TrimSpace(key.TagsJSON) == "" {
		t.Fatalf("tags_json must not be empty")
	}

	var valueCount int64
	if err := database.Model(&models.TranslationValue{}).
		Where("key_id = ? AND locale = ?", key.ID, row.Locale).
		Count(&valueCount).Error; err != nil {
		t.Fatalf("count values: %v", err)
	}
	if valueCount != 1 {
		t.Fatalf("expected 1 translation value, got %d", valueCount)
	}

	database.Where("`key` = ?", row.Key).Delete(&models.TranslationKey{})
}

func TestSeedInitialImportCanRunTwice(t *testing.T) {
	database := connectTranslationTestDB(t)
	ctx := context.Background()

	if _, err := SeedInitialImport(ctx, database); err != nil {
		t.Fatalf("first SeedInitialImport: %v", err)
	}
	if _, err := SeedInitialImport(ctx, database); err != nil {
		t.Fatalf("second SeedInitialImport: %v", err)
	}

	missing, err := CheckStartupKeysInDatabase(ctx, database, "fr")
	if err != nil {
		t.Fatalf("CheckStartupKeysInDatabase: %v", err)
	}
	if len(missing) > 0 {
		t.Fatalf("missing startup keys after double seed: %v", missing)
	}
}

