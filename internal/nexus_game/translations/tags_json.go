package translations

import (
	"context"
	"encoding/json"
	"strings"

	"cgwm/battle/internal/models"
	"gorm.io/gorm"
)

func normalizeTagsJSON(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "[]"
	}
	if !json.Valid([]byte(value)) {
		return "[]"
	}
	return value
}

func RepairInvalidTranslationTagsJSON(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return nil
	}
	return db.WithContext(ctx).
		Model(&models.TranslationKey{}).
		Where("tags_json IS NULL OR tags_json = ''").
		Update("tags_json", "[]").Error
}