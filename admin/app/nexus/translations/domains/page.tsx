"use client";

import { useEffect, useState } from "react";
import { AdminShell } from "../../../components/AdminShell";
import { ErrorState, LoadingState } from "../../../components/LoadState";
import type { TranslationDomain } from "../../../types";

export default function DomainsPage() {
  const [domains, setDomains] = useState<TranslationDomain[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [newDomain, setNewDomain] = useState({ code: "", name: "", description: "" });
  const [busy, setBusy] = useState(false);

  const reload = () => {
    fetch("/admin/api/translations/domains", { credentials: "same-origin" })
      .then((r) => r.json())
      .then((d) => setDomains(d?.domains || d || []))
      .catch((e: Error) => setError(e.message));
  };

  useEffect(() => {
    reload();
  }, []);

  const createDomain = async () => {
    if (!newDomain.code) return;
    setBusy(true);
    try {
      const res = await fetch("/admin/api/translations/domains", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "same-origin",
        body: JSON.stringify(newDomain),
      });
      if (!res.ok) throw new Error(await res.text());
      setNewDomain({ code: "", name: "", description: "" });
      reload();
    } catch (e: any) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  };

  return (
    <AdminShell title="Domaines de traduction" description="CRUD des domaines (ex: nexus_game, common). Appels Go uniquement.">
      {error ? <ErrorState message={error} /> : null}
      {!domains.length && !error ? <LoadingState /> : null}

      {domains.length > 0 && (
        <section className="panel">
          <h2>Domaines existants</h2>
          <table className="data-table">
            <thead>
              <tr>
                <th>Code</th>
                <th>Nom</th>
                <th>Description</th>
              </tr>
            </thead>
            <tbody>
              {domains.map((d) => (
                <tr key={d.ID}>
                  <td><code>{d.Code}</code></td>
                  <td>{d.Name}</td>
                  <td>{d.Description}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </section>
      )}

      <section className="panel">
        <h2>Créer un domaine</h2>
        <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
          <input
            placeholder="Code (ex: nexus_game_city)"
            value={newDomain.code}
            onChange={(e) => setNewDomain({ ...newDomain, code: e.target.value })}
          />
          <input
            placeholder="Nom"
            value={newDomain.name}
            onChange={(e) => setNewDomain({ ...newDomain, name: e.target.value })}
          />
          <input
            placeholder="Description"
            value={newDomain.description}
            onChange={(e) => setNewDomain({ ...newDomain, description: e.target.value })}
            style={{ minWidth: 300 }}
          />
          <button onClick={createDomain} disabled={busy || !newDomain.code}>
            {busy ? "Création..." : "Créer"}
          </button>
        </div>
        <p style={{ fontSize: "0.8em", color: "#666" }}>
          Le domaine sera créé côté Go. Pas de logique ici.
        </p>
      </section>
    </AdminShell>
  );
}
