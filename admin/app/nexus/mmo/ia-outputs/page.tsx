"use client";

import { useState, useEffect } from "react";
import { AdminShell } from "../../../components/AdminShell";

export default function IAOutputsPage() {
  const [outputs, setOutputs] = useState<any[]>([]);
  const [worlds, setWorlds] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedWorld, setSelectedWorld] = useState("all");

  const fetchData = async () => {
    setLoading(true);
    setError(null);
    try {
      const [outRes, worldRes] = await Promise.all([
        fetch("/api/nexus-game/ai-outputs", { credentials: "same-origin" }),
        fetch("/api/nexus-game/worlds", { credentials: "same-origin" }),
      ]);
      if (outRes.ok) {
        const oData = await outRes.json();
        setOutputs(oData.outputs || []);
      }
      if (worldRes.ok) {
        const wData = await worldRes.json();
        setWorlds(wData.worlds || []);
      }
    } catch (e: any) {
      setError(e.message || "Erreur de chargement");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  const filtered = selectedWorld === "all"
    ? outputs
    : outputs.filter((o: any) => String(o.world_id) === selectedWorld || String(o.world) === selectedWorld);

  const exportCSV = () => {
    if (filtered.length === 0) return;
    const headers = ["timestamp", "feature", "world_id", "linked_type", "linked_id", "prompt_version", "tokens_in", "tokens_out", "latency_ms", "status", "output"];
    const rows = [headers.join(",")];
    filtered.forEach((o: any) => {
      const outStr = JSON.stringify(o.output || o).replace(/"/g, '""');
      const row = [
        o.timestamp || "",
        o.feature || "",
        o.world_id || "",
        o.linked_type || "",
        o.linked_id || "",
        o.prompt_version || "",
        o.tokens_in || "",
        o.tokens_out || "",
        o.latency_ms || "",
        o.status || "",
        `"${outStr}"`
      ];
      rows.push(row.join(","));
    });
    const csv = rows.join("\n");
    const blob = new Blob([csv], { type: "text/csv;charset=utf-8;" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "ia_server_outputs.csv";
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <AdminShell title="IA Outputs" description="Historique textuel des générations de l'IA serveur (persisté DB + Redis)">
      <button onClick={() => window.location.href = '/nexus/mmo'} style={{ marginBottom: 16 }}>← Retour à Nexus MMO</button>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <h2>Outputs IA Serveur ({filtered.length}) - Historique cross-sessions</h2>
        <div>
          <select value={selectedWorld} onChange={e => setSelectedWorld(e.target.value)} style={{ padding: 8, marginRight: 8 }}>
            <option value="all">Tous les mondes</option>
            {worlds.map((w: any, i: number) => (
              <option key={i} value={String(w.id)}>{w.name || `Monde ${w.id}`}</option>
            ))}
          </select>
          <button onClick={fetchData} style={{ marginRight: 8 }}>Recharger</button>
          <button onClick={exportCSV}>Exporter CSV</button>
        </div>
      </div>

      {loading ? <p>Chargement...</p> : error ? <p style={{color:'red'}}>{error}</p> : (
        <table className="data-table" style={{ width: '100%', borderCollapse: 'collapse' }}>
          <thead>
            <tr>
              <th style={{ textAlign: 'left', padding: 8 }}>Timestamp</th>
              <th style={{ textAlign: 'left', padding: 8 }}>Feature</th>
              <th style={{ textAlign: 'left', padding: 8 }}>Monde</th>
              <th style={{ textAlign: 'left', padding: 8 }}>Prompt Ver.</th>
              <th style={{ textAlign: 'right', padding: 8 }}>Tokens</th>
              <th style={{ textAlign: 'right', padding: 8 }}>Latency</th>
              <th style={{ textAlign: 'left', padding: 8 }}>Output (textuel)</th>
            </tr>
          </thead>
          <tbody>
            {filtered.length === 0 && (
              <tr><td colSpan={7} style={{ padding: 8, color: '#64748b' }}>Aucun output IA. Générez via les outils (events, seeds, lore...).</td></tr>
            )}
            {filtered.map((o: any, i: number) => (
              <tr key={i} style={{ borderTop: '1px solid #334155' }}>
                <td style={{ padding: 8, fontSize: 12 }}>{o.timestamp}</td>
                <td style={{ padding: 8 }}>{o.feature}</td>
                <td style={{ padding: 8 }}>{o.world_id || o.world}</td>
                <td style={{ padding: 8 }}>{o.prompt_version}</td>
                <td style={{ padding: 8, textAlign: 'right' }}>{o.tokens_in}/{o.tokens_out}</td>
                <td style={{ padding: 8, textAlign: 'right' }}>{o.latency_ms}ms</td>
                <td style={{ padding: 8 }}>
                  <pre style={{ fontSize: 11, whiteSpace: 'pre-wrap', maxHeight: 120, overflow: 'auto', background: '#0f172a', padding: 4, borderRadius: 4 }}>
                    {typeof o.output === 'string' ? o.output : JSON.stringify(o.output || o, null, 2)}
                  </pre>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
      <p style={{ fontSize: 12, color: '#64748b', marginTop: 12 }}>Données de la table ai_outputs (GORM) + Redis cache. Filtre + export CSV inclus.</p>
    </AdminShell>
  );
}
