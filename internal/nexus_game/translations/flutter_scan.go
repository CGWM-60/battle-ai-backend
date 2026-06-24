package translations

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"cgwm/battle/internal/models"
	"gorm.io/gorm"
)

// FlutterScanEntry is one extracted string from the Flutter codebase.
type FlutterScanEntry struct {
	ID                 string   `json:"id"`
	Text               string   `json:"text"`
	SourceFile         string   `json:"sourceFile"`
	SourceLine         int      `json:"sourceLine"`
	Usage              string   `json:"usage"`
	Kind               string   `json:"kind"`
	Module             string   `json:"module"`
	SuggestedNamespace string   `json:"suggestedNamespace"`
	SuggestedKey       string   `json:"suggestedKey"`
	Tags               []string `json:"tags"`
	NeedsParams        bool     `json:"needsParams"`
	Params             []string `json:"params"`
	DefaultText        string   `json:"defaultText"`
}

// FlutterScanImportReport summarizes a flutter_scan upsert run.
type FlutterScanImportReport struct {
	Source            string   `json:"source"`
	CreatedKeys       []string `json:"createdKeys"`
	UpdatedKeys       []string `json:"updatedKeys"`
	CreatedTags       []string `json:"createdTags"`
	CreatedNamespaces []string `json:"createdNamespaces"`
	SkippedEntries    []string `json:"skippedEntries"`
	Conflicts         []string `json:"conflicts"`
	DuplicateTexts    []string `json:"duplicateTexts"`
	ReusedCommonKeys  []string `json:"reusedCommonKeys"`
}

