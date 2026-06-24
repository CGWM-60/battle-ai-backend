package translations

import (
	"context"
	"strings"

	"cgwm/battle/internal/models"
	"gorm.io/gorm"
)

// FlutterScanCleanupReport summarizes technical key cleanup.
type FlutterScanCleanupReport struct {
	MarkedUnused []string `json:"markedUnused"`
	DeletedKeys  []string `json:"deletedKeys"`
	DeletedValues int     `json:"deletedValues"`
}

// CleanupTechnicalFlutterScanKeys marks or deletes flutter_scan technical keys.
func CleanupTechnicalFlutterScanKeys(ctx context.Context, database *gorm.DB, delete bool) (*FlutterScanCleanupReport, error) {
	if database == nil {
		return nil, nil
	}
	report := &FlutterScanCleanupReport{}

	return report, database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var keys []models.TranslationKey
		if err := tx.Where("import_source = ? OR import_source = ''", "flutter_scan").Find(&keys).Error; err != nil {
			return err
		}
		for _, key := range keys {
			entry := NormalizedFlutterScanEntry{
				FullKey:     key.Key,
				DefaultText: key.Key,
				Text:        key.Key,
				SourceFile:  key.SourceFile,
			}
			ok, _ := isTranslatableFlutterEntry(entry)
			if ok {
				continue
			}
			if isFlutterTechnicalKey(strings.ToLower(key.Key)) ||
				strings.Contains(strings.ToLower(key.Key), ".const.") {
				if delete {
					if err := tx.Where("key_id = ?", key.ID).Delete(&models.TranslationValue{}).Error; err != nil {
						return err
					}
					report.DeletedValues++
					if err := tx.Delete(&key).Error; err != nil {
						return err
					}
					report.DeletedKeys = append(report.DeletedKeys, key.Key)
				} else {
					if err := tx.Model(&key).Updates(map[string]interface{}{
						"status": "unused_candidate",
					}).Error; err != nil {
						return err
					}
					report.MarkedUnused = append(report.MarkedUnused, key.Key)
				}
			}
		}
		return nil
	})
}

// ImportSeedRow is a minimal seed row used by cleanup tooling.
type ImportSeedRow struct {
	Domain string
	Key    string
	Value  string
}

// FilterSeedImportRows removes technical rows from seed import rows.
func FilterSeedImportRows(rows []ImportSeedRow) ([]ImportSeedRow, []string) {
	kept := make([]ImportSeedRow, 0, len(rows))
	removed := make([]string, 0)
	for _, row := range rows {
		entry := NormalizedFlutterScanEntry{
			FullKey:     row.Key,
			Domain:      row.Domain,
			DefaultText: row.Value,
			Text:        row.Value,
		}
		if ok, _ := isTranslatableFlutterEntry(entry); ok {
			kept = append(kept, row)
			continue
		}
		removed = append(removed, row.Key)
	}
	return kept, removed
}