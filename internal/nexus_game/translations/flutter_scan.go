package translations

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"cgwm/battle/internal/models"
	"gorm.io/gorm"
)

// FlutterScanImportOptions controls upsert behavior.
type FlutterScanImportOptions struct {
	Force bool
}

// FlutterScanImportReport summarizes a flutter_scan upsert run.
type FlutterScanImportReport struct {
	Source            string   `json:"source"`
	CreatedKeys       []string `json:"createdKeys"`
	UpdatedKeys       []string `json:"updatedKeys"`
	CreatedTags       []string `json:"createdTags"`
	CreatedNamespaces []string `json:"createdNamespaces"`
	SkippedEntries    []string `json:"skippedEntries"`
	SkippedTechnical  []string `json:"skippedTechnical"`
	SkippedAssets     []string `json:"skippedAssets"`
	SkippedRoutes     []string `json:"skippedRoutes"`
	SkippedProviders  []string `json:"skippedProviders"`
	SkippedStorageKeys []string `json:"skippedStorageKeys"`
	Conflicts         []string `json:"conflicts"`
	DuplicateTexts    []string `json:"duplicateTexts"`
	ReusedCommonKeys  []string `json:"reusedCommonKeys"`
	MissingTranslations []string `json:"missingTranslations,omitempty"`
}

func (s *dbTranslationService) ImportFlutterScan(
	ctx context.Context,
	entries []FlutterScanEntry,
	defaultLocale string,
	opts *FlutterScanImportOptions,
) (*FlutterScanImportReport, error) {
	force := false
	if opts != nil {
		force = opts.Force
	}
	return importFlutterScanEntries(ctx, s.db, entries, defaultLocale, force)
}

func importFlutterScanEntries(
	ctx context.Context,
	database *gorm.DB,
	entries []FlutterScanEntry,
	defaultLocale string,
	force bool,
) (*FlutterScanImportReport, error) {
	if database == nil {
		return nil, fmt.Errorf("database is required for flutter_scan import")
	}
	locale := strings.TrimSpace(defaultLocale)
	if locale == "" {
		locale = "fr"
	}

	report := &FlutterScanImportReport{Source: "flutter_scan"}
	seenTextToKey := make(map[string]string)

	err := database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, raw := range entries {
			entry := raw.Normalized()
			ok, reason := isTranslatableFlutterEntry(entry)
			if !ok {
				appendSkipped(report, entry, reason)
				continue
			}

			fullKey := entry.FullKey
			domain, _ := splitDomainAndKey(fullKey)
			value := entry.DefaultText
			if value == "" {
				value = entry.Text
			}

			if existingKey, ok := seenTextToKey[value]; ok && existingKey != fullKey {
				report.DuplicateTexts = append(report.DuplicateTexts, fullKey+"<=>"+existingKey)
			} else {
				seenTextToKey[value] = fullKey
			}

			if _, err := ensureDomain(tx, domain, report); err != nil {
				return err
			}

			created, updated, err := upsertFlutterScanKey(tx, fullKey, domain, locale, value, entry, report, force)
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

// PreviewFlutterScan filters entries without touching the database.
func PreviewFlutterScan(entries []FlutterScanEntry) (*FlutterScanImportReport, error) {
	report := &FlutterScanImportReport{Source: "flutter_scan"}
	seen := make(map[string]string)
	for _, raw := range entries {
		entry := raw.Normalized()
		ok, reason := isTranslatableFlutterEntry(entry)
		if !ok {
			appendSkipped(report, entry, reason)
			continue
		}
		if prev, ok := seen[entry.DefaultText]; ok && prev != entry.FullKey {
			report.DuplicateTexts = append(report.DuplicateTexts, entry.FullKey+"<=>"+prev)
		} else {
			seen[entry.DefaultText] = entry.FullKey
		}
		report.CreatedKeys = append(report.CreatedKeys, entry.FullKey)
		if strings.HasPrefix(entry.FullKey, "common.") {
			report.ReusedCommonKeys = append(report.ReusedCommonKeys, entry.FullKey)
		}
	}
	return report, nil
}

func appendSkipped(report *FlutterScanImportReport, entry NormalizedFlutterScanEntry, reason FlutterScanSkipReason) {
	label := entry.FullKey
	if label == "" {
		label = entry.ID
	}
	report.SkippedEntries = append(report.SkippedEntries, label)
	switch reason {
	case SkipTechnical:
		report.SkippedTechnical = append(report.SkippedTechnical, label)
	case SkipAssets:
		report.SkippedAssets = append(report.SkippedAssets, label)
	case SkipRoutes:
		report.SkippedRoutes = append(report.SkippedRoutes, label)
	case SkipProviders:
		report.SkippedProviders = append(report.SkippedProviders, label)
	case SkipStorageKeys:
		report.SkippedStorageKeys = append(report.SkippedStorageKeys, label)
	default:
		report.SkippedTechnical = append(report.SkippedTechnical, label)
	}
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
	fullKey, domain, locale, value string,
	entry NormalizedFlutterScanEntry,
	report *FlutterScanImportReport,
	force bool,
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

	val := models.TranslationValue{
		KeyID:  keyRow.ID,
		Locale: locale,
		Value:  value,
		Source: "flutter_scan",
	}
	var existing models.TranslationValue
	findErr := tx.Where("key_id = ? AND locale = ?", keyRow.ID, locale).First(&existing).Error
	if findErr == gorm.ErrRecordNotFound {
		if err = tx.Create(&val).Error; err != nil {
			return created, updated, err
		}
	} else if findErr != nil {
		return created, updated, findErr
	} else if existing.Reviewed && !force {
		// Keep human-reviewed translations intact.
	} else if strings.TrimSpace(existing.Value) != value {
		updates := map[string]interface{}{
			"value":  value,
			"source": "flutter_scan",
		}
		if err = tx.Model(&existing).Updates(updates).Error; err != nil {
			return created, updated, err
		}
	}

	if strings.HasPrefix(fullKey, "common.") {
		report.ReusedCommonKeys = append(report.ReusedCommonKeys, fullKey)
	}
	return created, updated, nil
}

func buildFlutterScanDescription(entry NormalizedFlutterScanEntry) string {
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