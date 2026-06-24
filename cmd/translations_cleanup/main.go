package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"cgwm/battle/internal/db"
	translations "cgwm/battle/internal/nexus_game/translations"
)

func main() {
	seedPath := flag.String("seed", "", "optional seed JSON to clean in-place")
	deleteKeys := flag.Bool("delete", false, "delete technical keys instead of marking unused_candidate")
	flag.Parse()

	if strings.TrimSpace(*seedPath) != "" {
		if err := cleanSeedFile(*seedPath); err != nil {
			fmt.Fprintf(os.Stderr, "cleanup seed: %v\n", err)
			os.Exit(1)
		}
	}

	if flag.NArg() > 0 && flag.Arg(0) == "--db" {
		cleanupDatabase(*deleteKeys)
	}
}

func cleanSeedFile(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var payload struct {
		Language interface{}              `json:"language"`
		Locale   string                   `json:"locale"`
		FileName string                   `json:"file_name"`
		Rows     []map[string]interface{} `json:"rows"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	rows := make([]translations.ImportSeedRow, 0, len(payload.Rows))
	for _, row := range payload.Rows {
		rows = append(rows, translations.ImportSeedRow{
			Domain: fmt.Sprint(row["domain"]),
			Key:    fmt.Sprint(row["key"]),
			Value:  fmt.Sprint(row["value"]),
		})
	}
	keptRows, removed := translations.FilterSeedImportRows(rows)
	newRows := make([]map[string]interface{}, 0, len(keptRows))
	for _, row := range keptRows {
		newRows = append(newRows, map[string]interface{}{
			"domain":   row.Domain,
			"key":      row.Key,
			"locale":   "fr",
			"value":    row.Value,
			"language": "fr",
		})
	}
	payload.Rows = newRows
	out, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return err
	}
	fmt.Printf("seed cleaned: kept=%d removed=%d path=%s\n", len(keptRows), len(removed), path)
	return nil
}

func cleanupDatabase(deleteKeys bool) {
	ctx := context.Background()
	database := db.DbConnect()
	report, err := translations.CleanupTechnicalFlutterScanKeys(ctx, database, deleteKeys)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cleanup db: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("cleanup db marked=%d deleted=%d deleted_values=%d\n",
		len(report.MarkedUnused), len(report.DeletedKeys), report.DeletedValues)
}