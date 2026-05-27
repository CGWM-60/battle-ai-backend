"use client";

import { useEffect, useMemo, useState } from "react";
import { AdminShell } from "../components/AdminShell";
import { ErrorState, LoadingState } from "../components/LoadState";
import { formatDate, formatNumber, loadAdminData } from "../components/api";

type AnyRecord = Record<string, unknown>;

export type GameAdminConfig = {
  title: string;
  description: string;
  endpoint: string;
  itemKey?: string;
  columns: string[];
  filters?: string[];
  actions?: GameAdminAction[];
};

type GameAdminAction = {
  label: string;
  method: "POST" | "DELETE";
  path: (item: AnyRecord) => string;
  confirm?: string;
};

export function GameAdminPage({ config }: { config: GameAdminConfig }) {
  const [data, setData] = useState<AnyRecord[]>([]);
  const [filters, setFilters] = useState<Record<string, string>>({});
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshKey, setRefreshKey] = useState(0);

  const query = useMemo(() => {
    const params = new URLSearchParams();
    for (const [key, value] of Object.entries(filters)) {
      if (value.trim()) {
        params.set(key, value.trim());
      }
    }
    const qs = params.toString();
    return qs ? `${config.endpoint}?${qs}` : config.endpoint;
  }, [config.endpoint, filters]);

  useEffect(() => {
    setLoading(true);
    setError(null);
    loadAdminData<AnyRecord>(query)
      .then((payload) => {
        const raw = payload[config.itemKey ?? "items"];
        setData(Array.isArray(raw) ? (raw as AnyRecord[]) : Array.isArray(payload) ? (payload as AnyRecord[]) : []);
      })
      .catch((err: Error) => setError(err.message))
      .finally(() => setLoading(false));
  }, [query, config.itemKey, refreshKey]);

  async function runAction(action: GameAdminAction, item: AnyRecord) {
    if (action.confirm && !window.confirm(action.confirm)) {
      return;
    }
    const response = await fetch(`/admin/api/${action.path(item)}`, {
      method: action.method,
      credentials: "same-origin",
      headers: { Accept: "application/json" },
    });
    if (!response.ok) {
      setError(`Action echouee: HTTP ${response.status}`);
      return;
    }
    setRefreshKey((value) => value + 1);
  }

  return (
    <AdminShell title={config.title} description={config.description}>
      {config.filters?.length ? (
        <section className="panel game-filters">
          {config.filters.map((filter) => (
            <label key={filter}>
              <span>{filter}</span>
              <input value={filters[filter] ?? ""} onChange={(event) => setFilters((prev) => ({ ...prev, [filter]: event.target.value }))} />
            </label>
          ))}
          <button className="secondary" type="button" onClick={() => setRefreshKey((value) => value + 1)}>
            Actualiser
          </button>
        </section>
      ) : null}
      {error ? <ErrorState message={error} /> : null}
      {loading ? <LoadingState /> : null}
      {!loading && !error ? (
        <section className="panel">
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  {config.columns.map((column) => (
                    <th key={column}>{column}</th>
                  ))}
                  {config.actions?.length ? <th>Actions</th> : null}
                </tr>
              </thead>
              <tbody>
                {data.map((item, index) => (
                  <tr key={`${String(item.id ?? item.Id ?? index)}`}>
                    {config.columns.map((column) => (
                      <td key={column}>{formatValue(item[column] ?? item[toPascal(column)] ?? item[toCamel(column)])}</td>
                    ))}
                    {config.actions?.length ? (
                      <td className="row-actions">
                        {config.actions.map((action) => (
                          <button className={action.method === "DELETE" ? "danger" : "secondary"} key={action.label} type="button" onClick={() => runAction(action, item)}>
                            {action.label}
                          </button>
                        ))}
                      </td>
                    ) : null}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {!data.length ? <p className="hint">Aucune donnee.</p> : null}
        </section>
      ) : null}
    </AdminShell>
  );
}

function formatValue(value: unknown): string {
  if (value === null || value === undefined || value === "") {
    return "-";
  }
  if (typeof value === "number") {
    return formatNumber(value);
  }
  if (typeof value === "boolean") {
    return value ? "oui" : "non";
  }
  if (typeof value === "string") {
    if (/^\d{4}-\d{2}-\d{2}T/.test(value)) {
      return formatDate(value);
    }
    return value;
  }
  return JSON.stringify(value);
}

function toCamel(value: string): string {
  return value.replace(/_([a-z])/g, (_, letter: string) => letter.toUpperCase());
}

function toPascal(value: string): string {
  const camel = toCamel(value);
  return camel.charAt(0).toUpperCase() + camel.slice(1);
}
