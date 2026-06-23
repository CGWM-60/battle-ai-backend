package translations

import (
	"context"
	"strings"

	"cgwm/battle/internal/models"
	"gorm.io/gorm"
)

var deprecatedTranslationDomains = map[string]struct{}{
	"building":     {},
	"unit":         {},
	"research":     {},
	"nexus_game":   {},
	"sandbox":      {},
	"local_models": {},
}

var deprecatedTranslationKeyPrefixes = []string{
	"home.card.sandbox.",
}

func isRetainedTranslation(domain, key string) bool {
	domain = strings.TrimSpace(domain)
	key = strings.TrimSpace(key)
	if domain == "" || key == "" {
		return false
	}
	if _, deprecated := deprecatedTranslationDomains[domain]; deprecated {
		return false
	}
	for _, prefix := range deprecatedTranslationKeyPrefixes {
		if strings.HasPrefix(key, prefix) {
			return false
		}
	}
	return true
}

func PurgeDeprecatedTranslations(ctx context.Context, database *gorm.DB) error {
	if database == nil {
		return nil
	}

	domains := make([]string, 0, len(deprecatedTranslationDomains))
	for domain := range deprecatedTranslationDomains {
		domains = append(domains, domain)
	}

	return database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := purgeDeprecatedTranslationKeyPrefixes(tx); err != nil {
			return err
		}

		var domainRows []models.TranslationDomain
		if err := tx.Where("code IN ?", domains).Find(&domainRows).Error; err != nil {
			return err
		}
		if len(domainRows) == 0 {
			return nil
		}

		domainIDs := make([]uint, 0, len(domainRows))
		for _, domain := range domainRows {
			domainIDs = append(domainIDs, domain.ID)
		}

		var keyIDs []uint
		if err := tx.Model(&models.TranslationKey{}).Where("domain_id IN ?", domainIDs).Pluck("id", &keyIDs).Error; err != nil {
			return err
		}
		if len(keyIDs) > 0 {
			if err := tx.Where("key_id IN ?", keyIDs).Delete(&models.TranslationValue{}).Error; err != nil {
				return err
			}
			if err := tx.Where("id IN ?", keyIDs).Delete(&models.TranslationKey{}).Error; err != nil {
				return err
			}
		}

		return tx.Where("id IN ?", domainIDs).Delete(&models.TranslationDomain{}).Error
	})
}

func purgeDeprecatedTranslationKeyPrefixes(tx *gorm.DB) error {
	for _, prefix := range deprecatedTranslationKeyPrefixes {
		var keyIDs []uint
		if err := tx.Model(&models.TranslationKey{}).Where("`key` LIKE ?", prefix+"%").Pluck("id", &keyIDs).Error; err != nil {
			return err
		}
		if len(keyIDs) == 0 {
			continue
		}
		if err := tx.Where("key_id IN ?", keyIDs).Delete(&models.TranslationValue{}).Error; err != nil {
			return err
		}
		if err := tx.Where("id IN ?", keyIDs).Delete(&models.TranslationKey{}).Error; err != nil {
			return err
		}
	}
	return nil
}
