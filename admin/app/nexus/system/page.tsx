"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { Activity, Coins, Play, RefreshCw, Shield, Swords, Users } from "lucide-react";
import { AdminShell } from "../../components/AdminShell";

type GrantResourceForm = {
  profileGamerId: string;
  resourceCode: string;
  amount: string;
  reason: string;
};

type GrantUnitForm = {
  profileGamerId: string;
  unitCode: string;
  quantity: string;
  reason: string;
};

const emptyResource: GrantResourceForm = { profileGamerId: "", resourceCode: "credits", amount: "100", reason: "fiche de jour" };
const emptyUnit: GrantUnitForm = { profileGamerId: "", unitCode: "", quantity: "1", reason: "admin" };

function formatDate(value?: string) {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString("fr-FR", { dateStyle: "short", timeStyle: "medium" });
}

function asList(value: any): any[] {
  return Array.isArray(value) ? value : [];
}

export default function NexusSystemPage() {
  const [data, setData] = useState<any>(null);
  const [resourceForm, setResourceForm] = useState<GrantResourceForm>(emptyResource);
  const [unitForm, setUnitForm] = useState<GrantUnitForm>(emptyUnit);
  const [loading, setLoading] = useState(false);
  const [running, setRunning] = useState("");
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [refreshedAt, setRefreshedAt] = useState("");

  const snapshot = data?.snapshot || {};
  const players = asList(snapshot.players);
  const units = asList(snapshot.unitCatalog);
  const queues = asList(snapshot.trainingQueues);
  const formations = asList(snapshot.formations);
  const reports = asList(snapshot.combatReports);
  const jobs = asList(data?.jobs);
  const modules = asList(snapshot.modules);

  const stats = useMemo(() => {
    return [
      ["Joueurs Nexus", players.length],
      ["Unites catalogue", units.length],
      ["Queues training", queues.length],
      ["Formations", formations.length],
      ["Rapports combat", reports.length],
      ["Jobs IA", jobs.length],
    ];
  }, [players.length, units.length, queues.length, formations.length, reports.length, jobs.length]);

  const load = async () => {
    setLoading(true);
    setError("");
    try {
      const res = await fetch("/admin/api/nexus-system", { credentials: "same-origin" });
      if (!res.ok) throw new Error(await res.text());
      setData(await res.json());
      setRefreshedAt(new Date().toISOString());
    } catch (e: any) {
      setError(e.message || "Erreur chargement gestion systeme");
    } finally {
      setLoading(false);
    }
  };

  const post = async (path: string, body?: any, label = "action") => {
    setRunning(label);
    setError("");
    setMessage("");
    try {
      const res = await fetch(path, {
        method: "POST",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json" },
        body: body ? JSON.stringify(body) : "{}",
      });
      if (!res.ok) throw new Error(await res.text());
      await res.json().catch(() => null);
      setMessage(`${label}: OK`);
      await load();
    } catch (e: any) {
      setError(e.message || `Erreur ${label}`);
    } finally {
      setRunning("");
    }
  };

  useEffect(() => {
    load();
  }, []);

  return (
    <AdminShell
      title="Gestion systeme Nexus"
      description="Vue centrale backend: modules, CRUD, crons IA, joueurs, ressources, unites, queues et formations."
    >
      <div style={{ display: "flex", gap: 8, flexWrap: "wrap", marginBottom: 16 }}>
        <button onClick={load} disabled={loading}><RefreshCw size={16} aria-hidden /> Rafraichir</button>
        <button onClick={() => post("/admin/api/nexus-system/ai/jobs/run-due", {}, "jobs IA dus")} disabled={!!running}><Play size={16} aria-hidden /> Lancer jobs IA dus</button>
        <Link href="/nexus/mmo/buildings"><button>CRUD batiments</button></Link>
        <Link href="/nexus/mmo/units"><button>CRUD unites</button></Link>
        <Link href="/nexus/mmo/research"><button>CRUD recherches</button></Link>
        <Link href="/nexus/ai-server/jobs"><button>Crons IA detailles</button></Link>
      </div>

      {error ? <div className="alert error">{error}</div> : null}
      {message ? <div className="alert ok">{message}</div> : null}
      <p className="muted">Dernier refresh: {formatDate(refreshedAt)} {running ? `- action en cours: ${running}` : ""}</p>

      <section className="panel">
        <div className="metric-grid">
          {stats.map(([label, value]) => (
            <div className="metric-card" key={label}>
              <span>{label}</span>
              <strong>{value}</strong>
            </div>
          ))}
        </div>
      </section>

      <section className="panel">
        <h2>Actions joueur</h2>
        <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(320px, 1fr))", gap: 16 }}>
          <form
            onSubmit={(event) => {
              event.preventDefault();
              post("/admin/api/nexus-system/resources/grant", {
                profileGamerId: Number(resourceForm.profileGamerId),
                resourceCode: resourceForm.resourceCode,
                amount: Number(resourceForm.amount),
                reason: resourceForm.reason,
              }, "ressource joueur");
            }}
          >
            <h3><Coins size={16} aria-hidden /> Ajouter / debiter ressource</h3>
            <label>Joueur
              <select value={resourceForm.profileGamerId} onChange={(e) => setResourceForm({ ...resourceForm, profileGamerId: e.target.value })} required>
                <option value="">Choisir</option>
                {players.map((p: any) => <option value={p.id} key={p.id}>#{p.id} {p.pseudo || p.city_name || p.cityName || `user ${p.user_id}`}</option>)}
              </select>
            </label>
            <label>Ressource
              <input value={resourceForm.resourceCode} onChange={(e) => setResourceForm({ ...resourceForm, resourceCode: e.target.value })} />
            </label>
            <label>Montant
              <input type="number" value={resourceForm.amount} onChange={(e) => setResourceForm({ ...resourceForm, amount: e.target.value })} />
            </label>
            <label>Raison
              <input value={resourceForm.reason} onChange={(e) => setResourceForm({ ...resourceForm, reason: e.target.value })} />
            </label>
            <button type="submit" disabled={!!running}>Appliquer ressource</button>
          </form>

          <form
            onSubmit={(event) => {
              event.preventDefault();
              post("/admin/api/nexus-system/units/grant", {
                profileGamerId: Number(unitForm.profileGamerId),
                unitCode: unitForm.unitCode,
                quantity: Number(unitForm.quantity),
                reason: unitForm.reason,
              }, "unite joueur");
            }}
          >
            <h3><Swords size={16} aria-hidden /> Ajouter / retirer unites</h3>
            <label>Joueur
              <select value={unitForm.profileGamerId} onChange={(e) => setUnitForm({ ...unitForm, profileGamerId: e.target.value })} required>
                <option value="">Choisir</option>
                {players.map((p: any) => <option value={p.id} key={p.id}>#{p.id} {p.pseudo || p.city_name || p.cityName || `user ${p.user_id}`}</option>)}
              </select>
            </label>
            <label>Unite
              <select value={unitForm.unitCode} onChange={(e) => setUnitForm({ ...unitForm, unitCode: e.target.value })} required>
                <option value="">Choisir</option>
                {units.map((u: any) => <option value={u.contentId} key={u.contentId}>{u.contentId}</option>)}
              </select>
            </label>
            <label>Quantite
              <input type="number" value={unitForm.quantity} onChange={(e) => setUnitForm({ ...unitForm, quantity: e.target.value })} />
            </label>
            <label>Raison
              <input value={unitForm.reason} onChange={(e) => setUnitForm({ ...unitForm, reason: e.target.value })} />
            </label>
            <button type="submit" disabled={!!running}>Appliquer unites</button>
          </form>
        </div>
      </section>

      <section className="panel">
        <h2><Shield size={18} aria-hidden /> Modules et CRUD</h2>
        <table className="data-table">
          <thead><tr><th>Module</th><th>Endpoint</th><th>CRUD</th></tr></thead>
          <tbody>
            {modules.map((m: any) => <tr key={m.key}><td>{m.label}</td><td><code>{m.endpoint}</code></td><td>{m.crud ? "oui" : "lecture/action"}</td></tr>)}
          </tbody>
        </table>
      </section>

      <section className="panel">
        <h2><Activity size={18} aria-hidden /> Jobs IA serveur</h2>
        <table className="data-table">
          <thead><tr><th>Job</th><th>Frequence</th><th>Etat</th><th>Dernier run</th><th>Prochain</th></tr></thead>
          <tbody>
            {jobs.map((job: any) => (
              <tr key={job.key}>
                <td>{job.name}<br /><span className="muted">{job.key}</span></td>
                <td>{job.frequency}</td>
                <td><span className={`status ${job.due ? "started" : job.lastRun?.status || "skipped"}`}>{job.due ? "du" : job.lastRun?.status || "jamais lance"}</span></td>
                <td>{formatDate(job.lastRun?.startedAt)}</td>
                <td>{formatDate(job.nextRunAt)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      <section className="panel">
        <h2><Users size={18} aria-hidden /> Training queues et formations</h2>
        <table className="data-table">
          <thead><tr><th>Queue</th><th>Joueur</th><th>Unite</th><th>Qte</th><th>Statut</th><th>Fin</th></tr></thead>
          <tbody>
            {queues.map((q: any) => <tr key={q.id}><td>#{q.id}</td><td>{q.profileGamerId}</td><td>{q.unitCode}</td><td>{q.quantity}</td><td>{q.status}</td><td>{formatDate(q.completedAt)}</td></tr>)}
            {queues.length === 0 ? <tr><td colSpan={6}>Aucune queue.</td></tr> : null}
          </tbody>
        </table>
        <table className="data-table" style={{ marginTop: 16 }}>
          <thead><tr><th>Formation</th><th>Joueur</th><th>Type</th><th>Power</th><th>Statut</th></tr></thead>
          <tbody>
            {formations.map((f: any) => <tr key={f.id}><td>#{f.id} {f.name}</td><td>{f.profileGamerId}</td><td>{f.type}</td><td>{f.totalPower}</td><td>{f.status}</td></tr>)}
            {formations.length === 0 ? <tr><td colSpan={5}>Aucune formation encore creee.</td></tr> : null}
          </tbody>
        </table>
      </section>
    </AdminShell>
  );
}
