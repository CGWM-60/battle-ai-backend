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
	filePath := flag.String("file", "", "import JSON file path (required)")
	source := flag.String("source", "seed", "import source: seed|flutter_scan")
	defaultLocale := flag.String("default-locale", "fr", "default locale for flutter_scan import")
	dryRun := flag.Bool("dry-run", false, "preview only, do not persist")
	reportPath := flag.String("report", "", "optional path to write import report JSON")
	flag.Parse()

	if strings.TrimSpace(*filePath) == "" {
		fmt.Fprintln(os.Stderr, "--file is required")
		os.Exit(1)
	}

	raw, err := os.ReadFile(*filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read import file: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	switch strings.TrimSpace(*source) {
	case "flutter_scan":
		runFlutterScan(ctx, raw, *defaultLocale, *dryRun, *reportPath)
	default:
		runSeedImport(ctx, raw, *dryRun)
	}
}

func runSeedImport(ctx context.Context, raw []byte, dryRun bool) {
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
	preview, err := service.PreviewImport(ctx, payload.Rows)
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
	if dryRun {
		fmt.Println("dry-run: import not committed")
		return
	}

	database := db.DbConnect()
	service = translations.NewTranslationService(database)
	imp, err := service.CommitImportRows(ctx, preview, payload.FileName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "commit import: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("committed import_id=%d rows=%d status=%s\n", imp.ID, imp.RowCount, imp.Status)
}

func runFlutterScan(ctx context.Context, raw []byte, defaultLocale string, dryRun bool, reportPath string) {
	entries, err := translations.ParseFlutterScanBytes(raw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse flutter_scan file: %v\n", err)
		os.Exit(1)
	}
	if len(entries) == 0 {
		fmt.Fprintln(os.Stderr, "flutter_scan file contains no entries")
		os.Exit(1)
	}

	fmt.Printf("flutter_scan entries=%d locale=%s dry_run=%v\n", len(entries), defaultLocale, dryRun)
	if dryRun {
		fmt.Println("dry-run: flutter_scan import not committed")
		writeReport(reportPath, map[string]interface{}{
			"source":        "flutter_scan",
			"entries":       len(entries),
			"defaultLocale": defaultLocale,
			"dryRun":        true,
		})
		return
	}

	database := db.DbConnect()
	service := translations.NewTranslationService(database)
	report, err := service.ImportFlutterScan(ctx, entries, defaultLocale)
	if err != nil {
		fmt.Fprintf(os.Stderr, "import flutter_scan: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf(
		"created_keys=%d updated_keys=%d created_tags=%d created_namespaces=%d skipped=%d\n",
		len(report.CreatedKeys),
		len(report.UpdatedKeys),
		len(report.CreatedTags),
		len(report.CreatedNamespaces),
		len(report.SkippedEntries),
	)
	writeReport(reportPath, report)
}

func writeReport(path string, payload interface{}) {
	if strings.TrimSpace(path) == "" {
		return
	}
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal report: %v\n", err)
		return
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write report: %v\n", err)
	}
}