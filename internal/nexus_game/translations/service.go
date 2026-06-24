package translations

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"cgwm/battle/internal/models"
	tribunaladapters "cgwm/battle/internal/nexus_tribunal/adapters"
	"gorm.io/gorm"
)

// TranslationEntry représente une traduction ligne par ligne.
type TranslationEntry struct {
	Key       string    `json:"key"`
	Locale    string    `json:"locale"`
	Value     string    `json:"value"`
	Domain    string    `json:"domain"` // ex: nexus_game, nexus_game_city, common...
	UpdatedAt time.Time `json:"updated_at"`
	UpdatedBy string    `json:"updated_by"`
}

// TranslationService est le contrat pour le module traduction serveur.
// Le serveur reste la source de vérité (AGENTS.md).
type TranslationService interface {
	// GetTranslations retourne les traductions pour une langue (optionnellement filtrées par domaines).
	GetTranslations(ctx context.Context, lang string, domains []string) (map[string]string, error)

	// UpsertBatch permet à l'admin d'importer/mettre à jour par lot.
	UpsertBatch(ctx context.Context, entries []TranslationEntry) error

	// ListMissingKeys renvoie les clés utilisées dans le code mais absentes pour une langue.
	ListMissingKeys(ctx context.Context, lang string) ([]string, error)

	// GetAvailableLanguages liste les langues supportées.
	GetAvailableLanguages(ctx context.Context) ([]string, error)

	// GetDomains liste les domaines disponibles.
	GetDomains(ctx context.Context) ([]models.TranslationDomain, error)

	// GetDomainTranslations retourne les traductions pour un domaine spécifique.
	GetDomainTranslations(ctx context.Context, domain string, lang string) (map[string]string, error)

	// SetUserLocale met à jour la préférence de langue de l'utilisateur.
	SetUserLocale(ctx context.Context, userID uint, locale string) error

	// LogMissingKey enregistre une clé manquante signalée par le client.
	LogMissingKey(ctx context.Context, key string, locale string) error

	// Admin methods for POINT 03
	GetAllDomains(ctx context.Context) ([]models.TranslationDomain, error)
	CreateDomain(ctx context.Context, d *models.TranslationDomain) error
	GetAllKeys(ctx context.Context) ([]models.TranslationKey, error)
	CreateKey(ctx context.Context, k *models.TranslationKey) error
	UpdateKey(ctx context.Context, id uint, k *models.TranslationKey) error
	DeleteKey(ctx context.Context, id uint) error
	GetAllValues(ctx context.Context) ([]models.TranslationValue, error)
	UpdateValue(ctx context.Context, id uint, v *models.TranslationValue) error
	PreviewImport(ctx context.Context, rows []models.TranslationImportRow) ([]models.TranslationImportRow, error)
	CommitImport(ctx context.Context, importID uint) error
	CommitImportRows(ctx context.Context, rows []models.TranslationImportRow, fileName string) (*models.TranslationImport, error)
	GetImports(ctx context.Context) ([]models.TranslationImport, error)
	GetImportByID(ctx context.Context, id uint) (*models.TranslationImport, error)
	GetMissing(ctx context.Context) ([]models.TranslationMissingLog, error)
	ExportTranslations(ctx context.Context, lang string) (map[string]string, error)
	BatchUpdate(ctx context.Context, entries []TranslationEntry) error
	GetSupportedLocaleCatalog(ctx context.Context) ([]TranslationLocaleOption, error)
	GetAITranslationProviders(ctx context.Context) ([]TranslationAIProviderStatus, error)
	AITranslateMissing(ctx context.Context, req AITranslateMissingRequest) (*AITranslateMissingResult, error)
}

// dbTranslationService implémente TranslationService avec GORM.
type dbTranslationService struct {
	db *gorm.DB
}

type TranslationAIProviderStatus struct {
	ProviderType string `json:"providerType"`
	DisplayName  string `json:"displayName"`
	DefaultModel string `json:"defaultModel"`
	Configured   bool   `json:"configured"`
}

type TranslationLocaleOption struct {
	Code  string `json:"code"`
	Label string `json:"label"`
	Group string `json:"group"`
}

