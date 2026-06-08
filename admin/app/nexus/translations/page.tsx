"use client";

import { useEffect, useState } from "react";
import { AdminShell } from "../../components/AdminShell";
import { ErrorState, LoadingState } from "../../components/LoadState";
import { loadAdminData } from "../../components/api";
import type { TranslationDomain, TranslationKey, TranslationValue, TranslationMissingLog } from "../../types";

export default function TranslationsPage() {
  const [domains, setDomains] = useState<TranslationDomain[]>([]);
  const [keys, setKeys] = useState<TranslationKey[]>([]);
  const [values, setValues] = useState<TranslationValue[]>([]);
  const [missing, setMissing] = useState<TranslationMissingLog[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");

  const reload = () => {
    setLoading(true);
    setError(null);
    Promise.all([
      loadAdminData<any>("translations/domains"),
      loadAdminData<any>("translations/keys"),
      loadAdminData<any>("translations/values"),
      loadAdminData<any>("translations/missing"),
    ])
      .then(([d, k, v, m]) => {
        setDomains(d?.domains || d || []);
        setKeys(k?.keys || k || []);
        setValues(v?.values || v || []);
        setMissing(m?.missing || m || []);
      })
      .catch((e: Error) => setError(e.message))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    reload();
  }, []);

  const filteredKeys = keys.filter((k) =>
    k.Key.toLowerCase().includes(search.toLowerCase()) ||
    (k.Description || "").toLowerCase().includes(search.toLowerCase())
  );

  return (
    <AdminShell
      title="Traductions Nexus"
      description="Gestion des traductions serveur (domaines, clés, valeurs, imports, missing). Tout passe par les APIs Go."
    >
      {error ? <ErrorState message={error} /> : null}
      {loading && !error ? <LoadingState /> : null}

      {!loading && !error && (
        <>
          <section className="panel">
            <h2>Domaines</h2>
            <p>{domains.length} domaine(s), {keys.length} clé(s), {values.length} valeur(s), {missing.length} clé(s) manquante(s).</p>
            <p><a href="/nexus/translations/domains">Gérer les domaines →</a></p>
            {domains.length ? (
              <ul>
                {domains.map((d) => (
                  <li key={d.ID}>{d.Code} — {d.Name} ({d.Description})</li>
                ))}
              </ul>
            ) : (
              <p className="muted">Aucun domaine pour le moment. Lance un import ou crée un domaine.</p>
            )}
          </section>

          <section className="panel">
            <h2>Recherche clés</h2>
            <input
              type="text"
              placeholder="Filtrer par clé ou description..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              style={{ width: "100%", padding: 8, marginBottom: 12 }}
            />
            <table className="data-table">
              <thead>
                <tr>
                  <th>Domaine</th>
                  <th>Clé</th>
                  <th>Description</th>
                  <th>Valeurs</th>
                </tr>
              </thead>
              <tbody>
                {filteredKeys.slice(0, 50).map((k) => {
                  const keyValues = values.filter((v) => v.KeyID === k.ID);
                  return (
                    <tr key={k.ID}>
                      <td>{k.Domain?.Code || k.DomainID}</td>
                      <td><code>{k.Key}</code></td>
                      <td>{k.Description}</td>
                      <td>
                        {keyValues.map((v) => (
                          <div key={v.ID}>{v.Locale}: {v.Value}</div>
                        ))}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
            {!filteredKeys.length ? <p className="muted">Aucune clé à afficher.</p> : null}
            {filteredKeys.length > 50 && <p>... et {filteredKeys.length - 50} autres</p>}
          </section>

          <section className="panel">
            <h2>Actions</h2>
            <ul>
              <li><a href="/nexus/translations/import">Import / Preview / Commit →</a></li>
              <li><a href="/nexus/translations/missing">Clés manquantes →</a></li>
              <li><a href="/nexus/translations/logs">Logs imports →</a></li>
            </ul>
          </section>

          <button onClick={reload}>Rafraîchir</button>
        </>
      )}
    </AdminShell>
  );
}
