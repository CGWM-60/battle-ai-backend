"use client";

import Link from "next/link";
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

type LocaleOption = {
  code: string;
  label: string;
  group: string;
};

type TranslationGridRow = {
  keyId: number;
  domain: string;
  key: string;
  description: string;
  sourceValue: string;
  values: Record<string, TranslationValue | undefined>;
  missingLocales: string[];
  state: "complete" | "partial" | "missing";
};

const ISO_LOCALES: LocaleOption[] = [
  { code: "fr", label: "Français source", group: "Base" },
  { code: "fr-FR", label: "Français France", group: "Europe" },
  { code: "en-GB", label: "English UK", group: "Europe" },
  { code: "en-US", label: "English US", group: "US" },
  { code: "es-US", label: "Español US", group: "US" },
  { code: "de-DE", label: "Deutsch", group: "Europe" },
  { code: "es-ES", label: "Español España", group: "Europe" },
  { code: "it-IT", label: "Italiano", group: "Europe" },
  { code: "pt-PT", label: "Português Portugal", group: "Europe" },
  { code: "nl-NL", label: "Nederlands", group: "Europe" },
  { code: "sv-SE", label: "Svenska", group: "Europe" },
  { code: "da-DK", label: "Dansk", group: "Europe" },
  { code: "fi-FI", label: "Suomi", group: "Europe" },
  { code: "no-NO", label: "Norsk", group: "Europe" },
  { code: "pl-PL", label: "Polski", group: "Europe" },
  { code: "cs-CZ", label: "Čeština", group: "Europe" },
  { code: "sk-SK", label: "Slovenčina", group: "Europe" },
  { code: "sl-SI", label: "Slovenščina", group: "Europe" },
  { code: "hr-HR", label: "Hrvatski", group: "Europe" },
  { code: "hu-HU", label: "Magyar", group: "Europe" },
  { code: "ro-RO", label: "Română", group: "Europe" },
  { code: "bg-BG", label: "Български", group: "Europe" },
  { code: "el-GR", label: "Ελληνικά", group: "Europe" },
  { code: "et-EE", label: "Eesti", group: "Europe" },
  { code: "lv-LV", label: "Latviešu", group: "Europe" },
  { code: "lt-LT", label: "Lietuvių", group: "Europe" },
  { code: "ga-IE", label: "Gaeilge", group: "Europe" },
  { code: "mt-MT", label: "Malti", group: "Europe" },
  { code: "is-IS", label: "Íslenska", group: "Europe" },
  { code: "uk-UA", label: "Українська", group: "Europe" },
];

const DEFAULT_VISIBLE_LOCALES = ["fr", "en-US", "en-GB", "de-DE", "es-ES", "it-IT"];