type AITranslateMissingRequest struct {
	TargetLocale  string   `json:"targetLocale"`
	SourceLocale  string   `json:"sourceLocale"`
	Provider      string   `json:"provider"`
	Model         string   `json:"model"`
	APIKey        string   `json:"apiKey"`
	LocalEndpoint string   `json:"localEndpoint"`
	Limit         int      `json:"limit"`
	Domains       []string `json:"domains"`
	Keys          []string `json:"keys"`
}

type AITranslateMissingResult struct {
	Provider     string                  `json:"provider"`
	Model        string                  `json:"model"`
	SourceLocale string                  `json:"sourceLocale"`
	TargetLocale string                  `json:"targetLocale"`
	Translated   int                     `json:"translated"`
	Errors       int                     `json:"errors"`
	Items        []AITranslatedValueItem `json:"items"`
}

type AITranslatedValueItem struct {
	Domain string `json:"domain"`
	Key    string `json:"key"`
	Source string `json:"source"`
	Value  string `json:"value"`
	Error  string `json:"error,omitempty"`
}

func NewTranslationService(db *gorm.DB) TranslationService {
	return &dbTranslationService{db: db}
}

func (s *dbTranslationService) GetTranslations(ctx context.Context, lang string, domains []string) (map[string]string, error) {
	var results []struct {
		TranslationKey string
		Value          string
	}

	query := s.db.Table("nexus_translation_values as v").
		Select("k.`key` as translation_key, v.value as value").
		Joins("JOIN nexus_translation_keys as k ON k.id = v.key_id").
		Joins("JOIN nexus_translation_domains as d ON d.id = k.domain_id").
		Where("v.locale = ?", lang)

	if len(domains) > 0 {
		query = query.Where("d.code IN ?", domains)
	}

	if err := query.Find(&results).Error; err != nil {
		return nil, err
	}

	m := make(map[string]string, len(results))
	for _, r := range results {
		m[r.TranslationKey] = r.Value
	}

	// Fallback sur FR si certaines clés manquent pour la langue demandée (simple implémentation)
	if lang != "fr" {
		var frResults []struct {
			TranslationKey string
			Value          string
		}
		frQuery := s.db.Table("nexus_translation_values as v").
			Select("k.`key` as translation_key, v.value as value").
			Joins("JOIN nexus_translation_keys as k ON k.id = v.key_id").
			Joins("JOIN nexus_translation_domains as d ON d.id = k.domain_id").
			Where("v.locale = ?", "fr")
		if len(domains) > 0 {
			frQuery = frQuery.Where("d.code IN ?", domains)
		}
		_ = frQuery.Find(&frResults).Error
		for _, r := range frResults {
			if _, ok := m[r.TranslationKey]; !ok {
				m[r.TranslationKey] = r.Value
			}
		}
	}

	return m, nil
}

func (s *dbTranslationService) GetDomains(ctx context.Context) ([]models.TranslationDomain, error) {
	var domains []models.TranslationDomain
	if err := s.db.Where("deleted_at IS NULL").Order("code").Find(&domains).Error; err != nil {
		return nil, err
	}
	return domains, nil
}

func (s *dbTranslationService) GetDomainTranslations(ctx context.Context, domain string, lang string) (map[string]string, error) {
	var results []struct {
		TranslationKey string
		Value          string
	}

	query := s.db.Table("nexus_translation_values as v").
		Select("k.`key` as translation_key, v.value as value").
		Joins("JOIN nexus_translation_keys as k ON k.id = v.key_id").
		Joins("JOIN nexus_translation_domains as d ON d.id = k.domain_id").
		Where("d.code = ? AND v.locale = ?", domain, lang)

	if err := query.Find(&results).Error; err != nil {
		return nil, err
	}

	m := make(map[string]string, len(results))
	for _, r := range results {
		m[r.TranslationKey] = r.Value
	}
	return m, nil
}

