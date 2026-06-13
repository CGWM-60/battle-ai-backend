"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { Activity, Coins, Play, RefreshCw, ShieldAlert } from "lucide-react";
import { AdminShell } from "../../components/AdminShell";

const API_BASE = (process.env.NEXT_PUBLIC_NEXUS_API_BASE || "").replace(/\/$/, "");

type JobView = {
  key: string;
  name: string;
  category: string;
  frequency: string;
  due: boolean;
  nextRunAt?: string;
  lastRun?: {
    status: string;
    startedAt: string;
    finishedAt?: string;
    processed: number;
    createdCount: number;
    updatedCount: number;
    skippedCount: number;
    errorMessage?: string;
  };
};

type GrantForm = {
  profileGamerId: string;
  resourceCode: string;
  amount: string;
  reason: string;
};

const emptyGrant: GrantForm = {
  profileGamerId: "",
  resourceCode: "credits",
  amount: "100",
  reason: "fiche de jour",
};

function formatDate(value?: string) {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString("fr-FR", { dateStyle: "short", timeStyle: "medium" });
}

function asArray(value: any) {
  return Array.isArray(value) ? value : [];
}

export default function AIServerDashboardPage() {
  const [dashboard, setDashboard] = useState<any>(null);
  const [jobs, setJobs] = useState<JobView[]>([]);
  const [attacks, setAttacks] = useState<any[]>([]);
  const [players, setPlayers] = useState<any[]>([]);
  const [resources, setResources] = useState<any[]>([]);
  const [grant, setGrant] = useState<GrantForm>(emptyGrant);
  const [result, setResult] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [running, setRunning] = useState("");
  const [refreshedAt, setRefreshedAt] = useState("");

  const dueJobs = useMemo(() => jobs.filter((job) => job.due), [jobs]);
  const failedJobs = useMemo(() => jobs.filter((job) => job.lastRun?.status === "failed"), [jobs]);

  const load = async () => {
    setError("");
    setLoading(true);
    try {
      const [dashboardRes, jobsRes, attacksRes, playersRes, resourcesRes] = await Promise.all([
        fetch(`${API_BASE}/api/nexus-game/admin/ai-server/dashboard`, { credentials: "same-origin" }),
        fetch(`${API_BASE}/api/nexus-game/admin/ai-server/jobs`, { credentials: "same-origin" }),
        fetch(`${API_BASE}/api/nexus-game/admin/ai-server/attacks`, { credentials: "same-origin" }),
        fetch(`${API_BASE}/api/nexus-game/admin/resources/players`, { credentials: "same-origin" }),
        fetch(`${API_BASE}/api/nexus-game/admin/resources/catalog`, { credentials: "same-origin" }),
      ]);
      for (const res of [dashboardRes, jobsRes, attacksRes, playersRes, resourcesRes]) {
        if (!res.ok) throw new Error(await res.text());
      }
      const dashboardData = await dashboardRes.json();
      const jobsData = await jobsRes.json();
      const attacksData = await attacksRes.json();
      const playersData = await playersRes.json();
      const resourcesData = await resourcesRes.json();
      setDashboard(dashboardData);
      setJobs(jobsData.jobs || []);
      setAttacks(attacksData.attacks || []);
      setPlayers(playersData.players || []);
      setResources(resourcesData.resources || []);
      setRefreshedAt(new Date().toISOString());
    } catch (e: any) {
      setError(e.message || "Erreur chargement console IA serveur");
    } finally {
      setLoading(false);
    }
  };

  const runJob = async (jobKey: string, label: string) => {
    setRunning(label);
    setError("");
    setResult("");
    try {
      const res = await fetch(`${API_BASE}/api/nexus-game/admin/ai-server/jobs/${jobKey}/run`, {
        method: "POST",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json" },
        body: "{}",
      });
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      setResult(`${label}: ${data.result?.status || "termine"} (${data.result?.createdCount || 0} crees, ${data.result?.updatedCount || 0} modifies)`);
      await load();
    } catch (e: any) {
      setError(e.message || `Erreur execution ${label}`);
    } finally {
      setRunning("");
    }
  };

  const runDue = async () => {
    setRunning("jobs dus");
    setError("");
    setResult("");
    try {
      const res = await fetch(`${API_BASE}/api/nexus-game/admin/ai-server/jobs/run-due`, {
        method: "POST",
        credentials: "same-origin",
      });
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      setResult(`Jobs dus executes: ${asArray(data.results).length}`);
      await load();
    } catch (e: any) {
      setError(e.message || "Erreur execution jobs dus");
    } finally {
      setRunning("");
    }
  };

  const submitGrant = async (event: React.FormEvent) => {
    event.preventDefault();
    setRunning("grant");
    setError("");
    setResult("");
    try {
      const res = await fetch(`${API_BASE}/api/nexus-game/admin/resources/grant`, {
        method: "POST",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          profileGamerId: Number(grant.profileGamerId),
          resourceCode: grant.resourceCode,
          amount: Number(grant.amount),
          reason: grant.reason,
        }),
      });
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      setResult(`Ressource appliquee: ${data.resource?.resourceCode} = ${data.resource?.amount}`);
      await load();
    } catch (e: any) {
      setError(e.message || "Erreur ajout ressource");
    } finally {
      setRunning("");
    }
  };

  useEffect(() => {
    load();
    const id = window.setInterval(load, 30000);
    return () => window.clearInterval(id);
  }, []);

  const cards = [
    ["Bastions IA", dashboard?.citiesCount],
    ["Conflits IA", attacks.length],
    ["Jobs dus", dueJobs.length],
    ["Jobs en erreur", failedJobs.length],
    ["Broadcasts", dashboard?.broadcastsCount],
    ["Appels 24h", dashboard?.callsLast24h],
  ];

  return (
    <AdminShell title="IA serveur Nexus" description="Console unique pour voir les crons, declencher les actions IA, suivre les conflits et ajouter des ressources joueur.">
      <div style={{ display: "flex", gap: 8, flexWrap: "wrap", marginBottom: 16 }}>
        <button onClick={load} disabled={loading}>
          <RefreshCw size={16} aria-hidden /> Rafraichir
        </button>
        <button onClick={runDue} disabled={!!running}>
          <Play size={16} aria-hidden /> Lancer les jobs dus
        </button>
        <button onClick={() => runJob("server_ai_attack_scheduler_job", "creation conflits IA")} disabled={!!running}>
          <ShieldAlert size={16} aria-hidden /> Creer conflits IA
        </button>
        <button onClick={() => runJob("server_ai_sabotage_job", "sabotages IA")} disabled={!!running}>
          <Activity size={16} aria-hidden /> Creer sabotages
        </button>
        <button onClick={() => runJob("server_ai_daily_broadcast_job", "broadcast quotidien")} disabled={!!running}>
          <Play size={16} aria-hidden /> Broadcast IA
        </button>
        <Link href="/nexus/ai-server/jobs"><button>Detail jobs</button></Link>
        <Link href="/nexus/ai-server/attacks"><button>Detail conflits</button></Link>
        <Link href="/nexus/ai-server/logs"><button>Logs & couts</button></Link>
      </div>

      {error ? <div className="alert error">{error}</div> : null}
      {result ? <div className="alert ok">{result}</div> : null}
      <p className="muted">Dernier refresh: {formatDate(refreshedAt)} {running ? `- action en cours: ${running}` : ""}</p>

      <section className="panel">
        <div className="metric-grid">
          {cards.map(([label, value]) => (
            <div className="metric-card" key={label}>
              <span>{label}</span>
              <strong>{value ?? "-"}</strong>
            </div>
          ))}
        </div>
      </section>

      <section className="panel">
        <h2>Fiche de jour: ressources joueur</h2>
        <form onSubmit={submitGrant} style={{ display: "grid", gridTemplateColumns: "repeat(5, minmax(140px, 1fr))", gap: 10, alignItems: "end" }}>
          <label>
            Joueur
            <select value={grant.profileGamerId} onChange={(e) => setGrant({ ...grant, profileGamerId: e.target.value })} required>
              <option value="">Choisir</option>
              {players.map((player) => (
                <option value={player.id} key={player.id}>
                  #{player.id} {player.pseudo || player.city_name || player.cityName || `user ${player.user_id}`}
                </option>
              ))}
            </select>
          </label>
          <label>
            Ressource
            <select value={grant.resourceCode} onChange={(e) => setGrant({ ...grant, resourceCode: e.target.value })}>
              {resources.map((resource) => (
                <option value={resource.code} key={resource.code}>{resource.code}</option>
              ))}
            </select>
          </label>
          <label>
            Montant
            <input type="number" value={grant.amount} onChange={(e) => setGrant({ ...grant, amount: e.target.value })} required />
          </label>
          <label>
            Raison
            <input value={grant.reason} onChange={(e) => setGrant({ ...grant, reason: e.target.value })} />
          </label>
          <button type="submit" disabled={running === "grant"}>
            <Coins size={16} aria-hidden /> Appliquer
          </button>
        </form>
      </section>

      <section className="panel">
        <h2>Crons IA serveur</h2>
        <table className="data-table">
          <thead><tr><th>Job</th><th>Frequence</th><th>Etat</th><th>Dernier run</th><th>Prochain</th><th>Action</th></tr></thead>
          <tbody>
            {jobs.map((job) => (
              <tr key={job.key}>
                <td><strong>{job.name}</strong><br /><span className="muted">{job.category}</span></td>
                <td>{job.frequency}</td>
                <td><span className={`status ${job.due ? "started" : job.lastRun?.status || "skipped"}`}>{job.due ? "du" : job.lastRun?.status || "jamais lance"}</span></td>
                <td>{formatDate(job.lastRun?.startedAt)}<br /><span className="muted">{job.lastRun?.errorMessage || `${job.lastRun?.createdCount || 0} crees, ${job.lastRun?.updatedCount || 0} modifies`}</span></td>
                <td>{formatDate(job.nextRunAt)}</td>
                <td><button onClick={() => runJob(job.key, job.name)} disabled={!!running}>Run</button></td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      <section className="panel">
        <h2>Conflits IA visibles</h2>
        <table className="data-table">
          <thead><tr><th>ID</th><th>Cible</th><th>Type</th><th>Puissance</th><th>Statut</th><th>Resolution</th></tr></thead>
          <tbody>
            {attacks.map((attack) => (
              <tr key={attack.id}>
                <td>{attack.id}</td>
                <td>{attack.targetUserId || attack.targetCityId || "-"}</td>
                <td>{attack.attackType || "-"}</td>
                <td>{attack.attackPower || 0}/{attack.defensePower || 0}</td>
                <td><span className={`status ${attack.status || "unknown"}`}>{attack.status || "-"}</span></td>
                <td>{attack.result || "-"}</td>
              </tr>
            ))}
            {attacks.length === 0 ? <tr><td colSpan={6}>Aucun conflit IA. Utilise "Creer conflits IA" pour lancer le job de planification.</td></tr> : null}
          </tbody>
        </table>
      </section>
    </AdminShell>
  );
}
