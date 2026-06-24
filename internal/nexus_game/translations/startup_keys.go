package translations

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"
)

var RequiredStartupKeys = []string{
	"launch.button.skip",
	"launch.button.enter_nexus",
	"launch.status.preparing",
	"launch.status.portal_ready",
	"launch.status.audio_locked",
	"launch.status.audio_unavailable",
	"launch.boot.net",
	"launch.boot.sig",
	"launch.boot.bia",
	"launch.boot.qst",
	"launch.boot.rpg",
	"launch.boot.coop",
	"launch.boot.nxs",
	"launch.boot.sys",
	"home.ui.text.connexion",
	"home.ui.text.creer_un_compte",
	"home.ui.text.le_reseau_vous_attend",
	"auth.ui.text.creer_un_profil",
	"auth.ui.text.initialiser_la_connexion",
}

func CheckStartupKeysInDatabase(ctx context.Context, db *gorm.DB, locale string) ([]string, error) {
	if db == nil {
		return nil, errors.New("database is nil")
	}

	locale = strings.TrimSpace(locale)
	if locale == "" {
		locale = "fr"
	}

	var existing []struct {
		Key string
	}
	if err := db.WithContext(ctx).
		Table("nexus_translation_keys as k").
		Select("k.`key` as `key`").
		Joins("JOIN nexus_translation_values as v ON v.key_id = k.id").
		Where("k.`key` IN ? AND v.locale = ? AND TRIM(v.value) <> ''", RequiredStartupKeys, locale).
		Find(&existing).Error; err != nil {
		return nil, err
	}

	present := make(map[string]struct{}, len(existing))
	for _, row := range existing {
		present[row.Key] = struct{}{}
	}

	missing := make([]string, 0)
	for _, key := range RequiredStartupKeys {
		if _, ok := present[key]; !ok {
			missing = append(missing, key)
		}
	}
	return missing, nil
}