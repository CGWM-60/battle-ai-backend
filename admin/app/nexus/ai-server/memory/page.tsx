"use client";

import { useEffect, useState } from "react";
import { AdminShell } from "../../../components/AdminShell";

const API_BASE = (process.env.NEXT_PUBLIC_NEXUS_API_BASE || "").replace(/\/$/, "");

export default function AIServerMemoryPage() {
  const [memory, setMemory] = useState<any[]>([]);
  const [playerMemory, setPlayerMemory] = useState<any[]>([]);
  const [error, setError] = useState("");

  const load = async () => {
    setError("");
    try {
      const [mRes, pRes] = await Promise.all([
        fetch(`${API_BASE}/api/nexus-game/admin/ai-server/memory`, { credentials: "same-origin" }),
        fetch(`${API_BASE}/api/nexus-game/admin/ai-server/player-memory`, { credentials: "same-origin" }),
      ]);
      if (mRes.ok) setMemory((await mRes.json()).memory || []);
      if (pRes.ok) setPlayerMemory((await pRes.json()).playerMemory || []);
    } catch (e: any) {
      setError(e.message || "Erreur mémoire IA");
    }
  };

  useEffect(() => { load(); }, []);

  const del = async (id: number) => {
    await fetch(`${API_BASE}/api/nexus-game/admin/ai-server/player-memory/${id}`, { method: "DELETE", credentials: "same-origin" });
    await load();
  };

  return (
    <AdminShell title="Mémoire IA" description="Mémoire globale et mémoire joueur du Noyau IA.">
      <button onClick={load}>Rafraîchir</button>
      {error ? <div className="alert error">{error}</div> : null}
      <section className="panel"><h2>Mémoire globale</h2><pre>{JSON.stringify(memory, null, 2)}</pre></section>
      <section className="panel">
        <h2>Mémoire joueur</h2>
        <table className="data-table">
          <thead><tr><th>User</th><th>Style</th><th>Threat</th><th>Rivalité</th><th>Actions</th></tr></thead>
          <tbody>{playerMemory.map((m) => <tr key={m.id}><td>{m.userId}</td><td>{m.playerStyle}</td><td>{m.threatScore}</td><td>{m.rivalryLevel}</td><td><button onClick={() => del(m.id)}>Supprimer</button></td></tr>)}</tbody>
        </table>
      </section>
    </AdminShell>
  );
}
