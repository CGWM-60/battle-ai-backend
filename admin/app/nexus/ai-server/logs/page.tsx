"use client";

import { useEffect, useState } from "react";
import { AdminShell } from "../../../components/AdminShell";

const API_BASE = (process.env.NEXT_PUBLIC_NEXUS_API_BASE || "").replace(/\/$/, "");

export default function AIServerLogsPage() {
  const [logs, setLogs] = useState<any[]>([]);
  const [costs, setCosts] = useState<any>(null);

  const load = async () => {
    const [lRes, cRes] = await Promise.all([
      fetch(`${API_BASE}/api/nexus-game/admin/ai-server/call-logs`, { credentials: "same-origin" }),
      fetch(`${API_BASE}/api/nexus-game/admin/ai-server/costs`, { credentials: "same-origin" }),
    ]);
    if (lRes.ok) setLogs((await lRes.json()).logs || []);
    if (cRes.ok) setCosts(await cRes.json());
  };

  useEffect(() => { load(); }, []);

  return (
    <AdminShell title="Logs et coûts IA" description="Appels IA serveur, tokens, coût estimé, latence et export manuel.">
      <button onClick={load}>Rafraîchir</button>
      <section className="panel"><h2>Coûts 24h</h2><pre>{JSON.stringify(costs, null, 2)}</pre></section>
      <section className="panel">
        <table className="data-table">
          <thead><tr><th>Date</th><th>Feature</th><th>Prompt</th><th>Tokens</th><th>Coût</th><th>Latence</th><th>Statut</th></tr></thead>
          <tbody>{logs.map((l) => <tr key={l.id}><td>{l.createdAt}</td><td>{l.feature}</td><td>{l.promptKey}</td><td>{l.tokensIn}/{l.tokensOut}</td><td>{l.costEstimate}</td><td>{l.latencyMs}ms</td><td>{l.status}</td></tr>)}</tbody>
        </table>
      </section>
    </AdminShell>
  );
}