func (s *dbTranslationService) SetUserLocale(ctx context.Context, userID uint, locale string) error {
	pref := models.UserLocalePreference{
		UserID: userID,
		Locale: locale,
	}
	return s.db.Where(models.UserLocalePreference{UserID: userID}).
		Assign(pref).
		FirstOrCreate(&pref).Error
}

func (s *dbTranslationService) LogMissingKey(ctx context.Context, key string, locale string) error {
	var log models.TranslationMissingLog
	err := s.db.Where("`key` = ? AND locale = ?", key, locale).First(&log).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	if err == gorm.ErrRecordNotFound {
		log = models.TranslationMissingLog{Key: key, Locale: locale, Count: 1}
		return s.db.Create(&log).Error
	}
	log.Count++
	return s.db.Save(&log).Error
}

func (s *dbTranslationService) UpsertBatch(ctx context.Context, entries []TranslationEntry) error {
	// Implémentation basique pour le point 02 (batch sera plus complet plus tard)
	for _, e := range entries {
		if !isRetainedTranslation(e.Domain, e.Key) {
			continue
		}

		// Trouver ou créer le domaine
		var domain models.TranslationDomain
		if err := s.db.Where("code = ?", e.Domain).FirstOrCreate(&domain, models.TranslationDomain{Code: e.Domain, Name: e.Domain}).Error; err != nil {
			return err
		}

		// Trouver ou créer la clé
		var key models.TranslationKey
		if err := s.db.Where("domain_id = ? AND `key` = ?", domain.ID, e.Key).FirstOrCreate(&key, models.TranslationKey{DomainID: domain.ID, Key: e.Key}).Error; err != nil {
			return err
		}

		// Upsert la valeur
		val := models.TranslationValue{
			KeyID:  key.ID,
			Locale: e.Locale,
			Value:  e.Value,
		}
		if err := s.db.Where("key_id = ? AND locale = ?", key.ID, e.Locale).
			Assign(val).
			FirstOrCreate(&val).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *dbTranslationService) ListMissingKeys(ctx context.Context, lang string) ([]string, error) {
	var logs []models.TranslationMissingLog
	if err := s.db.Where("locale = ?", lang).Find(&logs).Error; err != nil {
		return nil, err
	}
	keys := make([]string, len(logs))
	for i, l := range logs {
		keys[i] = l.Key
	}
	return keys, nil
}

func (s *dbTranslationService) GetAvailableLanguages(ctx context.Context) ([]string, error) {
	var langs []string
	if err := s.db.Model(&models.TranslationValue{}).
		Distinct().
		Pluck("locale", &langs).Error; err != nil {
		return nil, err
	}
	return langs, nil
}

// Admin method implementations for POINT 03

func (s *dbTranslationService) GetAllDomains(ctx context.Context) ([]models.TranslationDomain, error) {
	var domains []models.TranslationDomain
	if err := s.db.Find(&domains).Error; err != nil {
		return nil, err
	}
	return domains, nil
}

func (s *dbTranslationService) CreateDomain(ctx context.Context, d *models.TranslationDomain) error {
	return s.db.Create(d).Error
}

func (s *dbTranslationService) GetAllKeys(ctx context.Context) ([]models.TranslationKey, error) {
	var keys []models.TranslationKey
	if err := s.db.Preload("Domain").Find(&keys).Error; err != nil {
		return nil, err
	}
	return keys, nil
}

func (s *dbTranslationService) CreateKey(ctx context.Context, k *models.TranslationKey) error {
	return s.db.Create(k).Error
}

func (s *dbTranslationService) UpdateKey(ctx context.Context, id uint, k *models.TranslationKey) error {
	return s.db.Model(&models.TranslationKey{ID: id}).Updates(k).Error
}

func (s *dbTranslationService) DeleteKey(ctx context.Context, id uint) error {
	return s.db.Delete(&models.TranslationKey{}, id).Error
}

func (s *dbTranslationService) GetAllValues(ctx context.Context) ([]models.TranslationValue, error) {
	var values []models.TranslationValue
	if err := s.db.Preload("Key").Find(&values).Error; err != nil {
		return nil, err
	}
	return values, nil
}

func (s *dbTranslationService) UpdateValue(ctx context.Context, id uint, v *models.TranslationValue) error {
	return s.db.Model(&models.TranslationValue{ID: id}).Updates(v).Error
}

func (s *dbTranslationService) PreviewImport(ctx context.Context, rows []models.TranslationImportRow) ([]models.TranslationImportRow, error) {
	// Simulate preview: mark errors but do not persist
	for i := range rows {
		if rows[i].Key == "" || rows[i].Value == "" || rows[i].Locale == "" {
			rows[i].Status = "error"
			rows[i].Error = "missing required fields"
		} else if !isRetainedTranslation(rows[i].Domain, rows[i].Key) {
			rows[i].Status = "skipped"
			rows[i].Error = "deprecated translation domain"
		} else {
			rows[i].Status = "ok"
		}
	}
	return rows, nil
}

func (s *dbTranslationService) CommitImport(ctx context.Context, importID uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var imp models.TranslationImport
		if err := tx.First(&imp, importID).Error; err != nil {
			return err
		}

		var rows []models.TranslationImportRow
		if err := tx.Where("import_id = ?", importID).Find(&rows).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return errors.New("import has no rows to commit")
		}

		for _, row := range rows {
			if row.Status == "error" {
				return errors.New("import contains invalid rows")
			}
			if row.Status == "skipped" || !isRetainedTranslation(row.Domain, row.Key) {
				continue
			}
			if err := upsertTranslationRow(tx, row); err != nil {
				return err
			}
		}

		imp.Status = "committed"
		imp.RowCount = len(rows)
		return tx.Save(&imp).Error
	})
}

