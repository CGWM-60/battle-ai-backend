package translations

import (
	"encoding/json"
	"errors"
	"strings"

	"cgwm/battle/internal/models"
)

type ImportLanguage struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	NativeName string `json:"native_name"`
	Default    bool   `json:"default"`
}

type ImportPayload struct {
	ImportID uint                          `json:"import_id"`
	Language ImportLanguage                `json:"language"`
	Locale   string                        `json:"locale"`
	FileName string                        `json:"file_name"`
	Rows     []models.TranslationImportRow `json:"rows"`
}

type importPayloadJSON struct {
	ImportID uint            `json:"import_id"`
	Language ImportLanguage  `json:"language"`
	Locale   string          `json:"locale"`
	FileName string          `json:"file_name"`
	Rows     []importRowJSON `json:"rows"`
}

type importRowJSON struct {
	Domain   string `json:"domain"`
	Key      string `json:"key"`
	Locale   string `json:"locale"`
	Language string `json:"language"`
	Value    string `json:"value"`
	Status   string `json:"status"`
	Error    string `json:"error"`
}

func ParseImportPayloadBytes(raw []byte) (*ImportPayload, error) {
	if len(strings.TrimSpace(string(raw))) == 0 {
		return nil, errors.New("import payload is empty")
	}

	var rows []importRowJSON
	if err := json.Unmarshal(raw, &rows); err == nil {
		payload := &ImportPayload{
			Locale:   inferRowsLocale(rows),
			FileName: "admin-import.json",
			Rows:     normalizeImportRows(rows, ""),
		}
		if payload.Locale != "" {
			payload.Language = ImportLanguage{Code: payload.Locale}
		}
		return payload, nil
	}

	var data importPayloadJSON
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}

	locale := strings.TrimSpace(data.Locale)
	if locale == "" {
		locale = strings.TrimSpace(data.Language.Code)
	}

	return &ImportPayload{
		ImportID: data.ImportID,
		Language: data.Language,
		Locale:   locale,
		FileName: data.FileName,
		Rows:     normalizeImportRows(data.Rows, locale),
	}, nil
}

func normalizeImportRows(rows []importRowJSON, fallbackLocale string) []models.TranslationImportRow {
	out := make([]models.TranslationImportRow, 0, len(rows))
	for _, row := range rows {
		locale := strings.TrimSpace(row.Locale)
		if locale == "" {
			locale = strings.TrimSpace(row.Language)
		}
		if locale == "" {
			locale = fallbackLocale
		}

		out = append(out, models.TranslationImportRow{
			Domain: strings.TrimSpace(row.Domain),
			Key:    strings.TrimSpace(row.Key),
			Locale: locale,
			Value:  row.Value,
			Status: strings.TrimSpace(row.Status),
			Error:  row.Error,
		})
	}
	return out
}

func inferRowsLocale(rows []importRowJSON) string {
	for _, row := range rows {
		if locale := strings.TrimSpace(row.Locale); locale != "" {
			return locale
		}
		if language := strings.TrimSpace(row.Language); language != "" {
			return language
		}
	}
	return ""
}
