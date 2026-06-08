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
    (k.Description || "").toLowerCase().includes(search.toLowerCase()) ||
    String(k.Domain?.Code || k.DomainID).toLowerCase().includes(search.toLowerCase()) ||
    values.some((v) =>
      v.KeyID === k.ID &&
      (v.Locale.toLowerCase().includes(search.toLowerCase()) ||
        v.Value.toLowerCase().includes(search.toLowerCase()))
    )
  );

  const translationRows = filteredKeys.flatMap((k) => {
    const keyValues = values.filter((v) => v.KeyID === k.ID);
    if (!keyValues.length) {
      return [{
        id: `missing-${k.ID}`,
        domain: k.Domain?.Code || String(k.DomainID),
        key: k.Key,
        description: k.Description,
        locale: "-",
        value: "",
      }];
    }
    return keyValues.map((v) => ({
      id: String(v.ID),
      domain: k.Domain?.Code || String(k.DomainID),
      key: k.Key,
      description: k.Description,
      locale: v.Locale,
      value: v.Value,
    }));
  });

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
            <h2>Traductions Flutter</h2>
            <input
              type="text"
              placeholder="Filtrer par domaine, clé, description, locale ou valeur..."
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
                  <th>Locale</th>
                  <th>Valeur Flutter</th>
                </tr>
              </thead>
              <tbody>
                {translationRows.slice(0, 100).map((row) => (
                  <tr key={row.id}>
                    <td>{row.domain}</td>
                    <td><code>{row.key}</code></td>
                    <td>{row.description || "-"}</td>
                    <td><code>{row.locale}</code></td>
                    <td>{row.value || <span className="muted">Valeur manquante</span>}</td>
                  </tr>
                ))}
              </tbody>
            </table>
            {!filteredKeys.length ? <p className="muted">Aucune clé à afficher.</p> : null}
            {translationRows.length > 100 && <p>... et {translationRows.length - 100} autres lignes</p>}
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