func (s *dbTranslationService) CommitImportRows(ctx context.Context, rows []models.TranslationImportRow, fileName string) (*models.TranslationImport, error) {
	if len(rows) == 0 {
		return nil, errors.New("rows are required")
	}
	if fileName == "" {
		fileName = "admin-import.json"
	}

	var committedImport *models.TranslationImport
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		imp := models.TranslationImport{
			FileName: fileName,
			Status:   "committing",
			RowCount: len(rows),
		}
		if err := tx.Create(&imp).Error; err != nil {
			return err
		}

		for i := range rows {
			rows[i].ImportID = imp.ID
			if rows[i].Key == "" || rows[i].Value == "" || rows[i].Locale == "" || rows[i].Domain == "" {
				rows[i].Status = "error"
				if rows[i].Error == "" {
					rows[i].Error = "missing required fields"
				}
			} else if !isRetainedTranslation(rows[i].Domain, rows[i].Key) {
				rows[i].Status = "skipped"
				if rows[i].Error == "" {
					rows[i].Error = "deprecated translation domain"
				}
			} else if rows[i].Status == "" {
				rows[i].Status = "ok"
			}

			if err := tx.Create(&rows[i]).Error; err != nil {
				return err
			}
			if rows[i].Status == "error" {
				return errors.New("import contains invalid rows")
			}
			if rows[i].Status == "skipped" {
				continue
			}
			if err := upsertTranslationRow(tx, rows[i]); err != nil {
				return err
			}
		}

		imp.Status = "committed"
		if err := tx.Save(&imp).Error; err != nil {
			return err
		}
		committedImport = &imp
		return nil
	})
	if err != nil {
		return nil, err
	}
	return committedImport, nil
}

func (s *dbTranslationService) GetImports(ctx context.Context) ([]models.TranslationImport, error) {
	var imports []models.TranslationImport
	if err := s.db.Find(&imports).Error; err != nil {
		return nil, err
	}
	return imports, nil
}

func (s *dbTranslationService) GetImportByID(ctx context.Context, id uint) (*models.TranslationImport, error) {
	var imp models.TranslationImport
	if err := s.db.First(&imp, id).Error; err != nil {
		return nil, err
	}
	return &imp, nil
}

