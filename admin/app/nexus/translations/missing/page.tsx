"use client";

import { useEffect, useState } from "react";
import { AdminShell } from "../../../components/AdminShell";
import { ErrorState, LoadingState } from "../../../components/LoadState";
import { formatDate, loadAdminData } from "../../../components/api";
import type { TranslationMissingLog } from "../../../types";

export default function MissingPage() {
  const [logs, setLogs] = useState<TranslationMissingLog[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadAdminData<any>("translations/missing")
      .then((d) => setLogs(d?.missing || d || []))
      .catch((e: Error) => setError(e.message))
      .finally(() => setLoading(false));
  }, []);

  return (
    <AdminShell title="Clés manquantes" description="Logs des clés demandées par l'app mais absentes (depuis /api/translations/missing).">
      {error ? <ErrorState message={error} /> : null}
      {loading && !error ? <LoadingState /> : null}

      {!loading && !error && (
        <section className="panel">
          {logs.length ? (
            <table className="data-table">
              <thead>
                <tr>
                  <th>Clé</th>
                  <th>Locale</th>
                  <th>Count</th>
                  <th>Dernier signalement</th>
                </tr>
              </thead>
              <tbody>
                {logs.map((l) => (
                  <tr key={l.ID}>
                    <td><code>{l.Key}</code></td>
                    <td>{l.Locale}</td>
                    <td>{l.Count}</td>
                    <td>{formatDate(l.CreatedAt)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : (
            <p className="muted">Aucune clé manquante signalée.</p>
          )}
        </section>
      )}
    </AdminShell>
  );
}