export default function TranslationsPage() {
  const [domains, setDomains] = useState<TranslationDomain[]>([]);
  const [keys, setKeys] = useState<TranslationKey[]>([]);
  const [values, setValues] = useState<TranslationValue[]>([]);
  const [missing, setMissing] = useState<TranslationMissingLog[]>([]);
  const [providers, setProviders] = useState<AIProvider[]>([]);
  const [localeCatalog, setLocaleCatalog] = useState<LocaleOption[]>(ISO_LOCALES);

  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  const [visibleLocales, setVisibleLocales] = useState<string[]>(DEFAULT_VISIBLE_LOCALES);
  const [domainFilter, setDomainFilter] = useState("");
  const [keyFilter, setKeyFilter] = useState("");
  const [sourceFilter, setSourceFilter] = useState("");
  const [stateFilter, setStateFilter] = useState<"" | TranslationGridRow["state"]>("");
  const [localeFilters, setLocaleFilters] = useState<Record<string, string>>({});

  const [sourceLocale, setSourceLocale] = useState("fr");
  const [lineTargetLocale, setLineTargetLocale] = useState("en-US");
  const [provider, setProvider] = useState("");
  const [model, setModel] = useState("");
  const [batchLimit, setBatchLimit] = useState(50);

  const [pageIndex, setPageIndex] = useState(0);
  const [pageSize, setPageSize] = useState(25);
  const [editingCell, setEditingCell] = useState<string | null>(null);
  const [editingValue, setEditingValue] = useState("");

  const reload = () => {
    setLoading(true);
    setError(null);
    Promise.all([
      loadAdminData<any>("translations/domains"),
      loadAdminData<any>("translations/keys"),
      loadAdminData<any>("translations/values"),
      loadAdminData<any>("translations/missing"),
      loadAdminData<any>("translations/locales/catalog"),
      loadAdminData<any>("translations/ai/providers"),
    ])
      .then(([d, k, v, m, catalog, p]) => {
        setDomains(d?.domains || d || []);
        setKeys(k?.keys || k || []);
        setValues(v?.values || v || []);
        setMissing(m?.missing || m || []);
        const nextLocales = catalog?.locales || catalog || [];
        if (nextLocales.length) {
          setLocaleCatalog(nextLocales);
        }
        const nextProviders = p?.providers || [];
        setProviders(nextProviders);
        if (!provider && nextProviders.length) {
          const selected = nextProviders.find((item: AIProvider) => item.configured) || nextProviders[0];
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

  useEffect(() => {
    setPageIndex(0);
  }, [domainFilter, keyFilter, sourceFilter, stateFilter, localeFilters, visibleLocales, pageSize]);

  const localeOptions = useMemo(() => {
    const known = new Map(localeCatalog.map((item) => [item.code, item]));
    values.forEach((value) => {
      if (!known.has(value.Locale)) {
        known.set(value.Locale, { code: value.Locale, label: value.Locale, group: "Base" });
      }
    });
    return Array.from(known.values()).sort((a, b) => {
      if (a.group !== b.group) return a.group.localeCompare(b.group);
      return a.code.localeCompare(b.code);
    });
  }, [localeCatalog, values]);

  const gridRows = useMemo<TranslationGridRow[]>(() => {
    const valuesByKeyLocale = new Map<string, TranslationValue>();
    values.forEach((value) => valuesByKeyLocale.set(`${value.KeyID}:${value.Locale}`, value));

    return keys.map((item) => {
      const domain = item.Domain?.Code || String(item.DomainID);
      const rowValues: Record<string, TranslationValue | undefined> = {};
      visibleLocales.forEach((locale) => {
        rowValues[locale] = valuesByKeyLocale.get(`${item.ID}:${locale}`);
      });
      const sourceValue = valuesByKeyLocale.get(`${item.ID}:${sourceLocale}`)?.Value || "";
      const missingLocales = visibleLocales.filter((locale) => !(rowValues[locale]?.Value || "").trim());
      const state = missingLocales.length === 0 ? "complete" : missingLocales.length === visibleLocales.length ? "missing" : "partial";
      return {
        keyId: item.ID,
        domain,
        key: item.Key,
        description: item.Description || "",
        sourceValue,
        values: rowValues,
        missingLocales,
        state,
      };
    });
  }, [keys, sourceLocale, values, visibleLocales]);

  const filteredRows = useMemo(() => {
    const domainNeedle = domainFilter.trim().toLowerCase();
    const keyNeedle = keyFilter.trim().toLowerCase();
    const sourceNeedle = sourceFilter.trim().toLowerCase();
    return gridRows.filter((row) => {
      if (domainNeedle && !row.domain.toLowerCase().includes(domainNeedle)) return false;
      if (keyNeedle && !`${row.key} ${row.description}`.toLowerCase().includes(keyNeedle)) return false;
      if (sourceNeedle && !row.sourceValue.toLowerCase().includes(sourceNeedle)) return false;
      if (stateFilter && row.state !== stateFilter) return false;
      for (const locale of visibleLocales) {
        const filter = (localeFilters[locale] || "").trim().toLowerCase();
        if (filter && !(row.values[locale]?.Value || "").toLowerCase().includes(filter)) return false;
      }
      return true;
    });
  }, [domainFilter, gridRows, keyFilter, localeFilters, sourceFilter, stateFilter, visibleLocales]);

  const totalPages = Math.max(1, Math.ceil(filteredRows.length / pageSize));
  const safePageIndex = Math.min(pageIndex, totalPages - 1);
  const pageRows = filteredRows.slice(safePageIndex * pageSize, safePageIndex * pageSize + pageSize);

  const summary = useMemo(() => {
    const complete = filteredRows.filter((row) => row.state === "complete").length;
    const partial = filteredRows.filter((row) => row.state === "partial").length;
    const missingRows = filteredRows.filter((row) => row.state === "missing").length;
    const missingCells = filteredRows.reduce((count, row) => count + row.missingLocales.length, 0);
    return { complete, partial, missingRows, missingCells };
  }, [filteredRows]);

  const toggleLocale = (locale: string) => {
    setVisibleLocales((current) => {
      if (current.includes(locale)) {
        if (current.length === 1) return current;
        return current.filter((item) => item !== locale);
      }
      return [...current, locale];
    });
  };

  const setEuropeLocales = () => {
    setVisibleLocales(localeOptions.filter((item) => item.group === "Europe" || item.code === "fr").map((item) => item.code));
  };

  const saveCell = async (row: TranslationGridRow, locale: string) => {
    const value = editingValue.trim();
    if (!value) {
      setError("Valeur vide refusée.");
      return;
    }
    const existing = row.values[locale];
    setBusy(true);
    setError(null);
    setNotice(null);
    try {
      const response = existing
        ? await fetch(`/admin/api/translations/values/${existing.ID}`, {
            method: "PUT",
            headers: { "Content-Type": "application/json" },
            credentials: "same-origin",
            body: JSON.stringify({ KeyID: row.keyId, Locale: locale, Value: value }),
          })
        : await fetch("/admin/api/translations/batch-update", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            credentials: "same-origin",
            body: JSON.stringify([{ domain: row.domain, key: row.key, locale, value }]),
          });
      if (!response.ok) throw new Error(await response.text());
      setEditingCell(null);
      setEditingValue("");
      setNotice(`Valeur ${locale} enregistrée pour ${row.key}.`);
      reload();
    } catch (e: any) {
      setError(e.message || "Erreur de sauvegarde.");
    } finally {
      setBusy(false);
    }
  };

  const translateKeys = async (keysToTranslate: string[], targets: string[], limit: number) => {
    setBusy(true);
    setError(null);
    setNotice(null);
    let translated = 0;
    let errors = 0;
    try {
      for (const targetLocale of targets) {
        if (!targetLocale || targetLocale === sourceLocale) continue;
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
            keys: keysToTranslate.length ? keysToTranslate : undefined,
            domains: domainFilter ? [domainFilter] : undefined,
          }),
        });
        if (!response.ok) throw new Error(await response.text());
        const result = await response.json();
        translated += result.translated || 0;
        errors += result.errors || 0;
      }
      setNotice(`Traduction IA terminée : ${translated} valeur(s), ${errors} erreur(s).`);
      reload();
    } catch (e: any) {
      setError(e.message || "Traduction IA impossible.");
    } finally {
      setBusy(false);
    }
  };

  const exportVisible = async () => {
    const exportRows = filteredRows.map((row) => ({
      domain: row.domain,
      key: row.key,
      source: row.sourceValue,
      values: Object.fromEntries(visibleLocales.map((locale) => [locale, row.values[locale]?.Value || ""])),
      state: row.state,
    }));
    const blob = new Blob([JSON.stringify(exportRows, null, 2)], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = "nexus-translations-table.json";
    anchor.click();
    URL.revokeObjectURL(url);
  };

  const exportCsv = (locales: string[], fileName: string) => {
    const valuesByKeyLocale = new Map<string, TranslationValue>();
    values.forEach((value) => valuesByKeyLocale.set(`${value.KeyID}:${value.Locale}`, value));
    const headers = ["domain", "key", "description", ...locales];
    const rows = filteredRows.map((row) => [
      row.domain,
      row.key,
      row.description,
      ...locales.map((locale) => valuesByKeyLocale.get(`${row.keyId}:${locale}`)?.Value || ""),
    ]);
    downloadText(fileName, toCsv([headers, ...rows]), "text/csv;charset=utf-8");
  };

  const exportVisibleCsv = () => exportCsv(visibleLocales, "translations-visible.csv");
  const exportAllCsv = () => exportCsv(localeOptions.map((item) => item.code), "translations-all-locales.csv");

  return (
    <AdminShell
      title="Traductions Nexus"
      description="Tableau type Lexik: domaines, clés, locales dynamiques, filtres par colonne, pagination et traduction IA via Go."
    >
      {error ? <ErrorState message={error} /> : null}
      {notice ? <p className="alert ok">{notice}</p> : null}
      {loading && !error ? <LoadingState /> : null}

      {!loading && !error && (
        <>
          <section className="translation-hero">
            <div>
              <p className="eyebrow">Lexik-style translation grid</p>
              <h2>Catalogue serveur</h2>
              <p>
                {keys.length} clé(s), {values.length} valeur(s), {missing.length} log(s) missing. Toutes les actions passent par les APIs Go.
              </p>
            </div>
            <div className="translation-kpis">
              <div><span>Complètes</span><strong>{summary.complete}</strong></div>
              <div><span>Partielles</span><strong>{summary.partial}</strong></div>
              <div><span>Lignes vides</span><strong>{summary.missingRows}</strong></div>
              <div><span>Cellules à traduire</span><strong>{summary.missingCells}</strong></div>
            </div>
          </section>

          <section className="panel">
            <div className="translation-toolbar">
              <label>
                Source
                <select value={sourceLocale} onChange={(event) => setSourceLocale(event.target.value)}>
                  {localeOptions.map((item) => <option key={item.code} value={item.code}>{item.code} · {item.label}</option>)}
                </select>
              </label>
              <label>
                Locale ligne
                <select value={lineTargetLocale} onChange={(event) => setLineTargetLocale(event.target.value)}>
                  {localeOptions.map((item) => <option key={item.code} value={item.code}>{item.code} · {item.label}</option>)}
                </select>
              </label>
              <label>
                Provider IA
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
                Batch
                <input
                  type="number"
                  min={1}
                  max={100}
                  value={batchLimit}
                  onChange={(event) => setBatchLimit(Number(event.target.value))}
                />
              </label>
            </div>
            <div className="translation-actions">
              <button className="primary" onClick={() => translateKeys([], visibleLocales, batchLimit)} disabled={busy}>
                Traduction globale par batch
              </button>
              <button className="secondary" onClick={() => translateKeys([], [lineTargetLocale], batchLimit)} disabled={busy || lineTargetLocale === sourceLocale}>
                Traduire locale ligne sélectionnée
              </button>
              <button className="secondary" onClick={exportVisibleCsv} disabled={busy}>Exporter CSV visibles</button>
              <button className="secondary" onClick={exportAllCsv} disabled={busy}>Exporter CSV toutes langues</button>
              <button className="secondary" onClick={exportVisible} disabled={busy}>Exporter JSON vue</button>
              <Link className="secondary-link" href="/nexus/translations/import/">Importer CSV/JSON</Link>
              <button className="secondary" onClick={reload} disabled={busy}>Rafraîchir</button>
            </div>
          </section>

          <section className="panel">
            <h2>Locales ISO préparées</h2>
            <div className="locale-toolbar">
              <button className="secondary" onClick={() => setVisibleLocales(DEFAULT_VISIBLE_LOCALES)}>Défaut</button>
              <button className="secondary" onClick={setEuropeLocales}>Europe</button>
              <button className="secondary" onClick={() => setVisibleLocales(localeOptions.filter((item) => item.group === "US" || item.code === "fr").map((item) => item.code))}>US</button>
              <button className="secondary" onClick={() => setVisibleLocales(localeOptions.map((item) => item.code))}>Toutes</button>
            </div>
            <div className="locale-checkbox-grid">
              {localeOptions.map((item) => (
                <label key={item.code} className="locale-check">
                  <input
                    type="checkbox"
                    checked={visibleLocales.includes(item.code)}
                    onChange={() => toggleLocale(item.code)}
                  />
                  <span><strong>{item.code}</strong>{item.label}<em>{item.group}</em></span>
                </label>
              ))}
            </div>
          </section>

          <section className="panel translation-table-panel">
            <div className="translation-table-header">
              <h2>Tableau des traductions</h2>
              <div className="pagination-controls">
                <span>{filteredRows.length} ligne(s)</span>
                <select value={pageSize} onChange={(event) => setPageSize(Number(event.target.value))}>
                  {[10, 25, 50, 100].map((size) => <option key={size} value={size}>{size} / page</option>)}
                </select>
                <button className="secondary" onClick={() => setPageIndex(Math.max(0, safePageIndex - 1))} disabled={safePageIndex === 0}>Précédent</button>
                <strong>{safePageIndex + 1} / {totalPages}</strong>
                <button className="secondary" onClick={() => setPageIndex(Math.min(totalPages - 1, safePageIndex + 1))} disabled={safePageIndex + 1 >= totalPages}>Suivant</button>
              </div>
            </div>

            <div className="table-wrap translation-grid-wrap">
              <table className="translation-grid-table">
                <thead>
                  <tr>
                    <th className="sticky-col domain-col">Domaine</th>
                    <th className="sticky-col key-col">Clé</th>
                    <th>Valeur source</th>
                    {visibleLocales.map((locale) => <th key={locale}>{locale}</th>)}
                    <th>État</th>
                    <th>Action</th>
                  </tr>
                  <tr className="filter-row">
                    <th className="sticky-col domain-col">
                      <input value={domainFilter} onChange={(event) => setDomainFilter(event.target.value)} placeholder="Filtrer domaine" />
                    </th>
                    <th className="sticky-col key-col">
                      <input value={keyFilter} onChange={(event) => setKeyFilter(event.target.value)} placeholder="Filtrer clé" />
                    </th>
                    <th>
                      <input value={sourceFilter} onChange={(event) => setSourceFilter(event.target.value)} placeholder="Filtrer valeur" />
                    </th>
                    {visibleLocales.map((locale) => (
                      <th key={locale}>
                        <input
                          value={localeFilters[locale] || ""}
                          onChange={(event) => setLocaleFilters((current) => ({ ...current, [locale]: event.target.value }))}
                          placeholder={`Filtrer ${locale}`}
                        />
                      </th>
                    ))}
                    <th>
                      <select value={stateFilter} onChange={(event) => setStateFilter(event.target.value as "" | TranslationGridRow["state"])}>
                        <option value="">Tous</option>
                        <option value="complete">OK</option>
                        <option value="partial">Partiel</option>
                        <option value="missing">Missing</option>
                      </select>
                    </th>
                    <th />
                  </tr>
                </thead>
                <tbody>
                  {pageRows.map((row) => (
                    <tr key={row.key}>
                      <td className="sticky-col domain-col"><span className="status">{row.domain}</span></td>
                      <td className="sticky-col key-col">
                        <code>{row.key}</code>
                        {row.description ? <small>{row.description}</small> : null}
                      </td>
                      <td className="source-cell">{row.sourceValue || <span className="muted-panel">-</span>}</td>
                      {visibleLocales.map((locale) => {
                        const cell = row.values[locale];
                        const cellId = `${row.key}:${locale}`;
                        const isEditing = editingCell === cellId;
                        return (
                          <td key={locale} className={cell?.Value ? "translation-cell ok" : "translation-cell missing"}>
                            {isEditing ? (
                              <>
                                <textarea value={editingValue} onChange={(event) => setEditingValue(event.target.value)} />
                                <div className="cell-actions">
                                  <button className="primary" onClick={() => saveCell(row, locale)} disabled={busy}>OK</button>
                                  <button className="secondary" onClick={() => setEditingCell(null)} disabled={busy}>Annuler</button>
                                </div>
                              </>
                            ) : (
                              <>
                                <p>{cell?.Value || "Valeur manquante"}</p>
                                <button
                                  className="link-button"
                                  onClick={() => {
                                    setEditingCell(cellId);
                                    setEditingValue(cell?.Value || "");
                                  }}
                                >
                                  Éditer
                                </button>
                              </>
                            )}
                          </td>
                        );
                      })}
                      <td><span className={`status ${row.state}`}>{row.state}</span></td>
                      <td>
                        <div className="row-actions">
                          <button className="secondary" onClick={() => translateKeys([row.key], [lineTargetLocale], 1)} disabled={busy || lineTargetLocale === sourceLocale}>
                            Traduire ligne
                          </button>
                          <button className="secondary" onClick={() => translateKeys([row.key], visibleLocales, visibleLocales.length)} disabled={busy}>
                            Toutes locales
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            {!pageRows.length ? <p className="hint">Aucune traduction ne correspond aux filtres.</p> : null}
          </section>
        </>
      )}
    </AdminShell>
  );
}

function downloadText(fileName: string, text: string, type: string) {
  const blob = new Blob([text], { type });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = fileName;
  anchor.click();
  URL.revokeObjectURL(url);
}

function toCsv(rows: string[][]): string {
  return rows.map((row) => row.map(escapeCsvCell).join(",")).join("\r\n");
}

function escapeCsvCell(value: string): string {
  const safeValue = String(value ?? "");
  if (/[",\r\n]/.test(safeValue)) {
    return `"${safeValue.replace(/"/g, '""')}"`;
  }
  return safeValue;
}
