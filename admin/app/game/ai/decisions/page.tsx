"use client";

import { useEffect, useMemo, useState } from "react";
import { AdminShell } from "../../../components/AdminShell";
import { ErrorState, LoadingState } from "../../../components/LoadState";
import { formatDate, loadAdminData } from "../../../components/api";

type AIDecision = {
  id: number;
  worldId: number;
  continentId?: number | null;
  type: string;
  provider: string;
  model: string;
  status: string;
  error: string;
  inputSnapshotJson: unknown;
  outputDecisionJson: unknown;
  appliedChangesJson: unknown;
  createdAt: string;
};

export default function AIDecisionsPage() {
  const [items, setItems] = useState<AIDecision[]>([]);
  const [selected, setSelected] = useState<AIDecision | null>(null);
  const [filters, setFilters] = useState({ worldId: "", continentId: "", status: "", type: "" });
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const query = useMemo(() => {
    const params = new URLSearchParams();
    for (const [key, value] of Object.entries(filters)) {
      if (value.trim()) {
        params.set(key, value.trim());
      }
    }
    const qs = params.toString();
    return `game/ai/decisions${qs ? `?${qs}` : ""}`;
  }, [filters]);

  useEffect(() => {
    setLoading(true);
    setError(null);
    loadAdminData<{ items: AIDecision[] }>(query)
      .then((payload) => setItems(payload.items ?? []))
      .catch((err: Error) => setError(err.message))
      .finally(() => setLoading(false));
  }, [query]);

  async function dryRun(item: AIDecision) {
    const response = await fetch(`/admin/api/game/ai/decisions/${item.id}/replay-dry-run`, { method: "POST", credentials: "same-origin", headers: { Accept: "application/json" } });
    if (!response.ok) {
      setError(`Dry-run echoue: HTTP ${response.status}`);
      return;
    }
    const payload = await response.json();
    setSelected((payload.decision ?? payload.source ?? item) as AIDecision);
  }

  return (
    <AdminShell title="Décisions IA" description="Toutes les décisions NEXUS: contexte d'entrée, sortie JSON, changements appliqués, provider, modèle et erreurs.">
      <section className="panel game-filters">
        {(["worldId", "continentId", "status", "type"] as const).map((key) => (
          <label key={key}>
            <span>{key}</span>
            <input value={filters[key]} onChange={(event) => setFilters((prev) => ({ ...prev, [key]: event.target.value }))} />
          </label>
        ))}
      </section>
      {error ? <ErrorState message={error} /> : null}
      {loading ? <LoadingState /> : null}
      {!loading ? (
        <section className="panel">
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>ID</th>
                  <th>Date</th>
                  <th>Monde</th>
                  <th>Continent</th>
                  <th>Type</th>
                  <th>Provider</th>
                  <th>Modèle</th>
                  <th>Status</th>
                  <th>Erreur</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {items.map((item) => (
                  <tr key={item.id}>
                    <td>{item.id}</td>
                    <td>{formatDate(item.createdAt)}</td>
                    <td>{item.worldId}</td>
                    <td>{item.continentId ?? "-"}</td>
                    <td>{item.type || "-"}</td>
                    <td>{item.provider || "-"}</td>
                    <td>{item.model || "-"}</td>
                    <td>{item.status || "-"}</td>
                    <td>{item.error || "-"}</td>
                    <td className="row-actions">
                      <button className="secondary" type="button" onClick={() => setSelected(item)}>
                        Voir détail
                      </button>
                      <button className="secondary" type="button" onClick={() => dryRun(item)}>
                        Dry-run
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {!items.length ? <p className="hint">Aucune décision IA.</p> : null}
        </section>
      ) : null}
      {selected ? <DecisionModal item={selected} onClose={() => setSelected(null)} /> : null}
    </AdminShell>
  );
}

function DecisionModal({ item, onClose }: { item: AIDecision; onClose: () => void }) {
  return (
    <div className="modal-backdrop">
      <section className="panel json-modal decision-modal">
        <h2>Décision IA #{item.id ?? "-"}</h2>
        <div className="decision-grid">
          <DecisionBlock title="Input snapshot" value={item.inputSnapshotJson} />
          <DecisionBlock title="Output décision" value={item.outputDecisionJson} />
          <DecisionBlock title="Changements appliqués" value={item.appliedChangesJson} />
          <DecisionBlock title="Erreur" value={item.error || "Aucune"} />
        </div>
        <div className="editor-actions">
          <button className="secondary" type="button" onClick={onClose}>
            Fermer
          </button>
        </div>
      </section>
    </div>
  );
}

function DecisionBlock({ title, value }: { title: string; value: unknown }) {
  return (
    <article className="decision-block">
      <h3>{title}</h3>
      <pre>{formatJson(value)}</pre>
    </article>
  );
}

function formatJson(value: unknown): string {
  if (typeof value === "string") {
    try {
      return JSON.stringify(JSON.parse(value), null, 2);
    } catch {
      return value;
    }
  }
  return JSON.stringify(value ?? {}, null, 2);
}
