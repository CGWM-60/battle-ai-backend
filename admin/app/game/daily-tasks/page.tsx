"use client";

import { useEffect, useState } from "react";
import { AdminShell } from "../../../components/AdminShell";
import { ErrorState, LoadingState } from "../../../components/LoadState";
import { formatDate, loadAdminData } from "../../../components/api";

type DailyTask = {
  id: number;
  playerId: number;
  worldId: number;
  title: string;
  description: string;
  taskType: string;
  targetValue: number;
  currentValue: number;
  rewardType: string;
  rewardAmount: number;
  durationMinutes: number;
  status: string;
  expiresAt?: string | null;
  createdAt: string;
};

type GenerateResponse = {
  success: boolean;
  worldId: number;
  playersProcessed: number;
  generatedFor: number;
  message: string;
};

export default function DailyTasksAdminPage() {
  const [tasks, setTasks] = useState<DailyTask[]>([]);
  const [worldId, setWorldId] = useState<string>("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [lastGenerate, setLastGenerate] = useState<GenerateResponse | null>(null);

  async function reload() {
    setLoading(true);
    setError(null);
    try {
      const qs = worldId ? `?worldId=${worldId}` : "";
      const payload = await loadAdminData<{ tasks: DailyTask[] }>(`game/daily-tasks${qs}`);
      setTasks(payload.tasks ?? []);
    } catch (e: any) {
      setError(e?.message || "Erreur chargement tâches quotidiennes");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    reload();
  }, [worldId]);

  async function generateNow() {
    setBusy(true);
    setError(null);
    setLastGenerate(null);

    try {
      const body: any = {};
      if (worldId) body.worldId = parseInt(worldId, 10);

      const res = await fetch(`/admin/api/game/daily-tasks/generate`, {
        method: "POST",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json", Accept: "application/json" },
        body: JSON.stringify(body),
      });

      if (!res.ok) {
        const txt = await res.text();
        throw new Error(`HTTP ${res.status} - ${txt}`);
      }

      const data: GenerateResponse = await res.json();
      setLastGenerate(data);

      // Refresh list after generation
      await reload();
    } catch (e: any) {
      setError(e?.message || "Génération échouée");
    } finally {
      setBusy(false);
    }
  }

  // Group tasks by day (YYYY-MM-DD)
  const grouped = tasks.reduce<Record<string, DailyTask[]>>((acc, t) => {
    const day = (t.createdAt || "").slice(0, 10);
    if (!acc[day]) acc[day] = [];
    acc[day].push(t);
    return acc;
  }, {});

  const days = Object.keys(grouped).sort((a, b) => b.localeCompare(a)); // newest first

  return (
    <AdminShell
      title="Tâches Quotidiennes"
      description="Génération manuelle + historique des tâches quotidiennes générées par l'IA méchante (20-40 par joueur/jour)"
    >
      {error && <div className="alert error">{error}</div>}

      <div style={{ display: "flex", gap: 12, alignItems: "center", marginBottom: 16, flexWrap: "wrap" }}>
        <div>
          <label style={{ fontSize: 12, color: "#7BE04B" }}>World ID (optionnel)</label>
          <input
            type="number"
            value={worldId}
            onChange={(e) => setWorldId(e.target.value)}
            placeholder="1"
            style={{ width: 120, padding: "6px 10px", background: "#0A1628", border: "1px solid #2A3F5F", color: "white", borderRadius: 6 }}
          />
        </div>

        <button
          onClick={generateNow}
          disabled={busy}
          style={{
            background: busy ? "#333" : "#FF8A00",
            color: "black",
            fontWeight: 700,
            padding: "10px 18px",
            borderRadius: 8,
            border: "none",
            cursor: busy ? "not-allowed" : "pointer",
            fontSize: 14,
          }}
        >
          {busy ? "Génération en cours..." : "⚡ Générer les tâches quotidiennes MAINTENANT"}
        </button>

        <button onClick={reload} style={{ padding: "8px 14px", background: "#0A1628", border: "1px solid #7BE04B", color: "#7BE04B", borderRadius: 6 }}>
          Rafraîchir
        </button>
      </div>

      {lastGenerate && (
        <div className="alert ok" style={{ marginBottom: 16 }}>
          {lastGenerate.message} — Monde {lastGenerate.worldId} • {lastGenerate.generatedFor} joueurs impactés
        </div>
      )}

      {loading ? (
        <LoadingState />
      ) : tasks.length === 0 ? (
        <div style={{ padding: 24, background: "#0A1628", borderRadius: 8, color: "#7BE04B" }}>
          Aucune tâche quotidienne trouvée pour ce filtre. Clique sur le bouton de génération.
        </div>
      ) : (
        <div>
          <h3 style={{ color: "#7BE04B", marginBottom: 8 }}>Historique par jour ({tasks.length} tâches)</h3>

          {days.map((day) => (
            <div key={day} style={{ marginBottom: 20, background: "#0A1628", borderRadius: 8, padding: 12, border: "1px solid #1F2A3F" }}>
              <div style={{ fontWeight: 700, color: "#FF8A00", marginBottom: 8, fontSize: 13 }}>
                {day} — {grouped[day].length} tâches générées
              </div>

              <table style={{ width: "100%", borderCollapse: "collapse", fontSize: 12 }}>
                <thead>
                  <tr style={{ background: "#111B2E" }}>
                    <th style={{ textAlign: "left", padding: "6px 8px" }}>Titre</th>
                    <th style={{ textAlign: "left", padding: "6px 8px" }}>Type</th>
                    <th style={{ textAlign: "right", padding: "6px 8px" }}>Objectif</th>
                    <th style={{ textAlign: "left", padding: "6px 8px" }}>Récompense</th>
                    <th style={{ textAlign: "left", padding: "6px 8px" }}>Statut</th>
                    <th style={{ textAlign: "left", padding: "6px 8px" }}>Expire</th>
                  </tr>
                </thead>
                <tbody>
                  {grouped[day].map((t) => (
                    <tr key={t.id} style={{ borderTop: "1px solid #1F2A3F" }}>
                      <td style={{ padding: "6px 8px", color: "#E8F0FE" }}>{t.title}</td>
                      <td style={{ padding: "6px 8px", color: "#7BE04B" }}>{t.taskType}</td>
                      <td style={{ padding: "6px 8px", textAlign: "right" }}>
                        {t.currentValue} / {t.targetValue}
                      </td>
                      <td style={{ padding: "6px 8px", color: "#FF8A00" }}>
                        +{t.rewardAmount} {t.rewardType}
                      </td>
                      <td style={{ padding: "6px 8px" }}>
                        <span style={{
                          padding: "1px 6px",
                          borderRadius: 4,
                          background: t.status === "completed" ? "#1A3A1A" : t.status === "available" ? "#2A3F5F" : "#3A2A1A",
                          color: t.status === "completed" ? "#7BE04B" : "#E8F0FE",
                          fontSize: 11,
                        }}>
                          {t.status}
                        </span>
                      </td>
                      <td style={{ padding: "6px 8px", color: "#888", fontSize: 11 }}>
                        {t.expiresAt ? formatDate(t.expiresAt) : "—"}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ))}
        </div>
      )}

      <div style={{ marginTop: 24, fontSize: 12, color: "#666" }}>
        Les tâches sont générées par joueur (via l'IA méchante ou manuellement ici). Le cron automatique tourne à 4h du matin.
      </div>
    </AdminShell>
  );
}
