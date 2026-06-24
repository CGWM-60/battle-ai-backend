package models

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

// Translation models for Nexus Games i18n (server-driven translations)
// Tables use "nexus_" prefix as per spec.

// TranslationDomain groups keys by domain (e.g. "nexus_game", "common", "nexus_game_city")
type TranslationDomain struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	Code        string `gorm:"size:64;uniqueIndex"` // e.g. "nexus_game"
	Name        string `gorm:"size:120"`
	Description string `gorm:"type:text"`
}

func (TranslationDomain) TableName() string { return "nexus_translation_domains" }

// TranslationKey is a translatable string identifier within a domain.
type TranslationKey struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	DomainID uint              `gorm:"index"`
	Domain   TranslationDomain `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	Key          string `gorm:"size:255;uniqueIndex"` // e.g. "nexus_game.city.population"
	Description  string `gorm:"type:text"`
	SourceFile   string `gorm:"size:512"`
	SourceLine   int
	SourceModule string `gorm:"size:64"`
	Kind         string `gorm:"size:32"`  // ui, prompt, error, tooltip, button, title, label
	Status       string `gorm:"size:32"`  // active, unused_candidate
	Reviewed     bool   `gorm:"default:false"`
	ImportSource string `gorm:"size:64"` // flutter_scan, admin, seed
	TagsJSON     string `gorm:"type:json;default:'[]'"`
}

func (k *TranslationKey) BeforeCreate(tx *gorm.DB) error {
	if strings.TrimSpace(k.TagsJSON) == "" {
		k.TagsJSON = "[]"
	}
	return nil
}

func (k *TranslationKey) BeforeSave(tx *gorm.DB) error {
	if strings.TrimSpace(k.TagsJSON) == "" {
		k.TagsJSON = "[]"
	}
	return nil
}

func (k *TranslationKey) BeforeUpdate(tx *gorm.DB) error {
	if strings.TrimSpace(k.TagsJSON) == "" {
		k.TagsJSON = "[]"
	}
	return nil
}

func (TranslationKey) TableName() string { return "nexus_translation_keys" }

// TranslationValue is the actual translated text for a key + locale.
type TranslationValue struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	KeyID uint           `gorm:"index"`
	Key   TranslationKey `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	Locale string `gorm:"size:10;index"` // "fr", "en", ...
	Value  string `gorm:"type:text"`

	Reviewed           bool   `gorm:"default:false"`
	MachineTranslated  bool   `gorm:"default:false"`
	Source             string `gorm:"size:64"` // seed, flutter_scan, admin, ai
	UpdatedBy          string `gorm:"size:120"`
}

func (TranslationValue) TableName() string { return "nexus_translation_values" }

// TranslationBatch groups a set of translations for import/apply (e.g. one admin batch update).
type TranslationBatch struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	Locale    string `gorm:"size:10;index"`
	Status    string `gorm:"size:32;index"` // pending, applied, failed
	CreatedBy string `gorm:"size:120"`
	Note      string `gorm:"type:text"`
}

func (TranslationBatch) TableName() string { return "nexus_translation_batches" }

// TranslationImport tracks an import operation (file upload preview + commit).
type TranslationImport struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	BatchID *uint             `gorm:"index"`
	Batch   *TranslationBatch `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`

	FileName string `gorm:"size:255"`
	Status   string `gorm:"size:32;index"` // preview, committed, failed
	RowCount int
}

func (TranslationImport) TableName() string { return "nexus_translation_imports" }

// TranslationImportRow is one row from an import file.
type TranslationImportRow struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time

	ImportID uint              `gorm:"index"`
	Import   TranslationImport `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	Domain string `gorm:"size:64"`
	Key    string `gorm:"size:255"`
	Locale string `gorm:"size:10"`
	Value  string `gorm:"type:text"`
	Status string `gorm:"size:32;index"` // ok, error, skipped
	Error  string `gorm:"type:text"`
}

func (TranslationImportRow) TableName() string { return "nexus_translation_import_rows" }

// TranslationImportError logs errors during import.
type TranslationImportError struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time

	ImportID *uint              `gorm:"index"`
	Import   *TranslationImport `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`

	RowID   *uint  `gorm:"index"`
	Message string `gorm:"type:text"`
}

func (TranslationImportError) TableName() string { return "nexus_translation_import_errors" }

// TranslationMissingLog records when the app requests a missing translation key.
type TranslationMissingLog struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time

	Key    string `gorm:"size:255;index"`
	Locale string `gorm:"size:10;index"`
	Count  int    `gorm:"default:1"`
}

func (TranslationMissingLog) TableName() string { return "nexus_translation_missing_logs" }

// UserLocalePreference stores per-user locale choice.
type UserLocalePreference struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	UserID uint   `gorm:"uniqueIndex"`
	Locale string `gorm:"size:10"`
}

func (UserLocalePreference) TableName() string { return "nexus_user_locale_preferences" }

// TranslationTag is a reusable label for grouping translation keys.
type TranslationTag struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	Code string `gorm:"size:64;uniqueIndex"`
	Name string `gorm:"size:120"`
}

func (TranslationTag) TableName() string { return "nexus_translation_tags" }

// TranslationKeyTag links keys to tags (many-to-many).
type TranslationKeyTag struct {
	KeyID uint `gorm:"primaryKey;autoIncrement:false"`
	TagID uint `gorm:"primaryKey;autoIncrement:false"`
}

func (TranslationKeyTag) TableName() string { return "nexus_translation_key_tags" }
