"use client";

import { useEffect, useState } from "react";
import { AdminShell } from "../../../components/AdminShell";
import { ErrorState, LoadingState } from "../../../components/LoadState";
import { formatDate } from "../../../components/api";
import type { TranslationImport } from "../../../types";

export default function LogsPage() {
  const [imports, setImports] = useState<TranslationImport[]>([]);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetch("/admin/api/translations/imports", { credentials: "same-origin" })
      .then((r) => r.json())
      .then((d) => setImports(d?.imports || d || []))
      .catch((e: Error) => setError(e.message));
  }, []);

  return (
    <AdminShell title="Logs imports traductions" description="Historique des imports (preview/commit). Appels Go uniquement.">
      {error ? <ErrorState message={error} /> : null}
      {!imports.length && !error ? <LoadingState /> : null}

      {imports.length > 0 && (
        <section className="panel">
          <table className="data-table">
            <thead>
              <tr>
                <th>ID</th>
                <th>Fichier</th>
                <th>Status</th>
                <th>Rows</th>
                <th>Date</th>
              </tr>
            </thead>
            <tbody>
              {imports.map((i) => (
                <tr key={i.ID}>
                  <td>{i.ID}</td>
                  <td>{i.FileName}</td>
                  <td><span className={`status ${i.Status}`}>{i.Status}</span></td>
                  <td>{i.RowCount}</td>
                  <td>{formatDate(i.CreatedAt)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </section>
      )}
    </AdminShell>
  );
}
