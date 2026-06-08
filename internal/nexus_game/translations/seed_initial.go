package translations

import (
	"context"
	"embed"
	"errors"

	"cgwm/battle/internal/models"
	"gorm.io/gorm"
)

//go:embed imports/NEXUS_TRANSLATIONS_INITIAL_IMPORT.fr.json
var initialImportFS embed.FS

const initialImportPath = "imports/NEXUS_TRANSLATIONS_INITIAL_IMPORT.fr.json"

func SeedInitialImport(ctx context.Context, database *gorm.DB) (*models.TranslationImport, error) {
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

	locale := payload.Locale
	if locale == "" {
		locale = "fr"
	}

	var existingValues int64
	if err := database.WithContext(ctx).
		Model(&models.TranslationValue{}).
		Where("locale = ?", locale).
		Count(&existingValues).Error; err != nil {
		return nil, err
	}
	if existingValues >= int64(len(payload.Rows)) {
		return nil, nil
	}

	service := NewTranslationService(database)
	preview, err := service.PreviewImport(ctx, payload.Rows)
	if err != nil {
		return nil, err
	}
	for _, row := range preview {
		if row.Status == "error" {
			return nil, errors.New("initial translation import contains invalid rows")
		}
	}

	return service.CommitImportRows(ctx, preview, payload.FileName)
}