func ParseFlutterScanBytes(raw []byte) ([]FlutterScanEntry, error) {
	var entries []FlutterScanEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func (s *dbTranslationService) ImportFlutterScan(
	ctx context.Context,
	entries []FlutterScanEntry,
	defaultLocale string,
) (*FlutterScanImportReport, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database is required for flutter_scan import")
	}
	locale := strings.TrimSpace(defaultLocale)
	if locale == "" {
		locale = "fr"
	}

	report := &FlutterScanImportReport{
		Source: "flutter_scan",
	}
	seenTextToKey := make(map[string]string)

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, entry := range entries {
			fullKey := strings.TrimSpace(entry.SuggestedKey)
			if fullKey == "" {
				report.SkippedEntries = append(report.SkippedEntries, entry.ID)
				continue
			}
			domain, keyName := splitDomainAndKey(fullKey)
			if !isRetainedTranslation(domain, fullKey) {
				report.SkippedEntries = append(report.SkippedEntries, fullKey)
				continue
			}

			value := strings.TrimSpace(entry.DefaultText)
			if value == "" {
				value = strings.TrimSpace(entry.Text)
			}
			if value == "" {
				report.SkippedEntries = append(report.SkippedEntries, fullKey)
				continue
			}

			if existingKey, ok := seenTextToKey[value]; ok && existingKey != fullKey {
				report.DuplicateTexts = append(report.DuplicateTexts, fullKey+"<=>"+existingKey)
			} else {
				seenTextToKey[value] = fullKey
			}

			createdDomain, err := ensureDomain(tx, domain, report)
			if err != nil {
				return err
			}
			_ = createdDomain

			created, updated, err := upsertFlutterScanKey(tx, fullKey, domain, keyName, locale, value, entry, report)
			if err != nil {
				return err
			}
			if created {
				report.CreatedKeys = append(report.CreatedKeys, fullKey)
			}
			if updated {
				report.UpdatedKeys = append(report.UpdatedKeys, fullKey)
			}

			if err := upsertTagsForKey(tx, fullKey, entry.Tags, report); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return report, nil
}

func splitDomainAndKey(fullKey string) (string, string) {
	parts := strings.Split(fullKey, ".")
	if len(parts) == 0 {
		return "common", fullKey
	}
	return parts[0], fullKey
}

func ensureDomain(tx *gorm.DB, code string, report *FlutterScanImportReport) (bool, error) {
	var domain models.TranslationDomain
	err := tx.Where("code = ?", code).First(&domain).Error
	if err == nil {
		return false, nil
	}
	if err != gorm.ErrRecordNotFound {
		return false, err
	}
	domain = models.TranslationDomain{Code: code, Name: code}
	if err := tx.Create(&domain).Error; err != nil {
		return false, err
	}
	report.CreatedNamespaces = append(report.CreatedNamespaces, code)
	return true, nil
}

func upsertFlutterScanKey(
	tx *gorm.DB,
	fullKey, domain, keyName, locale, value string,
	entry FlutterScanEntry,
	report *FlutterScanImportReport,
) (created bool, updated bool, err error) {
	var domainRow models.TranslationDomain
	if err = tx.Where("code = ?", domain).
		FirstOrCreate(&domainRow, models.TranslationDomain{Code: domain, Name: domain}).Error; err != nil {
		return false, false, err
	}

	var keyRow models.TranslationKey
	err = tx.Where("domain_id = ? AND `key` = ?", domainRow.ID, fullKey).First(&keyRow).Error
	if err == gorm.ErrRecordNotFound {
		keyRow = models.TranslationKey{
			DomainID:     domainRow.ID,
			Key:          fullKey,
			Description:  buildFlutterScanDescription(entry),
			SourceFile:   entry.SourceFile,
			SourceLine:   entry.SourceLine,
			SourceModule: entry.Module,
			Kind:         entry.Kind,
			Status:       "active",
			Reviewed:     false,
			ImportSource: "flutter_scan",
			TagsJSON:     tagsJSON(entry.Tags),
		}
		if err = tx.Create(&keyRow).Error; err != nil {
			return false, false, err
		}
		created = true
	} else if err != nil {
		return false, false, err
	} else {
		updates := map[string]interface{}{
			"description":   buildFlutterScanDescription(entry),
			"source_file":   entry.SourceFile,
			"source_line":   entry.SourceLine,
			"source_module": entry.Module,
			"kind":          entry.Kind,
			"status":        "active",
			"import_source": "flutter_scan",
			"tags_json":     tagsJSON(entry.Tags),
		}
		if err = tx.Model(&keyRow).Updates(updates).Error; err != nil {
			return false, false, err
		}
		updated = true
	}

	val := models.TranslationValue{KeyID: keyRow.ID, Locale: locale, Value: value}
	var existing models.TranslationValue
	findErr := tx.Where("key_id = ? AND locale = ?", keyRow.ID, locale).First(&existing).Error
	if findErr == gorm.ErrRecordNotFound {
		if err = tx.Create(&val).Error; err != nil {
			return created, updated, err
		}
	} else if findErr != nil {
		return created, updated, findErr
	} else if strings.TrimSpace(existing.Value) != value {
		if err = tx.Model(&existing).Update("value", value).Error; err != nil {
			return created, updated, err
		}
	}

	if strings.HasPrefix(fullKey, "common.") {
		report.ReusedCommonKeys = append(report.ReusedCommonKeys, fullKey)
	}
	return created, updated, nil
}

func buildFlutterScanDescription(entry FlutterScanEntry) string {
	parts := []string{
		fmt.Sprintf("Extracted from %s:%d", entry.SourceFile, entry.SourceLine),
		"usage=" + entry.Usage,
	}
	if entry.SuggestedNamespace != "" {
		parts = append(parts, "namespace="+entry.SuggestedNamespace)
	}
	return strings.Join(parts, " | ")
}

func tagsJSON(tags []string) string {
	if len(tags) == 0 {
		return "[]"
	}
	raw, err := json.Marshal(tags)
	if err != nil {
		return "[]"
	}
	return string(raw)
}

func upsertTagsForKey(tx *gorm.DB, fullKey string, tags []string, report *FlutterScanImportReport) error {
	if len(tags) == 0 {
		return nil
	}
	var keyRow models.TranslationKey
	if err := tx.Where("`key` = ?", fullKey).First(&keyRow).Error; err != nil {
		return err
	}
	for _, tagCode := range tags {
		tagCode = strings.TrimSpace(tagCode)
		if tagCode == "" {
			continue
		}
		var tag models.TranslationTag
		err := tx.Where("code = ?", tagCode).First(&tag).Error
		if err == gorm.ErrRecordNotFound {
			tag = models.TranslationTag{Code: tagCode, Name: tagCode}
			if err = tx.Create(&tag).Error; err != nil {
				return err
			}
			report.CreatedTags = append(report.CreatedTags, tagCode)
		} else if err != nil {
			return err
		}
		link := models.TranslationKeyTag{KeyID: keyRow.ID, TagID: tag.ID}
		if err := tx.Where(link).FirstOrCreate(&link).Error; err != nil {
			return err
		}
	}
	return nil
}