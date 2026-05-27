"use client";

import { useEffect, useMemo, useState } from "react";
import { AdminShell } from "../../components/AdminShell";
import { ErrorState, LoadingState } from "../../components/LoadState";
import { formatDate, formatNumber, loadAdminData } from "../../components/api";

type World = {
  id: number;
  name: string;
  status: string;
  currentPlayers: number;
  maxPlayers: number;
  globalTensionLevel: number;
  globalWeatherRisk: number;
  globalEconomicState: string;
  currentCycle: number;
  lastSimulationAt: string | null;
};

type Continent = {
  id: number;
  worldId: number;
  name: string;
  index: number;
  status: string;
  currentPlayers: number;
  maxPlayers: number;
  climateState: string;
  politicalState: string;
  economicState: string;
  tensionLevel: number;
  aiBehaviorProfile: string;
};

export default function WorldsPage() {
  const [worlds, setWorlds] = useState<World[]>([]);
  const [continents, setContinents] = useState<Continent[]>([]);
  const [selectedWorldId, setSelectedWorldId] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const selectedWorld = useMemo(() => worlds.find((world) => world.id === selectedWorldId) ?? worlds[0], [worlds, selectedWorldId]);

  useEffect(() => {
    refreshWorlds();
  }, []);

  useEffect(() => {
    if (!selectedWorld) {
      setContinents([]);
      return;
    }
    loadAdminData<{ continents: Continent[] }>(`game/worlds/${selectedWorld.id}/continents`)
      .then((payload) => setContinents(payload.continents ?? []))
      .catch((err: Error) => setError(err.message));
  }, [selectedWorld]);

  async function refreshWorlds() {
    setLoading(true);
    setError(null);
    try {
      const payload = await loadAdminData<{ items: World[] }>("game/worlds?limit=100");
      setWorlds(payload.items ?? []);
      setSelectedWorldId((current) => current ?? payload.items?.[0]?.id ?? null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Erreur inconnue");
    } finally {
      setLoading(false);
    }
  }

  async function postAction(path: string) {
    const response = await fetch(`/admin/api/${path}`, { method: "POST", credentials: "same-origin", headers: { Accept: "application/json" } });
    if (!response.ok) {
      setError(`Action echouee: HTTP ${response.status}`);
      return;
    }
    await refreshWorlds();
  }

  return (
    <AdminShell title="Mondes" description="Capacite, continents par monde, simulations NEXUS et maintenance des compteurs serveur.">
      {error ? <ErrorState message={error} /> : null}
      {loading ? <LoadingState /> : null}
      {!loading ? (
        <>
          <section className="panel game-toolbar">
            <button className="primary" type="button" onClick={() => postAction("game/worlds")}>
              Creer un monde
            </button>
            <button className="secondary" type="button" onClick={() => postAction("game/worlds/reconcile-counts")}>
              Recalculer joueurs
            </button>
            <button className="danger" type="button" onClick={() => postAction("game/worlds/archive-empty")}>
              Archiver mondes vides
            </button>
            <button className="secondary" type="button" onClick={refreshWorlds}>
              Recharger
            </button>
          </section>

          <section className="panel">
            <div className="table-wrap">
              <table>
                <thead>
                  <tr>
                    <th>ID</th>
                    <th>Nom</th>
                    <th>Statut</th>
                    <th>Joueurs</th>
                    <th>Tension</th>
                    <th>Meteo</th>
                    <th>Economie</th>
                    <th>Cycle</th>
                    <th>Derniere simulation</th>
                    <th>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {worlds.map((world) => (
                    <tr className={selectedWorld?.id === world.id ? "selected-row" : ""} key={world.id}>
                      <td>{world.id}</td>
                      <td>{world.name}</td>
                      <td>{world.status}</td>
                      <td>
                        {formatNumber(world.currentPlayers)} / {formatNumber(world.maxPlayers)}
                      </td>
                      <td>{formatNumber(world.globalTensionLevel)}</td>
                      <td>{formatNumber(world.globalWeatherRisk)}</td>
                      <td>{world.globalEconomicState || "-"}</td>
                      <td>{formatNumber(world.currentCycle)}</td>
                      <td>{formatDate(world.lastSimulationAt)}</td>
                      <td className="row-actions">
                        <button className="secondary" type="button" onClick={() => setSelectedWorldId(world.id)}>
                          Détail monde
                        </button>
                        <button className="secondary" type="button" onClick={() => postAction(`game/worlds/${world.id}/simulate`)}>
                          Simuler monde
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            {!worlds.length ? <p className="hint">Aucun monde.</p> : null}
          </section>

          {selectedWorld ? (
            <section className="panel">
              <h2>
                Continents par monde - monde #{selectedWorld.id} - {selectedWorld.name}
              </h2>
              <p className="muted-panel">Détail du monde sélectionné: capacité, profils IA continentaux, climat, politique, économie, tension et répartition des joueurs.</p>
              <div className="table-wrap">
                <table>
                  <thead>
                    <tr>
                      <th>Index</th>
                      <th>Nom</th>
                      <th>Statut</th>
                      <th>Joueurs</th>
                      <th>Profil IA</th>
                      <th>Climat</th>
                      <th>Politique</th>
                      <th>Economie</th>
                      <th>Tension</th>
                    </tr>
                  </thead>
                  <tbody>
                    {continents.map((continent) => (
                      <tr key={continent.id}>
                        <td>{continent.index}</td>
                        <td>{continent.name}</td>
                        <td>{continent.status}</td>
                        <td>
                          {formatNumber(continent.currentPlayers)} / {formatNumber(continent.maxPlayers)}
                        </td>
                        <td>{continent.aiBehaviorProfile || "-"}</td>
                        <td>{continent.climateState || "-"}</td>
                        <td>{continent.politicalState || "-"}</td>
                        <td>{continent.economicState || "-"}</td>
                        <td>{formatNumber(continent.tensionLevel)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              {!continents.length ? <p className="hint">Aucun continent pour ce monde.</p> : null}
            </section>
          ) : null}
        </>
      ) : null}
    </AdminShell>
  );
}
