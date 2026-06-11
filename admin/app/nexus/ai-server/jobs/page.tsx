"use client";

import { useEffect, useMemo, useState } from "react";
import { Activity, Play, RefreshCw, Zap } from "lucide-react";
import { AdminShell } from "../../../components/AdminShell";

const API_BASE = (process.env.NEXT_PUBLIC_NEXUS_API_BASE || "").replace(/\/$/, "");

type JobRun = {
  id: number;
  jobKey: string;
  jobName: string;
  category: string;
  frequency: string;
  triggerType: string;
  generativeMode: string;
  status: string;
  startedAt: string;
  finishedAt?: string;
  durationMs: number;
  worldId: number;
  processed: number;
  createdCount: number;
  updatedCount: number;
  skippedCount: number;
  errorMessage?: string;
  summaryJson?: string;
};

type JobView = {
  key: string;
  name: string;
  category: string;
  frequency: string;
  intervalSeconds: number;
  generativeMode: string;
  description: string;
  lastRun?: JobRun;
  nextRunAt?: string;
  due: boolean;
};

type JobResult = {
  jobKey: string;
  status: string;
  processed: number;
  createdCount: number;
  updatedCount: number;
  skippedCount: number;
  summary?: Record<string, unknown>;
};

function formatDate(value?: string) {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString("fr-FR", { dateStyle: "short", timeStyle: "medium" });
}

function statusClass(status?: string, due?: boolean) {
  if (due) return "status started";
  return `status ${status || "skipped"}`;
}

