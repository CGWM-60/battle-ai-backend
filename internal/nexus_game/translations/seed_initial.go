package translations

import (
	"context"
	"embed"
	"errors"
	"log"
	"strings"

	"cgwm/battle/internal/models"
	"gorm.io/gorm"
)

//go:embed imports/NEXUS_TRANSLATIONS_INITIAL_IMPORT.fr.json
var initialImportFS embed.FS

const initialImportPath = "imports/NEXUS_TRANSLATIONS_INITIAL_IMPORT.fr.json"

func loadInitialSeedRows() ([]models.TranslationImportRow, error) {
	raw, err := initialImportFS.ReadFile(initialImportPath)
	if err != nil {
		return nil, err
	}

	payload, err := ParseImportPayloadBytes(raw)
	if err != nil {
		return nil, err
	}

	locale := payload.Locale
	if locale == "" {
		locale = "fr"
	}

	return initialSeedRows(payload.Rows, locale), nil
}

func SeedInitialImport(ctx context.Context, database *gorm.DB) (*models.TranslationImport, error) {
	if err := RepairInvalidTranslationTagsJSON(ctx, database); err != nil {
		return nil, err
	}
	if err := PurgeDeprecatedTranslations(ctx, database); err != nil {
		return nil, err
	}

	raw, err := initialImportFS.ReadFile(initialImportPath)
	if err != nil {
		return nil, err
	}

	payload, err := ParseImportPayloadBytes(raw)
	if err != nil {
		return nil, err
	}
	if len(payload.Rows) == 0 {
		return nil, errors.New("initial translation import has no rows")
	}
	log.Printf("[translations] seed initial start rows=%d", len(payload.Rows))

	locale := payload.Locale
	if locale == "" {
		locale = "fr"
	}

	rows := initialSeedRows(payload.Rows, locale)
	rows, err = missingInitialSeedRows(ctx, database, rows)
	if err != nil {
		return nil, err
	}
	log.Printf("[translations] seed initial missing_rows=%d", len(rows))
	var committed *models.TranslationImport
	if len(rows) == 0 {
		log.Printf("[translations] seed initial no missing rows")
		if err := SeedForcedContentDescriptions(ctx, database); err != nil {
			return nil, err
		}
		return nil, nil
	}

	service := NewTranslationService(database)
	preview, err := service.PreviewImport(ctx, rows)
	if err != nil {
		return nil, err
	}
	for _, row := range preview {
		if row.Status == "error" {
			return nil, errors.New("initial translation import contains invalid rows")
		}
	}

	committed, err = service.CommitImportRows(ctx, preview, payload.FileName)
	if err != nil {
		return nil, err
	}
	log.Printf(
		"[translations] seed initial committed import_id=%d rows=%d",
		committed.ID,
		committed.RowCount,
	)
	if err := SeedForcedContentDescriptions(ctx, database); err != nil {
		return nil, err
	}
	return committed, nil
}

func initialSeedRows(importRows []models.TranslationImportRow, fallbackLocale string) []models.TranslationImportRow {
	byKeyLocale := make(map[string]models.TranslationImportRow, len(importRows))
	for _, row := range importRows {
		row.Domain = strings.TrimSpace(row.Domain)
		row.Key = strings.TrimSpace(row.Key)
		row.Locale = strings.TrimSpace(row.Locale)
		if row.Locale == "" {
			row.Locale = fallbackLocale
		}
		if row.Domain == "" || row.Key == "" || row.Locale == "" || !isRetainedTranslation(row.Domain, row.Key) {
			continue
		}
		byKeyLocale[row.Key+"|"+row.Locale] = row
	}
	rows := make([]models.TranslationImportRow, 0, len(byKeyLocale))
	for _, row := range byKeyLocale {
		rows = append(rows, row)
	}
	return rows
}

func missingInitialSeedRows(ctx context.Context, database *gorm.DB, rows []models.TranslationImportRow) ([]models.TranslationImportRow, error) {
	if len(rows) == 0 {
		return nil, nil
	}

	keys := make([]string, 0, len(rows))
	seenKeys := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		if _, exists := seenKeys[row.Key]; exists {
			continue
		}
		seenKeys[row.Key] = struct{}{}
		keys = append(keys, row.Key)
	}

	var existing []struct {
		Key    string
		Locale string
		Value  string
	}
	if err := database.WithContext(ctx).
		Table("nexus_translation_keys as k").
		Select("k.`key` as `key`, v.locale as locale, v.value as value").
		Joins("JOIN nexus_translation_values as v ON v.key_id = k.id").
		Where("k.`key` IN ?", keys).
		Find(&existing).Error; err != nil {
		return nil, err
	}

	existingValues := make(map[string]string, len(existing))
	for _, value := range existing {
		existingValues[value.Key+"|"+value.Locale] = strings.TrimSpace(value.Value)
	}

	missing := make([]models.TranslationImportRow, 0)
	for _, row := range rows {
		if existingValues[row.Key+"|"+row.Locale] != "" {
			continue
		}
		missing = append(missing, row)
	}
	return missing, nil
}
