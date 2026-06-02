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
  isActive?: boolean;
  error: string;
  inputSnapshotJson: unknown;
  outputDecisionJson: unknown;
  appliedChangesJson: unknown;
  createdAt: string;
};

type WorldAICron = {
  enabled: boolean;
  started: boolean;
  lightInterval: string;
  continentalInterval: string;
  dailyInterval: string;
  routineInterval: string;
  lastChangedBy: string;
  lastChangedAt: string;
  lastRunType: string;
  lastRunStatus: string;
  lastRunAt: string;
  lastError: string;
};

export default function AIDecisionsPage() {
  const [items, setItems] = useState<AIDecision[]>([]);
  const [selected, setSelected] = useState<AIDecision | null>(null);
  const [filters, setFilters] = useState({ worldId: "", continentId: "", status: "", type: "", isActive: "" });
  const [cron, setCron] = useState<WorldAICron | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [busyId, setBusyId] = useState<number | null>(null);
  const [cronBusy, setCronBusy] = useState(false);

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
    reload();
    loadCron();
  }, [query]);

  function reload() {
    setLoading(true);
    setError(null);
    loadAdminData<{ items: AIDecision[] }>(query)
      .then((payload) => setItems(payload.items ?? []))
      .catch((err: Error) => setError(err.message))
      .finally(() => setLoading(false));
  }

  async function loadCron() {
    try {
      const payload = await loadAdminData<{ cron: WorldAICron }>("game/ai/cron");
      setCron(payload.cron);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Chargement cron IA échoué");
    }
  }

  const counters = useMemo(() => {
    const active = items.filter((item) => item.isActive !== false).length;
    return {
      active,
      disabled: items.length - active,
      total: items.length,
    };
  }, [items]);

  async function dryRun(item: AIDecision) {
    const response = await fetch(`/admin/api/game/ai/decisions/${item.id}/replay-dry-run`, { method: "POST", credentials: "same-origin", headers: { Accept: "application/json" } });
    if (!response.ok) {
      setError(`Dry-run echoue: HTTP ${response.status}`);
      return;
    }
    const payload = await response.json();
    setSelected((payload.decision ?? payload.source ?? item) as AIDecision);
  }

  async function toggleActivation(item: AIDecision) {
    const nextActive = item.isActive === false;
    setBusyId(item.id);
    setError(null);
    const response = await fetch(`/admin/api/game/ai/decisions/${item.id}/${nextActive ? "enable" : "disable"}`, {
      method: "POST",
      credentials: "same-origin",
      headers: { Accept: "application/json" },
    });
    setBusyId(null);
    if (!response.ok) {
      setError(`${nextActive ? "Activation" : "Désactivation"} échouée: HTTP ${response.status}`);
      return;
    }
    const payload = await response.json();
    const updated = (payload.decision ?? { ...item, isActive: nextActive }) as AIDecision;
    setItems((prev) => prev.map((candidate) => (candidate.id === item.id ? updated : candidate)));
    setSelected((prev) => (prev?.id === item.id ? updated : prev));
  }

  async function toggleCron() {
    const nextEnabled = !(cron?.enabled ?? false);
    setCronBusy(true);
    setError(null);
    const response = await fetch(`/admin/api/game/ai/cron/${nextEnabled ? "enable" : "disable"}`, {
      method: "POST",
      credentials: "same-origin",
      headers: { Accept: "application/json" },
    });
    setCronBusy(false);
    if (!response.ok) {
      setError(`${nextEnabled ? "Activation" : "Désactivation"} du cron IA échouée: HTTP ${response.status}`);
      return;
    }
    const payload = await response.json();
    setCron(payload.cron);
  }

  return (
    <AdminShell title="Décisions IA" description="Toutes les décisions NEXUS: contexte d'entrée, sortie JSON, changements appliqués, provider, modèle et erreurs.">
      <section className="panel cron-control">
        <div>
          <h2>Tâche cron IA monde</h2>
          <p className="hint">
            <span className={`status ${cron?.enabled ? "enabled" : "disabled"}`}>{cron?.enabled ? "Activée" : "Désactivée"}</span>{" "}
            Routine 4 pages: {cron?.routineInterval || "-"} · Simulation légère: {cron?.lightInterval || "-"} · Continentale: {cron?.continentalInterval || "-"} · Daily: {cron?.dailyInterval || "-"}
          </p>
          <p className="hint">
            Dernier run: {cron?.lastRunType || "-"} · {cron?.lastRunStatus || "-"} · {formatDate(cron?.lastRunAt)}
            {cron?.lastError ? ` · ${cron.lastError}` : ""}
          </p>
        </div>
        <button className={cron?.enabled ? "danger" : "secondary"} type="button" disabled={cronBusy || !cron?.started} onClick={toggleCron}>
          {cron?.enabled ? "Désactiver le cron IA" : "Activer le cron IA"}
        </button>
      </section>
      <section className="game-kpi-grid">
        <article className="panel game-kpi">
          <span>Total affiché</span>
          <strong>{counters.total}</strong>
        </article>
        <article className="panel game-kpi">
          <span>Décisions activées</span>
          <strong>{counters.active}</strong>
        </article>
        <article className="panel game-kpi">
          <span>Décisions désactivées</span>
          <strong>{counters.disabled}</strong>
        </article>
      </section>
      <section className="panel game-filters">
        {(["worldId", "continentId", "status", "type"] as const).map((key) => (
          <label key={key}>
            <span>{key}</span>
            <input value={filters[key]} onChange={(event) => setFilters((prev) => ({ ...prev, [key]: event.target.value }))} />
          </label>
        ))}
        <label>
          <span>Activation</span>
          <select value={filters.isActive} onChange={(event) => setFilters((prev) => ({ ...prev, isActive: event.target.value }))}>
            <option value="">Toutes</option>
            <option value="true">Activées</option>
            <option value="false">Désactivées</option>
          </select>
        </label>
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
                  <th>Activation</th>
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
                    <td>
                      <span className={`status ${item.status || "unknown"}`}>{item.status || "-"}</span>
                    </td>
                    <td>
                      <span className={`status ${item.isActive === false ? "disabled" : "enabled"}`}>{item.isActive === false ? "Désactivée" : "Activée"}</span>
                    </td>
                    <td>{item.error || "-"}</td>
                    <td className="row-actions">
                      <button className="secondary" type="button" onClick={() => setSelected(item)}>
                        Voir détail
                      </button>
                      <button className={item.isActive === false ? "secondary" : "danger"} type="button" disabled={busyId === item.id} onClick={() => toggleActivation(item)}>
                        {item.isActive === false ? "Activer" : "Désactiver"}
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
        <p className="hint">
          <span className={`status ${item.isActive === false ? "disabled" : "enabled"}`}>{item.isActive === false ? "Désactivée" : "Activée"}</span>{" "}
          <span className={`status ${item.status || "unknown"}`}>{item.status || "-"}</span>
        </p>
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
