"use client";

import { useState } from "react";
import { AdminShell } from "../../../components/AdminShell";
import { ErrorState } from "../../../components/LoadState";
import type { TranslationImportPayload, TranslationImportRow } from "../../../types";

export default function ImportPage() {
  const [jsonText, setJsonText] = useState("");
  const [preview, setPreview] = useState<TranslationImportRow[] | null>(null);
  const [payload, setPayload] = useState<TranslationImportPayload | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const doPreview = async () => {
    setError(null);
    setBusy(true);
    try {
      let parsedPayload: TranslationImportPayload;
      try {
        parsedPayload = normalizeImportPayload(JSON.parse(jsonText));
      } catch {
        throw new Error("JSON invalide");
      }
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
      } catch (e) {
        console.error('[admin import preview] Bad JSON body prefix:', text.substring(0, 300));
        throw new Error('Invalid JSON from preview endpoint. See console.');
      }
      setPreview(data.preview || data);
      setPayload({ ...parsedPayload, rows: data.preview || parsedPayload.rows });
    } catch (e: any) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  };

  const doCommit = async () => {
    if (!preview) return;
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
          file_name: payload?.file_name || "admin-import.json",
        }),
      });
      if (!res.ok) throw new Error(await res.text());
      const text = await res.text();
      try {
        JSON.parse(text); // just validate it's JSON, we don't need the value
      } catch (e) {
        console.error('[admin import commit] Bad JSON body prefix:', text.substring(0, 300));
        throw new Error('Invalid JSON from commit endpoint. See console.');
      }
      alert("Import commit OK (côté Go).");
      setPreview(null);
      setPayload(null);
      setJsonText("");
    } catch (e: any) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  };

  const hasPreviewErrors = preview?.some((row) => row.Status === "error") ?? false;

  const handleFile = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = (ev) => setJsonText(String(ev.target?.result || ""));
    reader.readAsText(file);
  };

  return (
    <AdminShell
      title="Import traductions"
      description="Preview et commit via les APIs Go uniquement. Aucun traitement d'import ici."
    >
      {error ? <ErrorState message={error} /> : null}

      <section className="panel">
        <h2>Charger le fichier JSON</h2>
        <input type="file" accept=".json" onChange={handleFile} />
        <textarea
          value={jsonText}
          onChange={(e) => setJsonText(e.target.value)}
          placeholder='{"language":{"code":"fr"},"rows":[{"domain":"nexus_game","key":"...","locale":"fr","value":"..."}]}'
          rows={10}
          style={{ width: "100%", fontFamily: "monospace", marginTop: 8 }}
        />
        <button onClick={doPreview} disabled={busy || !jsonText}>
          {busy ? "..." : "Preview (appel Go)"}
        </button>
      </section>

      {preview && (
        <section className="panel">
          <h2>Preview (depuis Go)</h2>
          {payload?.language?.code ? <p>Langue: {payload.language.native_name || payload.language.name || payload.language.code}</p> : null}
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
              {preview.map((r, i) => (
                <tr key={i}>
                  <td>{r.Domain}</td>
                  <td><code>{r.Key}</code></td>
                  <td>{r.Locale}</td>
                  <td>{r.Value}</td>
                  <td className={r.Status === "ok" ? "good" : "bad"}>{r.Status || "ok"} {r.Error}</td>
                </tr>
              ))}
            </tbody>
          </table>
          <button onClick={doCommit} disabled={busy || hasPreviewErrors}>
            Commit import (appel Go)
          </button>
          <p style={{ fontSize: "0.8em" }}>Le commit est fait côté Go. Next.js n'écrit rien.</p>
        </section>
      )}
    </AdminShell>
  );
}

function normalizeImportPayload(parsed: unknown): TranslationImportPayload {
  if (Array.isArray(parsed)) {
    return { file_name: "admin-import.json", rows: parsed as TranslationImportRow[] };
  }
  if (parsed && typeof parsed === "object" && Array.isArray((parsed as { rows?: unknown }).rows)) {
    const data = parsed as TranslationImportPayload;
    return {
      language: data.language,
      locale: data.locale || data.language?.code,
      file_name: data.file_name || "admin-import.json",
      rows: data.rows,
    };
  }
  throw new Error("JSON invalide: tableau de lignes ou objet { language, rows } attendu");
}
