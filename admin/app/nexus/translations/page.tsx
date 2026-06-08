"use client";

import { useEffect, useMemo, useState } from "react";
import { AdminShell } from "../../components/AdminShell";
import { ErrorState, LoadingState } from "../../components/LoadState";
import { loadAdminData } from "../../components/api";
import type {
  TranslationDomain,
  TranslationKey,
  TranslationMissingLog,
  TranslationValue,
} from "../../types";

type AIProvider = {
  providerType: string;
  displayName: string;
  defaultModel: string;
  configured: boolean;
};

type TranslationTableRow = {
  id: string;
  valueId: number | null;
  keyId: number;
  domain: string;
  key: string;
  description: string;
  locale: string;
  value: string;
  missing: boolean;
};

export default function TranslationsPage() {
  const [domains, setDomains] = useState<TranslationDomain[]>([]);
  const [keys, setKeys] = useState<TranslationKey[]>([]);
  const [values, setValues] = useState<TranslationValue[]>([]);
  const [missing, setMissing] = useState<TranslationMissingLog[]>([]);
  const [providers, setProviders] = useState<AIProvider[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);

  const [search, setSearch] = useState("");
  const [domainFilter, setDomainFilter] = useState("");
  const [locale, setLocale] = useState("fr");
  const [sourceLocale, setSourceLocale] = useState("fr");
  const [targetLocale, setTargetLocale] = useState("en");
  const [provider, setProvider] = useState("");
  const [model, setModel] = useState("");
  const [limit, setLimit] = useState(25);

  const [editingRow, setEditingRow] = useState<string | null>(null);
  const [editingValue, setEditingValue] = useState("");

  const reload = () => {
    setLoading(true);
    setError(null);
    Promise.all([
      loadAdminData<any>("translations/domains"),
      loadAdminData<any>("translations/keys"),
      loadAdminData<any>("translations/values"),
      loadAdminData<any>("translations/missing"),
      loadAdminData<any>("translations/ai/providers"),
    ])
      .then(([d, k, v, m, p]) => {
        setDomains(d?.domains || d || []);
        setKeys(k?.keys || k || []);
        setValues(v?.values || v || []);
        setMissing(m?.missing || m || []);
        const nextProviders = p?.providers || [];
        setProviders(nextProviders);
        if (!provider && nextProviders.length) {
          const configured = nextProviders.find((item: AIProvider) => item.configured);
          const selected = configured || nextProviders[0];
          setProvider(selected.providerType);
          setModel(selected.defaultModel || "");
        }
      })
      .catch((e: Error) => setError(e.message))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    reload();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const locales = useMemo(() => {
    const all = new Set<string>(["fr", locale, sourceLocale, targetLocale]);
    values.forEach((value) => all.add(value.Locale));
    missing.forEach((item) => all.add(item.Locale));
    return Array.from(all).filter(Boolean).sort();
  }, [locale, missing, sourceLocale, targetLocale, values]);

  const tableRows = useMemo<TranslationTableRow[]>(() => {
    const needle = search.trim().toLowerCase();
    return keys
      .map((item) => {
        const domain = item.Domain?.Code || String(item.DomainID);
        const existing = values.find((value) => value.KeyID === item.ID && value.Locale === locale);
        return {
          id: existing ? `value-${existing.ID}` : `missing-${item.ID}-${locale}`,
          valueId: existing?.ID ?? null,
          keyId: item.ID,
          domain,
          key: item.Key,
          description: item.Description || "",
          locale,
          value: existing?.Value || "",
          missing: !existing || !existing.Value,
        };
      })
      .filter((row) => !domainFilter || row.domain === domainFilter)
      .filter((row) => {
        if (!needle) return true;
        return [row.domain, row.key, row.description, row.locale, row.value]
          .some((value) => value.toLowerCase().includes(needle));
      });
  }, [domainFilter, keys, locale, search, values]);

  const visibleRows = tableRows.slice(0, 250);
  const missingCount = tableRows.filter((row) => row.missing).length;

  const saveRow = async (row: TranslationTableRow) => {
    const value = editingValue.trim();
    if (!value) {
      setError("Valeur vide refusée.");
      return;
    }
    setBusy(true);
    setError(null);
    setNotice(null);
    try {
      const response = row.valueId
        ? await fetch(`/admin/api/translations/values/${row.valueId}`, {
            method: "PUT",
            headers: { "Content-Type": "application/json" },
            credentials: "same-origin",
            body: JSON.stringify({ KeyID: row.keyId, Locale: row.locale, Value: value }),
          })
        : await fetch("/admin/api/translations/batch-update", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            credentials: "same-origin",
            body: JSON.stringify([
              {
                domain: row.domain,
                key: row.key,
                locale: row.locale,
                value,
              },
            ]),
          });
      if (!response.ok) throw new Error(await response.text());
      setEditingRow(null);
      setEditingValue("");
      setNotice("Valeur enregistrée côté Go.");
      reload();
    } catch (e: any) {
      setError(e.message || "Erreur de sauvegarde.");
    } finally {
      setBusy(false);
    }
  };

  const exportLocale = async () => {
    setBusy(true);
    setError(null);
    try {
      const data = await loadAdminData<Record<string, string>>(
        `translations/export?locale=${encodeURIComponent(locale)}`,
      );
      const blob = new Blob([JSON.stringify(data, null, 2)], {
        type: "application/json",
      });
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement("a");
      anchor.href = url;
      anchor.download = `nexus-translations.${locale}.json`;
      anchor.click();
      URL.revokeObjectURL(url);
    } catch (e: any) {
      setError(e.message || "Export impossible.");
    } finally {
      setBusy(false);
    }
  };

  const translateMissing = async () => {
    setBusy(true);
    setError(null);
    setNotice(null);
    try {
      const response = await fetch("/admin/api/translations/ai/translate-missing", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "same-origin",
        body: JSON.stringify({
          sourceLocale,
          targetLocale,
          provider,
          model,
          limit,
          domains: domainFilter ? [domainFilter] : undefined,
        }),
      });
      if (!response.ok) throw new Error(await response.text());
      const result = await response.json();
      setLocale(targetLocale);
      setNotice(
        `Traduction IA terminée : ${result.translated || 0} valeur(s), ${result.errors || 0} erreur(s).`,
      );
      reload();
    } catch (e: any) {
      setError(e.message || "Traduction IA impossible.");
    } finally {
      setBusy(false);
    }
  };

  return (
    <AdminShell
      title="Traductions Nexus"
      description="Tableau Domaine / Clé / Description / Locale / Valeur. Next.js appelle uniquement les APIs Go."
    >
      {error ? <ErrorState message={error} /> : null}
      {notice ? <p className="good">{notice}</p> : null}
      {loading && !error ? <LoadingState /> : null}

      {!loading && !error && (
        <>
          <section className="panel">
            <h2>Vue traduction</h2>
            <p>
              {domains.length} domaine(s), {keys.length} clé(s), {values.length} valeur(s),{" "}
              {missing.length} log(s) missing. Locale affichée : <code>{locale}</code>.
            </p>
            <div style={{ display: "grid", gap: 12, gridTemplateColumns: "repeat(auto-fit, minmax(180px, 1fr))" }}>
              <label>
                Recherche
                <input
                  type="text"
                  placeholder="Domaine, clé, description, valeur..."
                  value={search}
                  onChange={(event) => setSearch(event.target.value)}
                />
              </label>
              <label>
                Domaine
                <select value={domainFilter} onChange={(event) => setDomainFilter(event.target.value)}>
                  <option value="">Tous</option>
                  {domains.map((domain) => (
                    <option key={domain.ID} value={domain.Code}>{domain.Code}</option>
                  ))}
                </select>
              </label>
              <label>
                Locale table
                <input value={locale} onChange={(event) => setLocale(event.target.value.trim())} list="translation-locales" />
              </label>
              <datalist id="translation-locales">
                {locales.map((item) => <option key={item} value={item} />)}
              </datalist>
            </div>
            <p className={missingCount ? "bad" : "good"}>
              {missingCount} valeur(s) manquante(s) dans la vue filtrée.
            </p>
            <p>
              <a href="/nexus/translations/import">Import / Preview / Commit</a>{" · "}
              <a href="/nexus/translations/missing">Logs clés manquantes</a>{" · "}
              <a href="/nexus/translations/logs">Logs imports</a>
            </p>
          </section>

          <section className="panel">
            <h2>Outil de traduction IA backend</h2>
            <div style={{ display: "grid", gap: 12, gridTemplateColumns: "repeat(auto-fit, minmax(180px, 1fr))" }}>
              <label>
                Source
                <input value={sourceLocale} onChange={(event) => setSourceLocale(event.target.value.trim())} />
              </label>
              <label>
                Cible
                <input value={targetLocale} onChange={(event) => setTargetLocale(event.target.value.trim())} />
              </label>
              <label>
                Provider backend
                <select
                  value={provider}
                  onChange={(event) => {
                    const selected = providers.find((item) => item.providerType === event.target.value);
                    setProvider(event.target.value);
                    setModel(selected?.defaultModel || "");
                  }}
                >
                  {providers.map((item) => (
                    <option key={item.providerType} value={item.providerType}>
                      {item.displayName} {item.configured ? "✓" : "non configuré"}
                    </option>
                  ))}
                </select>
              </label>
              <label>
                Modèle
                <input value={model} onChange={(event) => setModel(event.target.value)} />
              </label>
              <label>
                Limite
                <input
                  type="number"
                  min={1}
                  max={100}
                  value={limit}
                  onChange={(event) => setLimit(Number(event.target.value))}
                />
              </label>
            </div>
            <p>
              <button onClick={translateMissing} disabled={busy || !targetLocale || sourceLocale === targetLocale}>
                Traduire les valeurs manquantes via Go
              </button>{" "}
              <button onClick={exportLocale} disabled={busy}>Exporter locale affichée</button>{" "}
              <button onClick={reload} disabled={busy}>Rafraîchir</button>
            </p>
          </section>

          <section className="panel">
            <h2>Tableau Flutter</h2>
            <table className="data-table">
              <thead>
                <tr>
                  <th>Domaine</th>
                  <th>Clé</th>
                  <th>Description</th>
                  <th>Locale</th>
                  <th>Valeur</th>
                  <th>État</th>
                  <th>Action</th>
                </tr>
              </thead>
              <tbody>
                {visibleRows.map((row) => (
                  <tr key={row.id}>
                    <td>{row.domain}</td>
                    <td><code>{row.key}</code></td>
                    <td>{row.description || "-"}</td>
                    <td><code>{row.locale}</code></td>
                    <td>
                      {editingRow === row.id ? (
                        <textarea
                          value={editingValue}
                          onChange={(event) => setEditingValue(event.target.value)}
                          rows={3}
                          style={{ width: "100%" }}
                        />
                      ) : row.value ? (
                        row.value
                      ) : (
                        <span className="muted">Valeur manquante</span>
                      )}
                    </td>
                    <td className={row.missing ? "bad" : "good"}>
                      {row.missing ? "missing" : "ok"}
                    </td>
                    <td>
                      {editingRow === row.id ? (
                        <>
                          <button onClick={() => saveRow(row)} disabled={busy}>Enregistrer</button>{" "}
                          <button onClick={() => setEditingRow(null)} disabled={busy}>Annuler</button>
                        </>
                      ) : (
                        <button
                          onClick={() => {
                            setEditingRow(row.id);
                            setEditingValue(row.value);
                          }}
                        >
                          Modifier
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            {!visibleRows.length ? <p className="muted">Aucune clé à afficher.</p> : null}
            {tableRows.length > visibleRows.length ? (
              <p>... et {tableRows.length - visibleRows.length} autre(s) ligne(s). Affine la recherche pour éditer.</p>
            ) : null}
          </section>
        </>
      )}
    </AdminShell>
  );
}