func (s *dbTranslationService) GetMissing(ctx context.Context) ([]models.TranslationMissingLog, error) {
	var logs []models.TranslationMissingLog
	if err := s.db.Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

func (s *dbTranslationService) ExportTranslations(ctx context.Context, lang string) (map[string]string, error) {
	return s.GetTranslations(ctx, lang, nil)
}

func (s *dbTranslationService) BatchUpdate(ctx context.Context, entries []TranslationEntry) error {
	return s.UpsertBatch(ctx, entries)
}

func (s *dbTranslationService) GetSupportedLocaleCatalog(ctx context.Context) ([]TranslationLocaleOption, error) {
	return []TranslationLocaleOption{
		{Code: "fr", Label: "Français source", Group: "Base"},
		{Code: "fr-FR", Label: "Français France", Group: "Europe"},
		{Code: "en-GB", Label: "English UK", Group: "Europe"},
		{Code: "en-US", Label: "English US", Group: "US"},
		{Code: "es-US", Label: "Español US", Group: "US"},
		{Code: "de-DE", Label: "Deutsch", Group: "Europe"},
		{Code: "es-ES", Label: "Español España", Group: "Europe"},
		{Code: "it-IT", Label: "Italiano", Group: "Europe"},
		{Code: "pt-PT", Label: "Português Portugal", Group: "Europe"},
		{Code: "nl-NL", Label: "Nederlands", Group: "Europe"},
		{Code: "sv-SE", Label: "Svenska", Group: "Europe"},
		{Code: "da-DK", Label: "Dansk", Group: "Europe"},
		{Code: "fi-FI", Label: "Suomi", Group: "Europe"},
		{Code: "no-NO", Label: "Norsk", Group: "Europe"},
		{Code: "pl-PL", Label: "Polski", Group: "Europe"},
		{Code: "cs-CZ", Label: "Čeština", Group: "Europe"},
		{Code: "sk-SK", Label: "Slovenčina", Group: "Europe"},
		{Code: "sl-SI", Label: "Slovenščina", Group: "Europe"},
		{Code: "hr-HR", Label: "Hrvatski", Group: "Europe"},
		{Code: "hu-HU", Label: "Magyar", Group: "Europe"},
		{Code: "ro-RO", Label: "Română", Group: "Europe"},
		{Code: "bg-BG", Label: "Български", Group: "Europe"},
		{Code: "el-GR", Label: "Ελληνικά", Group: "Europe"},
		{Code: "et-EE", Label: "Eesti", Group: "Europe"},
		{Code: "lv-LV", Label: "Latviešu", Group: "Europe"},
		{Code: "lt-LT", Label: "Lietuvių", Group: "Europe"},
		{Code: "ga-IE", Label: "Gaeilge", Group: "Europe"},
		{Code: "mt-MT", Label: "Malti", Group: "Europe"},
		{Code: "is-IS", Label: "Íslenska", Group: "Europe"},
		{Code: "uk-UA", Label: "Українська", Group: "Europe"},
	}, nil
}

func (s *dbTranslationService) GetAITranslationProviders(ctx context.Context) ([]TranslationAIProviderStatus, error) {
	providers := []string{"mistral", "openai", "openrouter", "gemini", "anthropic", "xai", "ollama"}
	statuses := make([]TranslationAIProviderStatus, 0, len(providers))
	for _, providerType := range providers {
		statuses = append(statuses, TranslationAIProviderStatus{
			ProviderType: providerType,
			DisplayName:  translationProviderDisplayName(providerType),
			DefaultModel: translationDefaultModel(providerType),
			Configured:   translationProviderConfigured(providerType),
		})
	}
	return statuses, nil
}

func (s *dbTranslationService) AITranslateMissing(ctx context.Context, req AITranslateMissingRequest) (*AITranslateMissingResult, error) {
	targetLocale := strings.TrimSpace(req.TargetLocale)
	if targetLocale == "" {
		return nil, errors.New("targetLocale is required")
	}
	sourceLocale := strings.TrimSpace(req.SourceLocale)
	if sourceLocale == "" {
		sourceLocale = "fr"
	}
	if sourceLocale == targetLocale {
		return nil, errors.New("sourceLocale and targetLocale must be different")
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}

	providerType := strings.TrimSpace(req.Provider)
	if providerType == "" {
		providerType = translationDefaultProvider()
	}
	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = translationDefaultModel(providerType)
	}

	type translationCandidate struct {
		KeyID       uint
		Domain      string
		Key         string
		Description string
		SourceValue string
		TargetValue string
	}
	var candidates []translationCandidate
	query := s.db.WithContext(ctx).
		Table("nexus_translation_keys as k").
		Select("k.id as key_id, d.code as domain, k.`key` as `key`, k.description as description, sv.value as source_value, COALESCE(tv.value, '') as target_value").
		Joins("JOIN nexus_translation_domains as d ON d.id = k.domain_id").
		Joins("JOIN nexus_translation_values as sv ON sv.key_id = k.id AND sv.locale = ?", sourceLocale).
		Joins("LEFT JOIN nexus_translation_values as tv ON tv.key_id = k.id AND tv.locale = ?", targetLocale).
		Where("sv.value <> ''").
		Where("(tv.id IS NULL OR tv.value = '')")

	if len(req.Domains) > 0 {
		query = query.Where("d.code IN ?", cleanStringList(req.Domains))
	}
	if len(req.Keys) > 0 {
		query = query.Where("k.`key` IN ?", cleanStringList(req.Keys))
	}
	if err := query.Order("d.code ASC, k.`key` ASC").Limit(limit).Scan(&candidates).Error; err != nil {
		return nil, err
	}

	adapter := tribunaladapters.NewAIProviderAdapter(translationProviderEnvKey).WithTimeout(90 * time.Second)
	result := &AITranslateMissingResult{
		Provider:     tribunaladapters.NormalizeProviderType(providerType),
		Model:        model,
		SourceLocale: sourceLocale,
		TargetLocale: targetLocale,
		Items:        make([]AITranslatedValueItem, 0, len(candidates)),
	}

	for _, candidate := range candidates {
		item := AITranslatedValueItem{
			Domain: candidate.Domain,
			Key:    candidate.Key,
			Source: candidate.SourceValue,
		}
		translated, err := adapter.Generate(ctx, tribunaladapters.GenerateRequest{
			ProviderType:  providerType,
			Model:         model,
			APIKey:        req.APIKey,
			LocalEndpoint: req.LocalEndpoint,
			SystemPrompt:  translationAISystemPrompt(),
			Prompt:        translationAIUserPrompt(sourceLocale, targetLocale, candidate.Key, candidate.Description, candidate.SourceValue),
		})
		if err != nil {
			item.Error = err.Error()
			result.Errors++
			result.Items = append(result.Items, item)
			continue
		}
		value := cleanAITranslatedText(translated.Text)
		if value == "" {
			item.Error = "provider returned empty translation"
			result.Errors++
			result.Items = append(result.Items, item)
			continue
		}
		if err := upsertTranslationRow(s.db.WithContext(ctx), models.TranslationImportRow{
			Domain: candidate.Domain,
			Key:    candidate.Key,
			Locale: targetLocale,
			Value:  value,
			Status: "ok",
		}); err != nil {
			item.Error = err.Error()
			result.Errors++
			result.Items = append(result.Items, item)
			continue
		}
		item.Value = value
		result.Translated++
		result.Items = append(result.Items, item)
	}
	return result, nil
}

