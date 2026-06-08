"use client";

import { useState } from "react";
import { AdminShell } from "../../../components/AdminShell";
import { ErrorState } from "../../../components/LoadState";
import type { TranslationImportRow } from "../../../types";

export default function ImportPage() {
  const [jsonText, setJsonText] = useState("");
  const [preview, setPreview] = useState<TranslationImportRow[] | null>(null);
  const [importId, setImportId] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const doPreview = async () => {
    setError(null);
    setBusy(true);
    try {
      let rows: TranslationImportRow[];
      try {
        rows = JSON.parse(jsonText);
      } catch {
        throw new Error("JSON invalide");
      }
      const res = await fetch("/admin/api/translations/import/preview", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "same-origin",
        body: JSON.stringify(rows),
      });
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      setPreview(data.preview || data);
      // In real, the preview may return an import ID, here we simulate
      setImportId(1);
    } catch (e: any) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  };

  const doCommit = async () => {
    if (!importId) return;
    setBusy(true);
    setError(null);
    try {
      const res = await fetch("/admin/api/translations/import/commit", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "same-origin",
        body: JSON.stringify({ import_id: importId }),
      });
      if (!res.ok) throw new Error(await res.text());
      alert("Import commit OK (côté Go).");
      setPreview(null);
      setJsonText("");
      setImportId(null);
    } catch (e: any) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  };

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
          placeholder='[{"domain":"nexus_game","key":"...","locale":"fr","value":"..."}]'
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
          <button onClick={doCommit} disabled={busy || !importId}>
            Commit import (appel Go)
          </button>
          <p style={{ fontSize: "0.8em" }}>Le commit est fait côté Go. Next.js n'écrit rien.</p>
        </section>
      )}
    </AdminShell>
  );
}
