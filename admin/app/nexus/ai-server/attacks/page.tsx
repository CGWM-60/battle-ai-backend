"use client";

import { useEffect, useState } from "react";
import { AdminShell } from "../../../components/AdminShell";

const API_BASE = (process.env.NEXT_PUBLIC_NEXUS_API_BASE || "").replace(/\/$/, "");

export default function AIServerAttacksPage() {
  const [attacks, setAttacks] = useState<any[]>([]);
  const [error, setError] = useState("");

  const load = async () => {
    setError("");
    try {
      const res = await fetch(`${API_BASE}/api/nexus-game/admin/ai-server/attacks`, { credentials: "same-origin" });
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      setAttacks(data.attacks || []);
    } catch (e: any) {
      setError(e.message || "Erreur attaques IA");
    }
  };

  useEffect(() => { load(); }, []);

  const action = async (id: number, path: string) => {
    const res = await fetch(`${API_BASE}/api/nexus-game/admin/ai-server/attacks/${id}/${path}`, { method: "POST", credentials: "same-origin" });
    if (!res.ok) setError(await res.text());
    await load();
  };

  return (
    <AdminShell title="Attaques IA" description="Attaques scalées, anti-frustration, annulation et résolution contrôlée.">
      <button onClick={load}>Rafraîchir</button>
      {error ? <div className="alert error">{error}</div> : null}
      <section className="panel">
        <table className="data-table">
          <thead><tr><th>ID</th><th>Cible</th><th>Type</th><th>Power</th><th>Statut</th><th>Résultat</th><th>Actions</th></tr></thead>
          <tbody>
            {attacks.map((a) => (
              <tr key={a.id}>
                <td>{a.id}</td><td>{a.targetUserId || a.targetCityId}</td><td>{a.attackType}</td><td>{a.attackPower}/{a.defensePower}</td><td>{a.status}</td><td>{a.result}</td>
                <td><button onClick={() => action(a.id, "cancel")}>Annuler</button><button onClick={() => action(a.id, "resolve")} style={{ marginLeft: 6 }}>Résoudre</button></td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </AdminShell>
  );
}