func upsertTranslationRow(tx *gorm.DB, row models.TranslationImportRow) error {
	if !isRetainedTranslation(row.Domain, row.Key) {
		return nil
	}

	var domain models.TranslationDomain
	if err := tx.Where("code = ?", row.Domain).
		FirstOrCreate(&domain, models.TranslationDomain{Code: row.Domain, Name: row.Domain}).Error; err != nil {
		return err
	}

	var key models.TranslationKey
	if err := tx.Where("domain_id = ? AND `key` = ?", domain.ID, row.Key).
		FirstOrCreate(&key, models.TranslationKey{DomainID: domain.ID, Key: row.Key}).Error; err != nil {
		return err
	}

	value := models.TranslationValue{
		KeyID:  key.ID,
		Locale: row.Locale,
		Value:  row.Value,
	}
	return tx.Where("key_id = ? AND locale = ?", key.ID, row.Locale).
		Assign(value).
		FirstOrCreate(&value).Error
}

func translationAISystemPrompt() string {
	return "Tu es un outil de localisation produit. Traduis uniquement la valeur demandee, sans markdown, sans guillemets et sans commentaire. Preserve exactement les placeholders entre accolades comme {count}, {name}, {xp}. Preserve les acronymes produit comme IA, Nexus, Battle, Sandbox, Tribunal et Co-op si pertinent."
}

