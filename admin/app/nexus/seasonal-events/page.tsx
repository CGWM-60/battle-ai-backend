"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { AdminShell } from "../../components/AdminShell";

const API_BASE = (process.env.NEXT_PUBLIC_NEXUS_API_BASE || "").replace(/\/$/, "");

export default function SeasonalEventsPage() {
  const [events, setEvents] = useState<any[]>([]);

  const load = async () => {
    const res = await fetch(`${API_BASE}/api/nexus-game/admin/seasonal-events`, { credentials: "same-origin" });
    if (res.ok) setEvents((await res.json()).events || []);
  };

  useEffect(() => { load(); }, []);

  const propose = async () => {
    await fetch(`${API_BASE}/api/nexus-game/admin/seasonal-events/propose-by-ai`, { method: "POST", credentials: "same-origin" });
    await load();
  };

  return (
    <AdminShell title="Événements saisonniers" description="Propositions IA, validation admin, planification, activation et archivage.">
      <button onClick={propose}>Proposer par IA</button><button onClick={load} style={{ marginLeft: 8 }}>Rafraîchir</button>
      <section className="panel">
        <table className="data-table">
          <thead><tr><th>Titre</th><th>Type</th><th>Statut</th><th>Dates</th><th>Actions</th></tr></thead>
          <tbody>{events.map((e) => <tr key={e.id}><td>{e.title}</td><td>{e.eventType}</td><td>{e.status}</td><td>{e.startsAt || "-"} → {e.endsAt || "-"}</td><td><Link href={`/nexus/seasonal-events/detail?id=${e.id}`}>Détail</Link></td></tr>)}</tbody>
        </table>
      </section>
    </AdminShell>
  );
}
