"use client";

import { useEffect, useState } from "react";
import { AdminShell } from "../../../components/AdminShell";

const API_BASE = (process.env.NEXT_PUBLIC_NEXUS_API_BASE || "").replace(/\/$/, "");

export default function AIServerBroadcastsPage() {
  const [items, setItems] = useState<any[]>([]);

  const load = async () => {
    const res = await fetch(`${API_BASE}/api/nexus-game/admin/ai-server/broadcasts`, { credentials: "same-origin" });
    if (res.ok) setItems((await res.json()).broadcasts || []);
  };

  useEffect(() => { load(); }, []);

  const generate = async () => {
    await fetch(`${API_BASE}/api/nexus-game/admin/ai-server/broadcasts/generate`, { method: "POST", credentials: "same-origin" });
    await load();
  };

  const publish = async (id: number) => {
    await fetch(`${API_BASE}/api/nexus-game/admin/ai-server/broadcasts/${id}/publish`, { method: "POST", credentials: "same-origin" });
    await load();
  };

  return (
    <AdminShell title="Messages quotidiens IA" description="Transmissions menaçantes générées, modifiables et publiables.">
      <button onClick={generate}>Générer broadcast</button><button onClick={load} style={{ marginLeft: 8 }}>Rafraîchir</button>
      <section className="panel">
        <table className="data-table">
          <thead><tr><th>Date</th><th>Titre</th><th>Menace</th><th>Statut</th><th>Message</th><th>Actions</th></tr></thead>
          <tbody>{items.map((b) => <tr key={b.id}><td>{b.date}</td><td>{b.title}</td><td>{b.threatLevel}</td><td>{b.status}</td><td>{b.message}</td><td><button onClick={() => publish(b.id)}>Publier</button></td></tr>)}</tbody>
        </table>
      </section>
    </AdminShell>
  );
}
