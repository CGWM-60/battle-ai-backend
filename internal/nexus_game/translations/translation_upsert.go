package translations

import (
	"context"
	"errors"
	"strings"

	"cgwm/battle/internal/models"
	"gorm.io/gorm"
)

var requiredTranslationDomains = []string{
	"launch", "home", "auth", "common", "anima", "battle", "quests",
	"tribunal", "sandbox", "mmo", "billing", "settings", "profile",
	"prompts", "errors",
}

func isDuplicateEntryError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate entry") ||
		strings.Contains(msg, "error 1062")
}

func findOrCreateTranslationDomain(tx *gorm.DB, code string) (*models.TranslationDomain, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, errors.New("translation domain code is required")
	}

	var domain models.TranslationDomain
	err := tx.Unscoped().Where("code = ?", code).First(&domain).Error
	if err == nil {
		updates := map[string]any{}
		if domain.DeletedAt.Valid {
			updates["deleted_at"] = nil
		}
		if strings.TrimSpace(domain.Name) == "" {
			updates["name"] = code
		}
		if len(updates) > 0 {
			if err := tx.Unscoped().
				Model(&models.TranslationDomain{}).
				Where("id = ?", domain.ID).
				Updates(updates).Error; err != nil {
				return nil, err
			}
			if err := tx.Where("id = ?", domain.ID).First(&domain).Error; err != nil {
				return nil, err
			}
		}
		return &domain, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	domain = models.TranslationDomain{
		Code: code,
		Name: code,
	}
	if err := tx.Create(&domain).Error; err != nil {
		if isDuplicateEntryError(err) {
			if findErr := tx.Unscoped().Where("code = ?", code).First(&domain).Error; findErr != nil {
				return nil, findErr
			}
			if domain.DeletedAt.Valid {
				if restoreErr := tx.Unscoped().
					Model(&models.TranslationDomain{}).
					Where("id = ?", domain.ID).
					Update("deleted_at", nil).Error; restoreErr != nil {
					return nil, restoreErr
				}
				if findErr := tx.Where("id = ?", domain.ID).First(&domain).Error; findErr != nil {
					return nil, findErr
				}
			}
			return &domain, nil
		}
		return nil, err
	}

	return &domain, nil
}

func findOrCreateTranslationKey(
	tx *gorm.DB,
	domainID uint,
	keyValue string,
	defaults models.TranslationKey,
) (*models.TranslationKey, error) {
	keyValue = strings.TrimSpace(keyValue)
	if keyValue == "" {
		return nil, errors.New("translation key is required")
	}

	var key models.TranslationKey
	err := tx.Unscoped().Where("`key` = ?", keyValue).First(&key).Error
	if err == nil {
		updates := map[string]any{}
		if key.DeletedAt.Valid {
			updates["deleted_at"] = nil
		}
		if key.DomainID == 0 || key.DomainID != domainID {
			updates["domain_id"] = domainID
		}
		if strings.TrimSpace(key.TagsJSON) == "" {
			updates["tags_json"] = normalizeTagsJSON("")
		}
		if strings.TrimSpace(key.Status) == "" {
			updates["status"] = "active"
		}
		if len(updates) > 0 {
			if err := tx.Unscoped().
				Model(&models.TranslationKey{}).
				Where("id = ?", key.ID).
				Updates(updates).Error; err != nil {
				return nil, err
			}
			if err := tx.Where("id = ?", key.ID).First(&key).Error; err != nil {
				return nil, err
			}
		}
		return &key, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	defaults.DomainID = domainID
	defaults.Key = keyValue
	defaults.TagsJSON = normalizeTagsJSON(defaults.TagsJSON)
	if strings.TrimSpace(defaults.Status) == "" {
		defaults.Status = "active"
	}

	if err := tx.Create(&defaults).Error; err != nil {
		if isDuplicateEntryError(err) {
			if findErr := tx.Unscoped().Where("`key` = ?", keyValue).First(&key).Error; findErr != nil {
				return nil, findErr
			}
			updates := map[string]any{
				"tags_json": normalizeTagsJSON(key.TagsJSON),
				"status":    "active",
			}
			if key.DeletedAt.Valid {
				updates["deleted_at"] = nil
			}
			if key.DomainID == 0 || key.DomainID != domainID {
				updates["domain_id"] = domainID
			}
			if restoreErr := tx.Unscoped().
				Model(&models.TranslationKey{}).
				Where("id = ?", key.ID).
				Updates(updates).Error; restoreErr != nil {
				return nil, restoreErr
			}
			if findErr := tx.Where("id = ?", key.ID).First(&key).Error; findErr != nil {
				return nil, findErr
			}
			return &key, nil
		}
		return nil, err
	}

	return &defaults, nil
}

func upsertTranslationValue(tx *gorm.DB, keyID uint, locale, value string) error {
	locale = strings.TrimSpace(locale)
	if keyID == 0 || locale == "" {
		return errors.New("translation value key_id and locale are required")
	}

	var translationValue models.TranslationValue
	err := tx.Unscoped().
		Where("key_id = ? AND locale = ?", keyID, locale).
		First(&translationValue).Error
	if err == nil {
		updates := map[string]any{"value": value}
		if translationValue.DeletedAt.Valid {
			updates["deleted_at"] = nil
		}
		return tx.Unscoped().
			Model(&models.TranslationValue{}).
			Where("id = ?", translationValue.ID).
			Updates(updates).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	translationValue = models.TranslationValue{
		KeyID:  keyID,
		Locale: locale,
		Value:  value,
	}
	if err := tx.Create(&translationValue).Error; err != nil {
		if !isDuplicateEntryError(err) {
			return err
		}
		var existing models.TranslationValue
		if findErr := tx.Unscoped().
			Where("key_id = ? AND locale = ?", keyID, locale).
			First(&existing).Error; findErr != nil {
			return findErr
		}
		updates := map[string]any{"value": value}
		if existing.DeletedAt.Valid {
			updates["deleted_at"] = nil
		}
		return tx.Unscoped().
			Model(&models.TranslationValue{}).
			Where("id = ?", existing.ID).
			Updates(updates).Error
	}
	return nil
}

func RepairTranslationDomains(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return nil
	}
	return db.WithContext(ctx).Unscoped().
		Model(&models.TranslationDomain{}).
		Where("code IN ? AND deleted_at IS NOT NULL", requiredTranslationDomains).
		Update("deleted_at", nil).Error
}