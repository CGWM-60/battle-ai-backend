"use client";

import { useEffect, useState } from "react";
import { AdminShell } from "../../../components/AdminShell";

const API_BASE = (process.env.NEXT_PUBLIC_NEXUS_API_BASE || "").replace(/\/$/, "");

export default function AIServerCitiesPage() {
  const [cities, setCities] = useState<any[]>([]);
  const [error, setError] = useState("");

  const load = async () => {
    setError("");
    try {
      const res = await fetch(`${API_BASE}/api/nexus-game/admin/ai-server/cities`, { credentials: "same-origin" });
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      setCities(data.cities || []);
    } catch (e: any) {
      setError(e.message || "Erreur villes IA");
    }
  };

  useEffect(() => { load(); }, []);

  const forceWorld = async () => {
    const id = window.prompt("World ID à initialiser ?");
    if (!id) return;
    const res = await fetch(`${API_BASE}/api/nexus-game/admin/ai-server/worlds/${id}/ensure-cities`, { method: "POST", credentials: "same-origin" });
    if (!res.ok) setError(await res.text());
    await load();
  };

  return (
    <AdminShell title="Villes IA" description="Bastions IA par continent, bonus contrôlés, statut et puissance.">
      <div style={{ marginBottom: 12 }}><button onClick={forceWorld}>Créer/compléter bastions d’un monde</button><button onClick={load} style={{ marginLeft: 8 }}>Rafraîchir</button></div>
      {error ? <div className="alert error">{error}</div> : null}
      <section className="panel">
        <table className="data-table">
          <thead><tr><th>Nom</th><th>Monde</th><th>Continent</th><th>Niveau</th><th>Power</th><th>Statut</th><th>Bonus</th></tr></thead>
          <tbody>
            {cities.map((c) => (
              <tr key={c.id}>
                <td>{c.name}</td><td>{c.worldId}</td><td>{c.continentId}</td><td>{c.level}</td><td>{c.power}</td><td>{c.status}</td>
                <td>{Math.round((c.productionBonus || 0) * 100)}% prod / {Math.round((c.trainingBonus || 0) * 100)}% train</td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </AdminShell>
  );
}
