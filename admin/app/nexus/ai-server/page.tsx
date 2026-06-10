"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { AdminShell } from "../../components/AdminShell";

const API_BASE = (process.env.NEXT_PUBLIC_NEXUS_API_BASE || "").replace(/\/$/, "");

export default function AIServerDashboardPage() {
  const [data, setData] = useState<any>(null);
  const [error, setError] = useState("");

  const load = async () => {
    setError("");
    try {
      const res = await fetch(`${API_BASE}/api/nexus-game/admin/ai-server/dashboard`, { credentials: "same-origin" });
      if (!res.ok) throw new Error(await res.text());
      setData(await res.json());
    } catch (e: any) {
      setError(e.message || "Erreur chargement IA serveur");
    }
  };

  useEffect(() => { load(); }, []);

  const cards = [
    ["Bastions IA", data?.citiesCount],
    ["Attaques", data?.attacksCount],
    ["Events à revoir", data?.eventsToReview],
    ["Broadcasts", data?.broadcastsCount],
    ["Appels 24h", data?.callsLast24h],
  ];

  return (
    <AdminShell title="IA serveur Nexus" description="Pilotage du Noyau IA, bastions, pression, coûts et actions contrôlées.">
      <div style={{ display: "flex", gap: 8, flexWrap: "wrap", marginBottom: 16 }}>
        <Link href="/nexus/ai-server/cities"><button>Villes IA</button></Link>
        <Link href="/nexus/ai-server/attacks"><button>Attaques</button></Link>
        <Link href="/nexus/ai-server/memory"><button>Mémoire</button></Link>
        <Link href="/nexus/ai-server/prompts"><button>Prompts</button></Link>
        <Link href="/nexus/ai-server/logs"><button>Logs & coûts</button></Link>
        <Link href="/nexus/ai-server/broadcasts"><button>Broadcasts</button></Link>
        <Link href="/nexus/seasonal-events"><button>Événements saisonniers</button></Link>
        <button onClick={load}>Rafraîchir</button>
      </div>
      {error ? <div className="alert error">{error}</div> : null}
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
        <h2>Pression par continent</h2>
        <table className="data-table">
          <thead><tr><th>Monde</th><th>Continent</th><th>Niveau</th><th>Statut</th></tr></thead>
          <tbody>
            {(data?.pressures || []).map((p: any) => (
              <tr key={p.id}><td>{p.worldId}</td><td>{p.continentId}</td><td>{p.level}</td><td>{p.status}</td></tr>
            ))}
            {(!data?.pressures || data.pressures.length === 0) ? <tr><td colSpan={4}>Aucune pression enregistrée.</td></tr> : null}
          </tbody>
        </table>
      </section>
    </AdminShell>
  );
}
