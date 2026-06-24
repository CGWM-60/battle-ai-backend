"use client";

import { useMemo, useState, type ChangeEvent } from "react";
import { AdminShell } from "../../../components/AdminShell";
import { ErrorState } from "../../../components/LoadState";
import type { TranslationImportPayload, TranslationImportRow } from "../../../types";

type ImportFormat = "csv" | "json";

const CSV_RESERVED_HEADERS = new Set(["domain", "key", "description"]);

export default function ImportPage() {
  const [sourceText, setSourceText] = useState("");
  const [format, setFormat] = useState<ImportFormat>("csv");
  const [fileName, setFileName] = useState("admin-import.csv");
  const [preview, setPreview] = useState<TranslationImportRow[] | null>(null);
  const [payload, setPayload] = useState<TranslationImportPayload | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [showConfirm, setShowConfirm] = useState(false);

  const previewSummary = useMemo(() => summarizePreview(preview || []), [preview]);
  const hasPreviewErrors = previewSummary.errors > 0;

  const doPreview = async () => {
    setError(null);
    setBusy(true);
    setShowConfirm(false);
    try {
      const parsedPayload = normalizeImportPayload(sourceText, format, fileName);
      const res = await fetch("/admin/api/translations/import/preview", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "same-origin",
        body: JSON.stringify(parsedPayload),
      });
      if (!res.ok) throw new Error(await res.text());
      const text = await res.text();
      let data: any;
      try {
        data = JSON.parse(text);
      } catch {
        console.error("[admin import preview] Bad JSON body prefix:", text.substring(0, 300));
        throw new Error("Invalid JSON from preview endpoint. See console.");
      }
      const previewRows = data.preview || data;
      setPreview(previewRows);
      setPayload({ ...parsedPayload, rows: previewRows });
    } catch (e: any) {
      setPreview(null);
      setPayload(null);
      setError(e.message);
    } finally {
      setBusy(false);
    }
  };

  const doCommit = async () => {
    if (!preview || !payload || hasPreviewErrors) return;
    setBusy(true);
    setError(null);
    try {
      const res = await fetch("/admin/api/translations/import/commit", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "same-origin",
        body: JSON.stringify({
          ...payload,
          rows: preview,
          file_name: payload.file_name || fileName || "admin-import.csv",
        }),
      });
      if (!res.ok) throw new Error(await res.text());
      const text = await res.text();
      try {
        JSON.parse(text);
      } catch {
        console.error("[admin import commit] Bad JSON body prefix:", text.substring(0, 300));
        throw new Error("Invalid JSON from commit endpoint. See console.");
      }
      setShowConfirm(false);
      setPreview(null);
      setPayload(null);
      setSourceText("");
      setFileName(format === "csv" ? "admin-import.csv" : "admin-import.json");
      alert("Import commit OK côté Go.");
    } catch (e: any) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  };

  const handleFile = (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;
    const nextFormat = file.name.toLowerCase().endsWith(".json") ? "json" : "csv";
    setFormat(nextFormat);
    setFileName(file.name);
    setPreview(null);
    setPayload(null);
    setShowConfirm(false);
    const reader = new FileReader();
    reader.onload = (ev) => setSourceText(String(ev.target?.result || ""));
    reader.readAsText(file);
  };

  const openConfirm = () => {
    if (!preview || hasPreviewErrors) return;
    setShowConfirm(true);
  };

  return (
    <AdminShell
      title="Import traductions"
      description="Import JSON ou CSV avec preview Go, popin de contrôle, puis validation manuelle."
    >
      {error ? <ErrorState message={error} /> : null}

      <section className="panel">
        <h2>Charger le fichier</h2>
        <p className="hint">
          CSV attendu: <code>domain,key,description,fr,en-US,en-GB,...</code>. Les cellules vides sont ignorées pour éviter
          d'écraser une traduction avec du vide.
        </p>
        <div className="translation-import-controls">
          <label>
            Format
            <select value={format} onChange={(event) => setFormat(event.target.value as ImportFormat)}>
              <option value="csv">CSV colonnes par langue</option>
              <option value="json">JSON lignes d'import</option>
            </select>
          </label>
          <label>
            Nom du fichier
            <input value={fileName} onChange={(event) => setFileName(event.target.value)} />
          </label>
          <label>
            Fichier
            <input type="file" accept=".csv,.json,text/csv,application/json" onChange={handleFile} />
          </label>
        </div>
        <textarea
          value={sourceText}
          onChange={(event) => {
            setSourceText(event.target.value);
            setPreview(null);
            setPayload(null);
            setShowConfirm(false);
          }}
          placeholder={
            format === "csv"
              ? "domain,key,description,fr,en-US\ncommon,common.app.title,Titre,NEXUS GAMES,NEXUS GAMES"
              : '{"language":{"code":"fr"},"rows":[{"domain":"common","key":"...","locale":"fr","value":"..."}]}'
          }
          rows={12}
          style={{ width: "100%", fontFamily: "monospace", marginTop: 8 }}
        />
        <div className="translation-actions">
          <button className="primary" onClick={doPreview} disabled={busy || !sourceText.trim()}>
            {busy ? "..." : "Prévisualiser l'import"}
          </button>
        </div>
      </section>

      {preview && (
        <section className="panel">
          <div className="translation-table-header">
            <h2>Preview depuis Go</h2>
            <div className="pagination-controls">
              <span>{previewSummary.total} ligne(s)</span>
              <span className="good">{previewSummary.ok} OK</span>
              <span>{previewSummary.skipped} skipped</span>
              <span className={previewSummary.errors ? "bad" : "good"}>{previewSummary.errors} erreur(s)</span>
            </div>
          </div>
          <p className="hint">
            Locales détectées: {previewSummary.locales.join(", ") || "-"} · Domaines: {previewSummary.domains.join(", ") || "-"}
          </p>
          <div className="table-wrap translation-preview-wrap">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Domaine</th>
                  <th>Clé</th>
                  <th>Locale</th>
                  <th>Valeur</th>
                  <th>Status</th>
                </tr>
              </thead>
              <tbody>
                {preview.map((row, index) => (
                  <tr key={`${row.Domain}:${row.Key}:${row.Locale}:${index}`}>
                    <td>{row.Domain}</td>
                    <td><code>{row.Key}</code></td>
                    <td>{row.Locale}</td>
                    <td>{row.Value}</td>
                    <td className={row.Status === "error" ? "bad" : "good"}>{row.Status || "ok"} {row.Error}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <div className="translation-actions">
            <button className="primary" onClick={openConfirm} disabled={busy || hasPreviewErrors}>
              Contrôler et valider l'import
            </button>
            <button className="secondary" onClick={() => setPreview(null)} disabled={busy}>
              Annuler la preview
            </button>
          </div>
          {hasPreviewErrors ? <p className="alert error">Import bloqué: corrige les lignes en erreur puis relance la preview.</p> : null}
        </section>
      )}

      {showConfirm && preview && (
        <div className="modal-backdrop" role="presentation">
          <section className="panel json-modal translation-confirm-modal" role="dialog" aria-modal="true" aria-labelledby="confirm-import-title">
            <h2 id="confirm-import-title">Contrôle avant import</h2>
            <p className="hint">
              Cette action va écrire les traductions dans la base. Vérifie les volumes avant de confirmer.
            </p>
            <div className="translation-confirm-grid">
              <div><span>Total</span><strong>{previewSummary.total}</strong></div>
              <div><span>OK</span><strong>{previewSummary.ok}</strong></div>
              <div><span>Ignorées</span><strong>{previewSummary.skipped}</strong></div>
              <div><span>Locales</span><strong>{previewSummary.locales.length}</strong></div>
            </div>
            <dl className="translation-confirm-list">
              <dt>Fichier</dt>
              <dd>{payload?.file_name || fileName}</dd>
              <dt>Locales</dt>
              <dd>{previewSummary.locales.join(", ") || "-"}</dd>
              <dt>Domaines</dt>
              <dd>{previewSummary.domains.join(", ") || "-"}</dd>
            </dl>
            <div className="translation-actions">
              <button className="primary" onClick={doCommit} disabled={busy || hasPreviewErrors}>
                {busy ? "Import..." : "Valider l'import"}
              </button>
              <button className="secondary" onClick={() => setShowConfirm(false)} disabled={busy}>
                Retour à la preview
              </button>
            </div>
          </section>
        </div>
      )}
    </AdminShell>
  );
}

function normalizeImportPayload(sourceText: string, format: ImportFormat, fileName: string): TranslationImportPayload {
  if (format === "csv") {
    return csvToImportPayload(sourceText, fileName || "admin-import.csv");
  }

  try {
    return normalizeJsonImportPayload(JSON.parse(sourceText), fileName || "admin-import.json");
  } catch {
    throw new Error("JSON invalide");
  }
}

function normalizeJsonImportPayload(parsed: unknown, fileName: string): TranslationImportPayload {
  if (Array.isArray(parsed)) {
    return { file_name: fileName, rows: parsed as TranslationImportRow[] };
  }
  if (parsed && typeof parsed === "object" && Array.isArray((parsed as { rows?: unknown }).rows)) {
    const data = parsed as TranslationImportPayload;
    return {
      language: data.language,
      locale: data.locale || data.language?.code,
      file_name: data.file_name || fileName,
      rows: data.rows,
    };
  }
  throw new Error("JSON invalide: tableau de lignes ou objet { language, rows } attendu");
}

function csvToImportPayload(sourceText: string, fileName: string): TranslationImportPayload {
  const rows = parseCsv(sourceText).filter((row) => row.some((cell) => cell.trim()));
  if (rows.length < 2) {
    throw new Error("CSV invalide: en-tête et au moins une ligne attendus");
  }

  const headers = rows[0].map((header, index) => (index === 0 ? stripBom(header) : header).trim());
  const normalizedHeaders = headers.map((header) => header.toLowerCase());
  const domainIndex = normalizedHeaders.indexOf("domain");
  const keyIndex = normalizedHeaders.indexOf("key");
  if (domainIndex < 0 || keyIndex < 0) {
    throw new Error('CSV invalide: colonnes "domain" et "key" obligatoires');
  }

  const localeColumns = headers
    .map((header, index) => ({ header, index }))
    .filter((item) => item.header && !CSV_RESERVED_HEADERS.has(item.header.toLowerCase()));
  if (!localeColumns.length) {
    throw new Error("CSV invalide: ajoute au moins une colonne de langue, ex: fr ou en-US");
  }

  const importRows: TranslationImportRow[] = [];
  rows.slice(1).forEach((row, rowIndex) => {
    const domain = (row[domainIndex] || "").trim();
    const key = (row[keyIndex] || "").trim();
    if (!domain || !key) {
      throw new Error(`CSV invalide ligne ${rowIndex + 2}: domain et key sont obligatoires`);
    }
    localeColumns.forEach(({ header, index }) => {
      const value = row[index] ?? "";
      if (!value.trim()) return;
      importRows.push({
        Domain: domain,
        Key: key,
        Locale: header,
        Value: value,
      });
    });
  });

  if (!importRows.length) {
    throw new Error("CSV invalide: aucune cellule de traduction non vide à importer");
  }

  return {
    file_name: fileName,
    rows: importRows,
  };
}

function parseCsv(sourceText: string): string[][] {
  const rows: string[][] = [];
  let row: string[] = [];
  let cell = "";
  let inQuotes = false;

  for (let index = 0; index < sourceText.length; index += 1) {
    const char = sourceText[index];
    const nextChar = sourceText[index + 1];

    if (char === '"') {
      if (inQuotes && nextChar === '"') {
        cell += '"';
        index += 1;
      } else {
        inQuotes = !inQuotes;
      }
      continue;
    }

    if (char === "," && !inQuotes) {
      row.push(cell);
      cell = "";
      continue;
    }

    if ((char === "\n" || char === "\r") && !inQuotes) {
      row.push(cell);
      rows.push(row);
      row = [];
      cell = "";
      if (char === "\r" && nextChar === "\n") {
        index += 1;
      }
      continue;
    }

    cell += char;
  }

  row.push(cell);
  rows.push(row);
  return rows;
}

function stripBom(value: string): string {
  return value.replace(/^\uFEFF/, "");
}

function summarizePreview(rows: TranslationImportRow[]) {
  const locales = new Set<string>();
  const domains = new Set<string>();
  let errors = 0;
  let skipped = 0;
  let ok = 0;

  rows.forEach((row) => {
    if (row.Locale) locales.add(row.Locale);
    if (row.Domain) domains.add(row.Domain);
    if (row.Status === "error") {
      errors += 1;
    } else if (row.Status === "skipped") {
      skipped += 1;
    } else {
      ok += 1;
    }
  });

  return {
    total: rows.length,
    ok,
    skipped,
    errors,
    locales: Array.from(locales).sort(),
    domains: Array.from(domains).sort(),
  };
}
