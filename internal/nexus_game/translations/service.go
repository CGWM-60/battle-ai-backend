package translations

import (
	"context"
	"errors"
	"strings"
	"time"

	"cgwm/battle/internal/models"
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
}

// dbTranslationService implémente TranslationService avec GORM.
type dbTranslationService struct {
	db *gorm.DB
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

	if len(m) == 0 {
		for key, value := range DefaultFrenchFallback {
			m[key] = value
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
	if len(m) == 0 && domain == "nexus_game" {
		for key, value := range DefaultFrenchFallback {
			if strings.HasPrefix(key, "nexus_game.") {
				m[key] = value
			}
		}
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
	err := s.db.Where("key = ? AND locale = ?", key, locale).First(&log).Error
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

// DefaultFrenchFallback contient un minimum vital pour que l'app ne soit jamais vide.
// Le vrai contenu vient de la DB / admin.
var DefaultFrenchFallback = map[string]string{
	"common.app.title":           "NEXUS GAMES",
	"common.button.confirm":      "CONFIRMER",
	"common.button.cancel":       "ANNULER",
	"nexus_game.city.population": "Population",
	"nexus_game.city.moral":      "Moral",
	"nexus_game.ai.plan":         "Plan IA",
	"nexus_game.ai.ask":          "Demander à mon IA",
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
			} else if rows[i].Status == "" {
				rows[i].Status = "ok"
			}

			if err := tx.Create(&rows[i]).Error; err != nil {
				return err
			}
			if rows[i].Status == "error" {
				return errors.New("import contains invalid rows")
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

func upsertTranslationRow(tx *gorm.DB, row models.TranslationImportRow) error {
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