func translationAIUserPrompt(sourceLocale, targetLocale, key, description, value string) string {
	return "Source locale: " + sourceLocale +
		"\nTarget locale: " + targetLocale +
		"\nKey: " + key +
		"\nDescription: " + strings.TrimSpace(description) +
		"\nText:\n" + value
}

func cleanAITranslatedText(value string) string {
	cleaned := strings.TrimSpace(value)
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)
	if len(cleaned) >= 2 {
		if (strings.HasPrefix(cleaned, "\"") && strings.HasSuffix(cleaned, "\"")) ||
			(strings.HasPrefix(cleaned, "'") && strings.HasSuffix(cleaned, "'")) {
			cleaned = strings.TrimSpace(cleaned[1 : len(cleaned)-1])
		}
	}
	return cleaned
}

func cleanStringList(values []string) []string {
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	return cleaned
}

func translationDefaultProvider() string {
	for _, value := range []string{
		os.Getenv("TRANSLATION_AI_PROVIDER"),
		os.Getenv("WORLD_AI_PRIMARY_PROVIDER"),
		os.Getenv("TRIBUNAL_AI_PROVIDER"),
	} {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return "mistral"
}

func translationDefaultModel(providerType string) string {
	_, modelEnv := translationProviderEnvNames(providerType)
	if model := strings.TrimSpace(os.Getenv(modelEnv)); model != "" {
		return model
	}
	if model := strings.TrimSpace(os.Getenv("TRANSLATION_AI_MODEL")); model != "" {
		return model
	}
	return tribunaladapters.DefaultModelForProvider(providerType)
}

func translationProviderEnvKey(providerType string) string {
	keyEnv, _ := translationProviderEnvNames(providerType)
	return strings.TrimSpace(os.Getenv(keyEnv))
}

func translationProviderConfigured(providerType string) bool {
	if tribunaladapters.IsLocalProvider(providerType) {
		return strings.TrimSpace(os.Getenv("TRANSLATION_AI_LOCAL_ENDPOINT")) != "" ||
			strings.TrimSpace(os.Getenv("OLLAMA_ENDPOINT")) != "" ||
			strings.TrimSpace(os.Getenv("LMSTUDIO_ENDPOINT")) != ""
	}
	return translationProviderEnvKey(providerType) != ""
}

func translationProviderDisplayName(providerType string) string {
	switch tribunaladapters.NormalizeProviderType(providerType) {
	case "mistral":
		return "Mistral"
	case "openai":
		return "OpenAI"
	case "openrouter":
		return "OpenRouter"
	case "gemini", "google", "google_ai", "google-ai":
		return "Google Gemini"
	case "claude", "anthropic":
		return "Anthropic Claude"
	case "xia", "xai", "x-ai":
		return "xAI"
	case "ollama":
		return "Ollama local"
	default:
		return providerType
	}
}

func translationProviderEnvNames(providerType string) (string, string) {
	switch tribunaladapters.NormalizeProviderType(providerType) {
	case "mistral":
		return "MISTRAL_AI_KEY", "MISTRAL_AI_MODEL"
	case "claude", "anthropic":
		return "ANTHROPIC_AI_KEY", "ANTHROPIC_AI_MODEL"
	case "gemini", "google", "google_ai", "google-ai":
		return "GEMINI_AI_KEY", "GEMINI_AI_MODEL"
	case "xia", "xai", "x-ai":
		return "XAI_AI_KEY", "XAI_AI_MODEL"
	case "openrouter", "open_router":
		return "OPENROUTER_AI_KEY", "OPENROUTER_AI_MODEL"
	default:
		return "OPEN_AI_KEY", "OPEN_AI_MODEL"
	}
}
