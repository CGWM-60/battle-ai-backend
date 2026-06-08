package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"cgwm/battle/internal/db"
	translations "cgwm/battle/internal/nexus_game/translations"
)

func main() {
	filePath := flag.String("file", "internal/nexus_game/translations/imports/NEXUS_TRANSLATIONS_INITIAL_IMPORT.fr.json", "translation import JSON file")
	dryRun := flag.Bool("dry-run", false, "preview only, do not persist")
	flag.Parse()

	raw, err := os.ReadFile(*filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read import file: %v\n", err)
		os.Exit(1)
	}

	payload, err := translations.ParseImportPayloadBytes(raw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse import file: %v\n", err)
		os.Exit(1)
	}
	if len(payload.Rows) == 0 {
		fmt.Fprintln(os.Stderr, "import file contains no rows")
		os.Exit(1)
	}

	service := translations.NewTranslationService(nil)
	preview, err := service.PreviewImport(context.Background(), payload.Rows)
	if err != nil {
		fmt.Fprintf(os.Stderr, "preview import: %v\n", err)
		os.Exit(1)
	}

	errorsCount := 0
	for _, row := range preview {
		if row.Status == "error" {
			errorsCount++
		}
	}
	fmt.Printf("language=%s rows=%d preview_errors=%d\n", payload.Locale, len(preview), errorsCount)
	if errorsCount > 0 {
		os.Exit(1)
	}
	if *dryRun {
		fmt.Println("dry-run: import not committed")
		return
	}

	database := db.DbConnect()
	service = translations.NewTranslationService(database)
	imp, err := service.CommitImportRows(context.Background(), preview, payload.FileName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "commit import: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("committed import_id=%d rows=%d status=%s\n", imp.ID, imp.RowCount, imp.Status)
}
