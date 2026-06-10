"use client";

import { useEffect, useState } from "react";
import { AdminShell } from "../../../components/AdminShell";

const API_BASE = (process.env.NEXT_PUBLIC_NEXUS_API_BASE || "").replace(/\/$/, "");

export default function AIServerPromptsPage() {
  const [prompts, setPrompts] = useState<any[]>([]);
  const [error, setError] = useState("");
  const [test, setTest] = useState<any>(null);

  const load = async () => {
    setError("");
    try {
      const res = await fetch(`${API_BASE}/api/nexus-game/admin/ai-server/prompts`, { credentials: "same-origin" });
      if (!res.ok) throw new Error(await res.text());
      setPrompts((await res.json()).prompts || []);
    } catch (e: any) {
      setError(e.message || "Erreur prompts");
    }
  };

  useEffect(() => { load(); }, []);

  const seed = async () => {
    await fetch(`${API_BASE}/api/nexus-game/admin/ai-server/prompts/seed`, { method: "POST", credentials: "same-origin" });
    await load();
  };

  const runTest = async (id: number) => {
    const res = await fetch(`${API_BASE}/api/nexus-game/admin/ai-server/prompts/${id}/test`, { method: "POST", credentials: "same-origin" });
    setTest(res.ok ? await res.json() : await res.text());
  };

  return (
    <AdminShell title="Prompts IA serveur" description="Prompts versionnés, classes modèle, limites tokens, schémas et test.">
      <div style={{ marginBottom: 12 }}><button onClick={seed}>Seeder prompts spec</button><button onClick={load} style={{ marginLeft: 8 }}>Rafraîchir</button></div>
      {error ? <div className="alert error">{error}</div> : null}
      {test ? <section className="panel"><h2>Résultat test</h2><pre>{JSON.stringify(test, null, 2)}</pre></section> : null}
      <section className="panel">
        <table className="data-table">
          <thead><tr><th>Prompt</th><th>Domaine</th><th>Classe</th><th>Tokens</th><th>Actions</th></tr></thead>
          <tbody>
            {prompts.map((p) => (
              <tr key={p.id}>
                <td>{p.prompt_id || p.promptKey} @{p.version}</td><td>{p.domain}</td><td>{p.model_class}</td><td>{p.max_tokens_in}/{p.max_tokens_out}</td>
                <td><button onClick={() => runTest(p.id)}>Tester</button></td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </AdminShell>
  );
}