export default function AIServerJobsPage() {
  const [jobs, setJobs] = useState<JobView[]>([]);
  const [results, setResults] = useState<JobResult[]>([]);
  const [worldId, setWorldId] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [runningKey, setRunningKey] = useState("");
  const [refreshedAt, setRefreshedAt] = useState("");

  const stats = useMemo(() => {
    return {
      total: jobs.length,
      due: jobs.filter((job) => job.due).length,
      failed: jobs.filter((job) => job.lastRun?.status === "failed").length,
      deterministic: jobs.filter((job) => job.generativeMode === "none").length,
    };
  }, [jobs]);

  const endpointQuery = () => {
    const trimmed = worldId.trim();
    return trimmed ? `?world_id=${encodeURIComponent(trimmed)}` : "";
  };

  const load = async () => {
    setError("");
    setLoading(true);
    try {
      const res = await fetch(`${API_BASE}/api/nexus-game/admin/ai-server/jobs`, { credentials: "same-origin" });
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      setJobs(data.jobs || []);
      setRefreshedAt(new Date().toISOString());
    } catch (e: any) {
      setError(e.message || "Erreur chargement jobs IA serveur");
    } finally {
      setLoading(false);
    }
  };

  const runEndpoint = async (path: string, key: string) => {
    setRunningKey(key);
    setError("");
    try {
      const res = await fetch(`${API_BASE}${path}${endpointQuery()}`, {
        method: "POST",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json" },
        body: worldId.trim() ? JSON.stringify({ worldId: Number(worldId.trim()) }) : "{}",
      });
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      const nextResults = data.results || (data.result ? [data.result] : []);
      setResults(nextResults);
      await load();
    } catch (e: any) {
      setError(e.message || "Erreur execution job IA serveur");
    } finally {
      setRunningKey("");
    }
  };

  useEffect(() => {
    load();
    const id = window.setInterval(load, 30000);
    return () => window.clearInterval(id);
  }, []);

  return (
    <AdminShell
      title="Jobs IA serveur"
      description="Tour de controle du scheduler Nexus: frequences, verrous, memoires, attaques, sabotages, broadcasts, couts et evenements saisonniers."
    >
      {error ? <div className="alert error">{error}</div> : null}

      <section className="panel cron-control">
        <div>
          <p className="eyebrow">Noyau IA / scheduler</p>
          <h2>Controle temps reel</h2>
          <p className="hint">Dernier rafraichissement: {formatDate(refreshedAt)}. Les executions respectent les garde-fous anti-frustration et validation admin des saisons.</p>
        </div>
        <div className="game-toolbar">
          <label style={{ minWidth: 140 }}>
            <span className="hint">World ID</span>
            <input value={worldId} onChange={(e) => setWorldId(e.target.value)} placeholder="Tous" inputMode="numeric" />
          </label>
          <button className="secondary" type="button" onClick={load} disabled={loading || !!runningKey}>
            <RefreshCw size={16} aria-hidden /> Rafraichir
          </button>
          <button className="primary" type="button" onClick={() => runEndpoint("/api/nexus-game/admin/ai-server/jobs/run-due", "run-due")} disabled={!!runningKey}>
            <Play size={16} aria-hidden /> Jobs dus
          </button>
          <button className="danger" type="button" onClick={() => runEndpoint("/api/nexus-game/admin/ai-server/jobs/run-all", "run-all")} disabled={!!runningKey}>
            <Zap size={16} aria-hidden /> Tout lancer
          </button>
        </div>
      </section>

      <section className="metric-grid">
        <div className="metric-card"><span>Jobs declares</span><strong>{stats.total}</strong></div>
        <div className="metric-card"><span>A executer</span><strong>{stats.due}</strong></div>
        <div className="metric-card"><span>Echecs dernier run</span><strong>{stats.failed}</strong></div>
        <div className="metric-card"><span>Deterministes Go</span><strong>{stats.deterministic}</strong></div>
      </section>

      {results.length ? (
        <section className="panel">
          <h2>Derniere execution</h2>
          <div className="table-wrap">
            <table className="data-table">
              <thead><tr><th>Job</th><th>Statut</th><th>Traites</th><th>Crees</th><th>Maj</th><th>Skips</th><th>Resume</th></tr></thead>
              <tbody>
                {results.map((result) => (
                  <tr key={result.jobKey}>
                    <td>{result.jobKey}</td>
                    <td><span className={`status ${result.status}`}>{result.status}</span></td>
                    <td>{result.processed}</td>
                    <td>{result.createdCount}</td>
                    <td>{result.updatedCount}</td>
                    <td>{result.skippedCount}</td>
                    <td><code>{JSON.stringify(result.summary || {})}</code></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      ) : null}

      <section className="panel">
        <h2>Matrice des jobs IA</h2>
        <div className="table-wrap">
          <table className="data-table">
            <thead>
              <tr>
                <th>Job</th>
                <th>Frequence</th>
                <th>Mode IA</th>
                <th>Etat</th>
                <th>Dernier run</th>
                <th>Prochain</th>
                <th>Impact</th>
                <th>Action</th>
              </tr>
            </thead>
            <tbody>
              {jobs.map((job) => (
                <tr key={job.key}>
                  <td>
                    <strong>{job.name}</strong>
                    <small className="prewrap">{job.key}</small>
                    <p className="hint">{job.description}</p>
                  </td>
                  <td>{job.frequency}</td>
                  <td><span className="status">{job.generativeMode}</span></td>
                  <td><span className={statusClass(job.lastRun?.status, job.due)}>{job.due ? "due" : job.lastRun?.status || "jamais"}</span></td>
                  <td>
                    {formatDate(job.lastRun?.startedAt)}
                    {job.lastRun ? <small className="prewrap">{job.lastRun.triggerType} · {job.lastRun.durationMs}ms</small> : null}
                    {job.lastRun?.errorMessage ? <small className="bad">{job.lastRun.errorMessage}</small> : null}
                  </td>
                  <td>{formatDate(job.nextRunAt)}</td>
                  <td>
                    {job.lastRun ? (
                      <small className="prewrap">
                        p:{job.lastRun.processed} c:{job.lastRun.createdCount} u:{job.lastRun.updatedCount} s:{job.lastRun.skippedCount}
                      </small>
                    ) : "-"}
                  </td>
                  <td>
                    <button
                      className="secondary"
                      type="button"
                      onClick={() => runEndpoint(`/api/nexus-game/admin/ai-server/jobs/${encodeURIComponent(job.key)}/run`, job.key)}
                      disabled={!!runningKey}
                    >
                      <Activity size={15} aria-hidden /> {runningKey === job.key ? "Run" : "Lancer"}
                    </button>
                  </td>
                </tr>
              ))}
              {!jobs.length ? <tr><td colSpan={8}>Aucun job IA serveur disponible.</td></tr> : null}
            </tbody>
          </table>
        </div>
      </section>
    </AdminShell>
  );
}
