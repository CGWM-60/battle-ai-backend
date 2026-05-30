"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { AdminShell } from "../components/AdminShell";
import { ErrorState, LoadingState } from "../components/LoadState";
import { MetricGrid } from "../components/MetricGrid";
import { formatDate, formatNumber, loadAdminData } from "../components/api";

type GameDashboard = {
  stats: Record<string, number>;
  lastSimulation: string | null;
  providers: { name: string; displayName: string; configured: boolean; primary: boolean; fallback: boolean; model: string }[];
  charts: Record<string, ChartPoint[]>;
};

type ChartPoint = {
  label: string;
  count?: number;
  value?: number;
};

export default function GamePage() {
  const [data, setData] = useState<GameDashboard | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadAdminData<GameDashboard>("game/dashboard").then(setData).catch((err: Error) => setError(err.message));
  }, []);

  async function simulate() {
    const response = await fetch("/admin/api/game/world/simulate", { method: "POST", credentials: "same-origin" });
    if (!response.ok) {
      setError(`Simulation echouee: HTTP ${response.status}`);
      return;
    }
    loadAdminData<GameDashboard>("game/dashboard").then(setData).catch((err: Error) => setError(err.message));
  }

  return (
    <AdminShell title="Monde IA multijoueur" description="Pilotage des mondes, simulations NEXUS, joueurs, contenus dynamiques et activite sociale.">
      {error ? <ErrorState message={error} /> : null}
      {!data && !error ? <LoadingState /> : null}
      {data ? (
        <>
          <section className="panel">
            <h2>Pilotage monde</h2>
            <div className="quick-grid">
              <Link className="quick-link" href="/game/worlds/">
                <strong>Détail monde</strong>
                <span>Voir les mondes, sélectionner un monde et afficher ses continents.</span>
              </Link>
              <Link className="quick-link" href="/game/worlds/">
                <strong>Continents par monde</strong>
                <span>Afficher les 5 continents du monde sélectionné avec tension, climat, économie et joueurs.</span>
              </Link>
              <Link className="quick-link" href="/game/worlds/">
                <strong>Simulation et maintenance</strong>
                <span>Simuler un monde, recalculer les joueurs et archiver les mondes vides.</span>
              </Link>
              <Link className="quick-link" href="/game/research-trees/">
                <strong>Arbres de competence</strong>
                <span>Gerer les arbres de recherche rattaches aux batiments et leurs branches.</span>
              </Link>
              <Link className="quick-link" href="/game/research-nodes/">
                <strong>Noeuds de recherche</strong>
                <span>Modifier les niveaux, durees F2P, ressources et conditions de chaque branche.</span>
              </Link>
              <Link className="quick-link" href="/game/resources/">
                <strong>Ressources systeme</strong>
                <span>Maintenir les ressources referencees par les recherches et les couts dynamiques.</span>
              </Link>
              <Link className="quick-link" href="/game/ai/decisions/">
                <strong>Décisions IA</strong>
                <span>Voir toutes les décisions NEXUS avec input, output, changements appliqués, provider et erreurs.</span>
              </Link>
              <Link className="quick-link" href="/game/daily-tasks/">
                <strong>Tâches Quotidiennes</strong>
                <span>Générer manuellement les tâches quotidiennes (IA méchante, 20-40 par joueur), et voir le tableau historique groupé par jour avec toutes les tâches générées.</span>
              </Link>
            </div>
          </section>
          <MetricGrid
            items={[
              { label: "Mondes", value: formatNumber(data.stats.worlds) },
              { label: "Continents", value: formatNumber(data.stats.continents) },
              { label: "Joueurs serveur", value: formatNumber(data.stats.players) },
              { label: "Actifs 24h", value: formatNumber(data.stats.active24h) },
              { label: "Actifs 7j", value: formatNumber(data.stats.active7d) },
              { label: "Guildes", value: formatNumber(data.stats.guilds) },
              { label: "Events actifs", value: formatNumber(data.stats.activeEvents) },
              { label: "Conflits actifs", value: formatNumber(data.stats.activeConflicts) },
              { label: "Meteo active", value: formatNumber(data.stats.activeWeather) },
              { label: "Messages IA jour", value: formatNumber(data.stats.aiMessagesToday) },
              { label: "Assets actifs", value: formatNumber(data.stats.activeBuildingAssets) },
              { label: "Decisions IA", value: formatNumber(data.stats.aiDecisions) },
            ]}
          />
          <section className="split">
            <article className="panel">
              <h2>Simulation</h2>
              <dl>
                <dt>Derniere simulation</dt>
                <dd>{formatDate(data.lastSimulation)}</dd>
              </dl>
              <div className="editor-actions">
                <button className="primary" type="button" onClick={simulate}>
                  Lancer NEXUS
                </button>
              </div>
            </article>
            <article className="panel">
              <h2>Providers IA</h2>
              <div className="table-wrap">
                <table>
                  <thead>
                    <tr>
                      <th>Provider</th>
                      <th>Etat</th>
                      <th>Modele</th>
                      <th>Role</th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.providers.map((provider) => (
                      <tr key={provider.name}>
                        <td>{provider.displayName}</td>
                        <td className={provider.configured ? "good" : "bad"}>{provider.configured ? "configure" : "incomplet"}</td>
                        <td>{provider.model || "-"}</td>
                        <td>{provider.primary ? "principal" : provider.fallback ? "fallback" : "-"}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </article>
          </section>
          <section className="triple">
            <ChartPanel title="Activite chat" items={data.charts.chatActivity} metric="count" />
            <ChartPanel title="Croissance joueurs" items={data.charts.playerGrowth} metric="count" />
            <ChartPanel title="Rewards reclames" items={data.charts.rewardsClaimed} metric="count" />
          </section>
          <section className="triple">
            <ChartPanel title="Conflits par intensite" items={data.charts.conflictsByIntensity} metric="count" />
            <ChartPanel title="Meteo par severite" items={data.charts.weatherBySeverity} metric="count" />
            <ChartPanel title="Ressources serveur" items={data.charts.resources} metric="value" />
          </section>
        </>
      ) : null}
    </AdminShell>
  );
}

function ChartPanel({ title, items = [], metric }: { title: string; items?: ChartPoint[]; metric: "count" | "value" }) {
  const max = Math.max(1, ...items.map((item) => Number(item[metric] || 0)));
  return (
    <article className="panel">
      <h2>{title}</h2>
      {items.length === 0 ? (
        <p className="muted-panel">Aucune donnee.</p>
      ) : (
        <div className="mini-bars">
          {items.map((item) => {
            const value = Number(item[metric] || 0);
            return (
              <div className="mini-bar-row" key={`${title}-${item.label}`}>
                <span>{item.label}</span>
                <div className="mini-bar-track">
                  <div className="mini-bar-fill" style={{ width: `${Math.max(4, Math.round((value / max) * 100))}%` }} />
                </div>
                <strong>{formatNumber(value)}</strong>
              </div>
            );
          })}
        </div>
      )}
    </article>
  );
}
