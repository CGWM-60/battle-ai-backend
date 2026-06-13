"use client";

import Link from "next/link";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { AdminShell } from "../../components/AdminShell";
import { formatDate, formatNumber, loadAdminData } from "../../components/api";
import type { DashboardData } from "../../types";

const API_BASE = (process.env.NEXT_PUBLIC_NEXUS_API_BASE || "").replace(/\/$/, "");
const REFRESH_MS = 12000;

type Probe<T> = {
  label: string;
  endpoint: string;
  ok: boolean;
  data?: T;
  error?: string;
};

type CatalogCounts = {
  buildings: number;
  units: number;
  research: number;
  total: number;
};

type CockpitState = {
  dashboard: DashboardData | null;
  worlds: any[];
  factions: any[];
  companions: any[];
  publicPrompts: any[];
  serverPrompts: any[];
  catalogCounts: CatalogCounts;
  aiDashboard: any | null;
  aiLogs: any[];
  aiCosts: any | null;
  seasonalEvents: any[];
  broadcasts: any[];
  attacks: any[];
  probes: Probe<any>[];
  refreshedAt: string;
};

const emptyState: CockpitState = {
  dashboard: null,
  worlds: [],
  factions: [],
  companions: [],
  publicPrompts: [],
  serverPrompts: [],
  catalogCounts: { buildings: 0, units: 0, research: 0, total: 0 },
  aiDashboard: null,
  aiLogs: [],
  aiCosts: null,
  seasonalEvents: [],
  broadcasts: [],
  attacks: [],
  probes: [],
  refreshedAt: "",
};

async function readJson<T>(label: string, endpoint: string): Promise<Probe<T>> {
  try {
    const res = await fetch(`${API_BASE}${endpoint}`, { credentials: "same-origin" });
    if (!res.ok) {
      return { label, endpoint, ok: false, error: `${res.status} ${res.statusText}` };
    }
    return { label, endpoint, ok: true, data: (await res.json()) as T };
  } catch (error: any) {
    return { label, endpoint, ok: false, error: error?.message || "Erreur reseau" };
  }
}

function valueFrom(data: any, keys: string[], fallback: any = "-") {
  for (const key of keys) {
    if (data && data[key] !== undefined && data[key] !== null && data[key] !== "") {
      return data[key];
    }
  }
  return fallback;
}

function listFrom(data: any, keys: string[]) {
  for (const key of keys) {
    if (Array.isArray(data?.[key])) return data[key];
  }
  return [];
}

function normalizeCatalog(data: any): CatalogCounts {
  return {
    buildings: data?.counts?.buildings || data?.buildings?.length || 0,
    units: data?.counts?.units || data?.units?.length || 0,
    research: data?.counts?.research || data?.research?.length || 0,
    total: data?.counts?.total || 0,
  };
}

function statusClass(status: any) {
  return `status ${String(status || "unknown").toLowerCase().replace(/[^a-z0-9_-]/g, "_")}`;
}

function detailArray(value: any) {
  return Array.isArray(value) ? value : [];
}

function contentLabel(item: any, ownerKey: string) {
  return (
    item?.definition?.contentId ||
    item?.definition?.content_id ||
    item?.[ownerKey]?.contentId ||
    item?.[ownerKey]?.content_id ||
    item?.contentId ||
    item?.content_id ||
    "-"
  );
}

function resourceCode(resource: any) {
  return String(resource?.resourceCode || resource?.resource_code || "");
}

function resourceAmount(resource: any) {
  return Number(resource?.amount || 0);
}

export default function NexusWorldControlPage() {
  const [state, setState] = useState<CockpitState>(emptyState);
  const [loading, setLoading] = useState(true);
  const [manualError, setManualError] = useState("");
  const [actionMessage, setActionMessage] = useState("");
  const [deletingProfileId, setDeletingProfileId] = useState<number | null>(null);
  const [selectedPlayer, setSelectedPlayer] = useState<any | null>(null);
  const [playerDetail, setPlayerDetail] = useState<any | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailError, setDetailError] = useState("");
  const [resourceEdits, setResourceEdits] = useState<Record<string, string>>({});
  const [savingResources, setSavingResources] = useState<Record<string, boolean>>({});
  const resourceSaveTimers = useRef<Record<string, number>>({});

  const load = useCallback(async () => {
    setManualError("");
    setLoading(true);

    const dashboardProbe: Probe<DashboardData> = await loadAdminData<DashboardData>("dashboard")
      .then((data) => ({ label: "Admin dashboard", endpoint: "/admin/api/dashboard", ok: true, data }))
      .catch((error: any) => ({
        label: "Admin dashboard",
        endpoint: "/admin/api/dashboard",
        ok: false,
        error: error?.message || "Erreur dashboard",
      }));

    const probes = await Promise.all([
      readJson<any>("Mondes", "/api/nexus-game/worlds"),
      readJson<any>("Factions", "/api/nexus-game/factions"),
      readJson<any>("Compagnons IA", "/api/nexus-game/ia-companions"),
      readJson<any>("Prompts publics", "/api/nexus-game/prompts"),
      readJson<any>("Catalogue construction", "/api/nexus-game/admin/content/catalog"),
      readJson<any>("Dashboard IA serveur", "/api/nexus-game/admin/ai-server/dashboard"),
      readJson<any>("Prompts IA serveur", "/api/nexus-game/admin/ai-server/prompts"),
      readJson<any>("Logs IA", "/api/nexus-game/admin/ai-server/call-logs"),
      readJson<any>("Couts IA", "/api/nexus-game/admin/ai-server/costs"),
      readJson<any>("Evenements saisonniers", "/api/nexus-game/admin/seasonal-events"),
      readJson<any>("Broadcasts", "/api/nexus-game/admin/ai-server/broadcasts"),
      readJson<any>("Attaques IA", "/api/nexus-game/admin/ai-server/attacks"),
    ]);

    const byLabel = Object.fromEntries(probes.map((probe) => [probe.label, probe]));
    const nextState: CockpitState = {
      dashboard: dashboardProbe.ok ? dashboardProbe.data ?? null : null,
      worlds: listFrom(byLabel["Mondes"]?.data, ["worlds"]),
      factions: listFrom(byLabel["Factions"]?.data, ["factions"]),
      companions: listFrom(byLabel["Compagnons IA"]?.data, ["ia_companions", "companions"]),
      publicPrompts: listFrom(byLabel["Prompts publics"]?.data, ["prompts"]),
      serverPrompts: listFrom(byLabel["Prompts IA serveur"]?.data, ["prompts"]),
      catalogCounts: normalizeCatalog(byLabel["Catalogue construction"]?.data),
      aiDashboard: byLabel["Dashboard IA serveur"]?.data || null,
      aiLogs: listFrom(byLabel["Logs IA"]?.data, ["logs"]),
      aiCosts: byLabel["Couts IA"]?.data || null,
      seasonalEvents: listFrom(byLabel["Evenements saisonniers"]?.data, ["events", "items"]),
      broadcasts: listFrom(byLabel["Broadcasts"]?.data, ["broadcasts"]),
      attacks: listFrom(byLabel["Attaques IA"]?.data, ["attacks"]),
      probes: [dashboardProbe, ...probes],
      refreshedAt: new Date().toISOString(),
    };

    setState(nextState);
    setLoading(false);
    const errors = nextState.probes.filter((probe) => !probe.ok);
    if (errors.length > 0) {
      setManualError(`${errors.length} flux indisponible(s). Le cockpit affiche les donnees disponibles.`);
    }
  }, []);

  useEffect(() => {
    load();
    const timer = window.setInterval(load, REFRESH_MS);
    return () => window.clearInterval(timer);
  }, [load]);

  useEffect(() => {
    return () => {
      Object.values(resourceSaveTimers.current).forEach((timer) => window.clearTimeout(timer));
    };
  }, []);

  const worldStats = useMemo(() => {
    const continents = state.worlds.flatMap((world) =>
      (world.continents || []).map((continent: any) => ({ ...continent, worldName: world.name, worldId: world.id })),
    );
    const players = continents.flatMap((continent: any) =>
      (continent.players_list || []).map((player: any) => ({
        ...player,
        worldId: continent.worldId,
        worldName: continent.worldName,
        continentId: continent.id,
        continentName: continent.name,
      })),
    );
    const capacity = continents.reduce(
      (sum: number, continent: any) => sum + Number(continent.max_players || continent.maxPlayers || continent.capacity || 0),
      0,
    );
    const assigned = players.length;
    const activeWorlds = state.worlds.filter((world) => String(world.status || "").toLowerCase() !== "archived").length;
    return { continents, players, capacity, assigned, activeWorlds };
  }, [state.worlds]);

  const recentBattleQuests = state.dashboard?.Recent?.BattleQuests || [];
  const recentRoleplayQuests = state.dashboard?.Recent?.RolePlayQuests || [];
  const recentLives = state.dashboard?.Recent?.LiveSessions || [];
  const aiCalls24h = valueFrom(state.aiDashboard, ["callsLast24h"], state.aiLogs.length);
  const eventsToReview = valueFrom(state.aiDashboard, ["eventsToReview"], state.seasonalEvents.filter((event) => event.status === "draft").length);
  const liveStreaming = state.dashboard?.Stats?.LiveStreaming || 0;
  const endpointErrors = state.probes.filter((probe) => !probe.ok);

  const deleteWorldPlayer = async (player: any) => {
    const profileId = Number(player.profile_id || player.profileId || player.id);
    const worldId = Number(player.worldId || player.world_id);
    const label = player.pseudo || player.city_name || `profil ${profileId}`;
    if (!profileId || !worldId) {
      setManualError("Impossible de supprimer: profileId ou worldId manquant.");
      return;
    }
    const confirmed = window.confirm(
      `Supprimer integralement ${label} du monde ${player.worldName || worldId} ?\n\nCette action efface le profil Nexus, ressources, batiments, unites, recherches, plans, agents IA et donnees IA liees. Elle ne supprime pas le compte utilisateur global.`,
    );
    if (!confirmed) return;
    setDeletingProfileId(profileId);
    setManualError("");
    setActionMessage("");
    try {
      const res = await fetch(`${API_BASE}/api/nexus-game/worlds/${worldId}/players/${profileId}`, {
        method: "DELETE",
        credentials: "same-origin",
      });
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      const deleted = data?.result?.deleted || {};
      const total = Object.values(deleted).reduce((sum: number, value: any) => sum + Number(value || 0), 0);
      setActionMessage(`${label} supprime du monde. ${total} ligne(s) Nexus effacee(s).`);
      await load();
    } catch (error: any) {
      setManualError(error?.message || "Suppression impossible.");
    } finally {
      setDeletingProfileId(null);
    }
  };

  const openPlayerDetail = async (player: any) => {
    const profileId = Number(player.profile_id || player.profileId || player.id);
    const worldId = Number(player.worldId || player.world_id);
    if (!profileId || !worldId) {
      setManualError("Impossible d'ouvrir le joueur: profileId ou worldId manquant.");
      return;
    }
    setSelectedPlayer(player);
    setPlayerDetail(null);
    setDetailError("");
    setResourceEdits({});
    setSavingResources({});
    setDetailLoading(true);
    try {
      const res = await fetch(`${API_BASE}/api/nexus-game/worlds/${worldId}/players/${profileId}/detail`, {
        credentials: "same-origin",
      });
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      setPlayerDetail(data?.player || data);
    } catch (error: any) {
      setDetailError(error?.message || "Detail joueur indisponible.");
    } finally {
      setDetailLoading(false);
    }
  };

  const saveResourceAmount = async (resource: any, rawTarget: string) => {
    const code = resourceCode(resource);
    if (rawTarget.trim() === "") return;
    const target = Math.max(0, Math.trunc(Number(rawTarget)));
    const current = Math.trunc(resourceAmount(resource));
    const delta = target - current;
    const profileId = Number(playerDetail?.profile?.id || selectedPlayer?.profile_id || selectedPlayer?.profileId || selectedPlayer?.id || 0);
    const worldId = Number(
      selectedPlayer?.worldId ||
        selectedPlayer?.world_id ||
        playerDetail?.identity?.world?.id ||
        playerDetail?.profile?.worldId ||
        playerDetail?.profile?.world_id ||
        0,
    );

    if (!code || !Number.isFinite(target) || !profileId) return;
    if (delta === 0) {
      setResourceEdits((prev) => {
        const next = { ...prev };
        delete next[code];
        return next;
      });
      return;
    }

    setSavingResources((prev) => ({ ...prev, [code]: true }));
    setManualError("");
    try {
      const res = await fetch(`${API_BASE}/admin/api/nexus-system/resources/grant`, {
        method: "POST",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          profileGamerId: profileId,
          resourceCode: code,
          amount: delta,
          reason: "player_telemetry_adjust",
        }),
      });
      if (!res.ok) throw new Error(await res.text());

      if (worldId > 0) {
        const detailRes = await fetch(`${API_BASE}/api/nexus-game/worlds/${worldId}/players/${profileId}/detail`, {
          credentials: "same-origin",
        });
        if (detailRes.ok) {
          const data = await detailRes.json();
          setPlayerDetail(data?.player || data);
        }
      }
      setResourceEdits((prev) => {
        const next = { ...prev };
        delete next[code];
        return next;
      });
      setActionMessage(`${code}: stock ajuste de ${formatNumber(current)} a ${formatNumber(target)}.`);
    } catch (error: any) {
      setManualError(error?.message || `Impossible de modifier ${code}.`);
    } finally {
      setSavingResources((prev) => {
        const next = { ...prev };
        delete next[code];
        return next;
      });
    }
  };

  const scheduleResourceAmountSave = (resource: any, rawTarget: string) => {
    const code = resourceCode(resource);
    if (!code) return;
    setResourceEdits((prev) => ({ ...prev, [code]: rawTarget }));
    if (resourceSaveTimers.current[code]) {
      window.clearTimeout(resourceSaveTimers.current[code]);
    }
    if (rawTarget.trim() === "" || !Number.isFinite(Number(rawTarget))) {
      return;
    }
    resourceSaveTimers.current[code] = window.setTimeout(() => {
      delete resourceSaveTimers.current[code];
      void saveResourceAmount(resource, rawTarget);
    }, 650);
  };

  const commandLinks = [
    { href: "/nexus/mmo/", title: "Console MMO", detail: "Avatars, factions, compagnons, mondes et generation manuelle." },
    { href: "/nexus/mmo/buildings", title: "Batiments", detail: "Contraintes, couts, durees, effets et images de construction." },
    { href: "/nexus/mmo/units", title: "Unites", detail: "Stats, prerequisites, recrutement, assets et traductions." },
    { href: "/nexus/mmo/research", title: "Recherches", detail: "Arbre techno, durees, dependances, unlocks et descriptions." },
    { href: "/nexus/ai-server/", title: "IA serveur", detail: "Pression, bastions, generation et compteurs du noyau IA." },
    { href: "/nexus/ai-server/jobs", title: "Jobs IA", detail: "Scheduler complet: minutes, heures, daily, weekly, saison et executions forcees." },
    { href: "/nexus/ai-server/prompts", title: "Prompts IA", detail: "Prompts, tests, seed et versions de generation." },
    { href: "/nexus/ai-server/logs", title: "Logs & couts", detail: "Tokens, latence, erreurs et cout estime par appel." },
    { href: "/nexus/ai-server/attacks", title: "Attaques", detail: "Planification, annulation et resolution d'attaques IA." },
    { href: "/nexus/ai-server/broadcasts", title: "Broadcasts", detail: "Messages serveur generes, publies ou en attente." },
    { href: "/nexus/ai-server/memory", title: "Memoire IA", detail: "Memoire globale et memoire joueur nettoyable." },
    { href: "/nexus/seasonal-events", title: "Evenements", detail: "Propositions, validation, scheduling et lancement." },
    { href: "/nexus/translations/", title: "Traductions", detail: "Domaines, statuts, imports, manquants et logs." },
    { href: "/quests/", title: "Quetes", detail: "Quetes bataille generees, statut et recompenses." },
    { href: "/roleplay-quests/", title: "Quetes RP", detail: "Quetes roleplay, arcs narratifs et generation recue." },
    { href: "/live/", title: "Live", detail: "Sessions, streaming, replay et audience." },
    { href: "/tribunal-ai/", title: "Tribunal IA", detail: "Dossiers, decisions et controle de moderation." },
  ];

  return (
    <AdminShell
      title="Tour de controle Nexus World"
      description="Vue temps reel du monde Nexus: connexions, recherches, constructions, unites, IA serveur, evenements, quetes et flux admin."
    >
      {manualError ? <div className="alert error">{manualError}</div> : null}
      {actionMessage ? <div className="alert success">{actionMessage}</div> : null}

      <section className="control-hero">
        <div>
          <p className="eyebrow">Nexus World / Command center</p>
          <h2>Etat operationnel global</h2>
          <p>
            Rafraichissement automatique toutes les {REFRESH_MS / 1000}s. Dernier signal:{" "}
            {state.refreshedAt ? formatDate(state.refreshedAt) : "chargement"}.
          </p>
        </div>
        <div className="control-actions">
          <span className={endpointErrors.length === 0 ? "status running" : "status skipped"}>
            {endpointErrors.length === 0 ? "Tous les flux repondent" : `${endpointErrors.length} flux a verifier`}
          </span>
          <button className="primary" type="button" onClick={load} disabled={loading}>
            {loading ? "Synchronisation" : "Rafraichir"}
          </button>
        </div>
      </section>

      <section className="control-metric-grid">
        <div className="metric-tile">
          <span>Mondes actifs</span>
          <strong>{formatNumber(worldStats.activeWorlds)}</strong>
        </div>
        <div className="metric-tile">
          <span>Continents</span>
          <strong>{formatNumber(worldStats.continents.length)}</strong>
        </div>
        <div className="metric-tile">
          <span>Joueurs affectes</span>
          <strong>
            {formatNumber(worldStats.assigned)}
            {worldStats.capacity ? <small> / {formatNumber(worldStats.capacity)}</small> : null}
          </strong>
        </div>
        <div className="metric-tile">
          <span>Live streaming</span>
          <strong>{formatNumber(liveStreaming)}</strong>
        </div>
        <div className="metric-tile">
          <span>Batiments</span>
          <strong>{formatNumber(state.catalogCounts.buildings)}</strong>
        </div>
        <div className="metric-tile">
          <span>Unites</span>
          <strong>{formatNumber(state.catalogCounts.units)}</strong>
        </div>
        <div className="metric-tile">
          <span>Recherches</span>
          <strong>{formatNumber(state.catalogCounts.research)}</strong>
        </div>
        <div className="metric-tile">
          <span>Appels IA 24h</span>
          <strong>{formatNumber(Number(aiCalls24h) || 0)}</strong>
        </div>
      </section>

      <section className="split">
        <div className="panel">
          <h2>Connexions et occupation</h2>
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Monde</th>
                  <th>Continent</th>
                  <th>Joueurs</th>
                  <th>Capacite</th>
                  <th>Statut</th>
                </tr>
              </thead>
              <tbody>
                {worldStats.continents.slice(0, 10).map((continent: any) => (
                  <tr key={`${continent.worldId}-${continent.id || continent.name}`}>
                    <td>{continent.worldName || continent.worldId}</td>
                    <td>{continent.name || continent.id}</td>
                    <td>{(continent.players_list || []).length}</td>
                    <td>{continent.max_players || continent.maxPlayers || continent.capacity || "-"}</td>
                    <td><span className={statusClass(continent.status)}>{continent.status || "online"}</span></td>
                  </tr>
                ))}
                {worldStats.continents.length === 0 ? <tr><td colSpan={5}>Aucun continent recu.</td></tr> : null}
              </tbody>
            </table>
          </div>
        </div>

        <div className="panel">
          <h2>Noyau IA et evenements</h2>
          <div className="signal-list">
            <div><span>Bastions IA</span><strong>{valueFrom(state.aiDashboard, ["citiesCount"], "-")}</strong></div>
            <div><span>Attaques</span><strong>{valueFrom(state.aiDashboard, ["attacksCount"], state.attacks.length)}</strong></div>
            <div><span>Events a revoir</span><strong>{eventsToReview}</strong></div>
            <div><span>Broadcasts</span><strong>{valueFrom(state.aiDashboard, ["broadcastsCount"], state.broadcasts.length)}</strong></div>
            <div><span>Prompts serveur</span><strong>{state.serverPrompts.length}</strong></div>
            <div><span>Prompts publics</span><strong>{state.publicPrompts.length}</strong></div>
          </div>
        </div>
      </section>

      <section className="panel">
        <h2>Joueurs Nexus</h2>
        <p className="hint">
          Suppression complete du joueur dans Nexus Games uniquement: profil, ressources, batiments, unites, recherches, plans, agents et donnees IA liees.
        </p>
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Monde</th>
                <th>Continent</th>
                <th>Profil</th>
                <th>User ID</th>
                <th>Niveau</th>
                <th>Puissance</th>
                <th>Faction</th>
                <th>Action</th>
              </tr>
            </thead>
            <tbody>
              {worldStats.players.slice(0, 25).map((player: any) => {
                const profileId = Number(player.profile_id || player.profileId || player.id);
                return (
                  <tr
                    key={`${player.worldId}-${profileId}`}
                    onClick={() => openPlayerDetail(player)}
                    style={{ cursor: "pointer" }}
                    title="Ouvrir le detail complet du joueur"
                  >
                    <td>{player.worldName || player.worldId || "-"}</td>
                    <td>{player.continentName || player.continentId || "-"}</td>
                    <td>
                      <strong>{player.pseudo || "-"}</strong>
                      <br />
                      <span className="hint">{player.city_name || player.cityName || `profil #${profileId}`}</span>
                    </td>
                    <td>{player.user_id || player.userId || "-"}</td>
                    <td>{player.level || 1}</td>
                    <td>{formatNumber(Number(player.power || 0))}</td>
                    <td>{player.faction_name || player.factionName || "-"}</td>
                    <td onClick={(event) => event.stopPropagation()}>
                      <button
                        type="button"
                        disabled={deletingProfileId === profileId}
                        onClick={() => deleteWorldPlayer(player)}
                        style={{ background: "#991b1b", color: "white", border: 0, borderRadius: 6, padding: "7px 10px", fontWeight: 800 }}
                      >
                        {deletingProfileId === profileId ? "Suppression..." : "Supprimer"}
                      </button>
                    </td>
                  </tr>
                );
              })}
              {worldStats.players.length === 0 ? <tr><td colSpan={8}>Aucun joueur affecte a un monde.</td></tr> : null}
            </tbody>
          </table>
        </div>
      </section>

      <section className="triple">
        <div className="panel">
          <h2>Quetes generees</h2>
          <div className="event-feed">
            {recentBattleQuests.slice(0, 5).map((quest) => (
              <Link href="/quests/" key={quest.Id}>
                <strong>{quest.Title}</strong>
                <span>{quest.Status} - {quest.Theme || quest.Level}</span>
              </Link>
            ))}
            {recentBattleQuests.length === 0 ? <p className="hint">Aucune quete bataille recente.</p> : null}
          </div>
        </div>

        <div className="panel">
          <h2>Quetes RP recues</h2>
          <div className="event-feed">
            {recentRoleplayQuests.slice(0, 5).map((quest) => (
              <Link href="/roleplay-quests/" key={quest.Id}>
                <strong>{quest.Title}</strong>
                <span>{quest.Status} - {quest.Theme || quest.Level}</span>
              </Link>
            ))}
            {recentRoleplayQuests.length === 0 ? <p className="hint">Aucune quete RP recente.</p> : null}
          </div>
        </div>

        <div className="panel">
          <h2>Flux live</h2>
          <div className="event-feed">
            {recentLives.slice(0, 5).map((live) => (
              <Link href="/live/" key={live.Id}>
                <strong>{live.ChannelKey}</strong>
                <span>{live.Status} - {live.ViewerCount} spectateurs</span>
              </Link>
            ))}
            {recentLives.length === 0 ? <p className="hint">Aucune session live recente.</p> : null}
          </div>
        </div>
      </section>

      <section className="split">
        <div className="panel">
          <h2>Evenements et broadcasts</h2>
          <div className="table-wrap">
            <table>
              <thead><tr><th>Type</th><th>Titre</th><th>Statut</th><th>Date</th></tr></thead>
              <tbody>
                {state.seasonalEvents.slice(0, 6).map((event) => (
                  <tr key={`event-${event.id}`}>
                    <td>Event</td>
                    <td>{event.title || event.name || event.id}</td>
                    <td><span className={statusClass(event.status)}>{event.status || "-"}</span></td>
                    <td>{event.startsAt || event.startDate || event.createdAt || "-"}</td>
                  </tr>
                ))}
                {state.broadcasts.slice(0, 6).map((broadcast) => (
                  <tr key={`broadcast-${broadcast.id}`}>
                    <td>Broadcast</td>
                    <td>{broadcast.title || broadcast.headline || broadcast.id}</td>
                    <td><span className={statusClass(broadcast.status)}>{broadcast.status || "-"}</span></td>
                    <td>{broadcast.date || broadcast.createdAt || "-"}</td>
                  </tr>
                ))}
                {state.seasonalEvents.length + state.broadcasts.length === 0 ? <tr><td colSpan={4}>Aucun event ou broadcast recu.</td></tr> : null}
              </tbody>
            </table>
          </div>
        </div>

        <div className="panel">
          <h2>Sante des flux</h2>
          <div className="endpoint-grid">
            {state.probes.map((probe) => (
              <div className="endpoint-row" key={probe.endpoint}>
                <span className={probe.ok ? "status running" : "status failed"}>{probe.ok ? "OK" : "Erreur"}</span>
                <div>
                  <strong>{probe.label}</strong>
                  <small>{probe.endpoint}{probe.error ? ` - ${probe.error}` : ""}</small>
                </div>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="panel">
        <h2>Actions manuelles (sans cron)</h2>
        <p style={{ fontSize: 13, color: "#64748b", marginBottom: 12 }}>
          Déclenchez les actions serveur manuellement. A utiliser jusqu'à la mise en place d'un cron/scheduler.
        </p>
        <div style={{ display: "flex", flexWrap: "wrap", gap: 8 }}>
          {state.worlds.map((world: any) => (
            <div key={world.id} style={{ display: "flex", gap: 6, alignItems: "center", border: "1px solid #e2e8f0", borderRadius: 8, padding: "8px 12px" }}>
              <strong style={{ fontSize: 13 }}>{world.name || `Monde #${world.id}`}</strong>
              <button
                type="button"
                style={{ padding: "5px 10px", fontSize: 12, background: "#0f172a", color: "white", border: "none", borderRadius: 6, cursor: "pointer" }}
                onClick={async () => {
                  setManualError("");
                  const r = await fetch(`${API_BASE}/api/nexus-game/worlds/${world.id}/trigger-tick`, { method: "POST", credentials: "same-origin" });
                  const txt = await r.text();
                  setManualError(`[Tick ${world.name || world.id}] ${r.ok ? "OK" : "Erreur"}: ${txt.slice(0, 200)}`);
                  load();
                }}
              >
                Tick Monde
              </button>
              <button
                type="button"
                style={{ padding: "5px 10px", fontSize: 12, background: "#16a34a", color: "white", border: "none", borderRadius: 6, cursor: "pointer" }}
                onClick={async () => {
                  setManualError("");
                  const r = await fetch(`${API_BASE}/api/nexus-game/worlds/${world.id}/sync-production`, { method: "POST", credentials: "same-origin" });
                  const txt = await r.text();
                  setManualError(`[Sync Prod ${world.name || world.id}] ${r.ok ? "OK" : "Erreur"}: ${txt.slice(0, 300)}`);
                  load();
                }}
              >
                Sync Production
              </button>
              <button
                type="button"
                style={{ padding: "5px 10px", fontSize: 12, background: "#7c3aed", color: "white", border: "none", borderRadius: 6, cursor: "pointer" }}
                onClick={async () => {
                  setManualError("");
                  const r = await fetch(`${API_BASE}/api/nexus-game/worlds/${world.id}/generate-event`, { method: "POST", credentials: "same-origin" });
                  const txt = await r.text();
                  setManualError(`[Event ${world.name || world.id}] ${r.ok ? "OK" : "Erreur"}: ${txt.slice(0, 300)}`);
                  load();
                }}
              >
                Générer Événement IA
              </button>
            </div>
          ))}
          {state.worlds.length === 0 ? <p style={{ fontSize: 12, color: "#94a3b8" }}>Aucun monde chargé.</p> : null}
        </div>
      </section>

      <section className="panel">
        <h2>Postes de controle Nexus World</h2>
        <div className="quick-grid control-links">
          {commandLinks.map((link) => (
            <Link className="quick-link" href={link.href} key={link.href}>
              <strong>{link.title}</strong>
              <span>{link.detail}</span>
            </Link>
          ))}
        </div>
      </section>

      {selectedPlayer ? (
        <div
          onClick={() => setSelectedPlayer(null)}
          style={{
            position: "fixed",
            inset: 0,
            zIndex: 80,
            background: "rgba(2, 6, 23, 0.78)",
            backdropFilter: "blur(10px)",
            padding: 20,
            overflow: "auto",
          }}
        >
          <div
            className="panel"
            onClick={(event) => event.stopPropagation()}
            style={{
              maxWidth: 1280,
              margin: "0 auto",
              border: "1px solid rgba(34, 211, 238, 0.45)",
              boxShadow: "0 24px 80px rgba(8, 47, 73, 0.45)",
            }}
          >
            <div style={{ display: "flex", justifyContent: "space-between", gap: 16, alignItems: "flex-start", marginBottom: 16 }}>
              <div>
                <p className="eyebrow">Nexus player telemetry</p>
                <h2 style={{ marginBottom: 4 }}>{selectedPlayer.pseudo || playerDetail?.profile?.pseudo || "Joueur"}</h2>
                <p className="hint">
                  {selectedPlayer.worldName || playerDetail?.identity?.world?.name || "-"} /{" "}
                  {selectedPlayer.continentName || playerDetail?.identity?.continent?.name || "-"} /{" "}
                  {playerDetail?.city?.name || selectedPlayer.city_name || selectedPlayer.cityName || "ville inconnue"}
                </p>
              </div>
              <button
                type="button"
                onClick={() => setSelectedPlayer(null)}
                style={{ border: 0, borderRadius: 6, padding: "8px 12px", background: "#0f172a", color: "white", fontWeight: 800 }}
              >
                Fermer
              </button>
            </div>

            {detailLoading ? <div className="alert">Chargement du dossier joueur complet...</div> : null}
            {detailError ? <div className="alert error">{detailError}</div> : null}

            {playerDetail ? (
              <>
                <section className="control-metric-grid" style={{ marginBottom: 16 }}>
                  <div className="metric-tile"><span>Population</span><strong>{formatNumber(Number(playerDetail.city?.population || 0))}<small> / {formatNumber(Number(playerDetail.city?.populationCapacity || 0))}</small></strong><small>+{Number(playerDetail.city?.populationGrowthPerHour || playerDetail.city?.cityStats?.populationGrowthPerHour || 0).toFixed(2)} / h</small></div>
                  <div className="metric-tile"><span>Energie</span><strong>{formatNumber(Number(playerDetail.city?.energyBalance || 0))}</strong><small>{formatNumber(Number(playerDetail.city?.energyProduction || 0))} prod / {formatNumber(Number(playerDetail.city?.energyConsumption || 0))} conso</small></div>
                  <div className="metric-tile"><span>Morale</span><strong>{formatNumber(Number(playerDetail.city?.morale || 0))}</strong></div>
                  <div className="metric-tile"><span>Securite</span><strong>{formatNumber(Number(playerDetail.city?.security || 0))}</strong></div>
                  <div className="metric-tile"><span>Batiments</span><strong>{formatNumber(detailArray(playerDetail.buildings).length)}</strong><small>{formatNumber(detailArray(playerDetail.queues?.construction).length)} en chantier</small></div>
                  <div className="metric-tile"><span>Unites</span><strong>{formatNumber(detailArray(playerDetail.units).reduce((sum: number, item: any) => sum + Number(item?.playerUnit?.count || item?.count || 0), 0))}</strong></div>
                  <div className="metric-tile"><span>Recherches</span><strong>{formatNumber(detailArray(playerDetail.research).length)}</strong></div>
                  <div className="metric-tile"><span>Agents IA</span><strong>{formatNumber(detailArray(playerDetail.agents).length)}</strong></div>
                </section>

                <section className="split">
                  <div className="panel">
                    <h2>Identite et ville</h2>
                    <div className="signal-list">
                      <div><span>Profile ID</span><strong>{playerDetail.profile?.id}</strong></div>
                      <div><span>User ID</span><strong>{playerDetail.profile?.user_id || playerDetail.profile?.userId}</strong></div>
                      <div><span>Faction</span><strong>{playerDetail.identity?.faction?.name || "-"}</strong></div>
                      <div><span>Niveau</span><strong>{playerDetail.profile?.level || 1}</strong></div>
                      <div><span>Puissance</span><strong>{formatNumber(Number(playerDetail.profile?.power || 0))}</strong></div>
                      <div><span>Stockage</span><strong>{formatNumber(Number(playerDetail.city?.cityStats?.storageCapacity || 0))}</strong></div>
                      <div><span>Nourriture</span><strong>{formatNumber(Number(playerDetail.city?.cityStats?.foodBalance || 0))}</strong></div>
                      <div><span>Derniere prod</span><strong>{playerDetail.city?.cityStats?.lastProductionSyncAt ? formatDate(playerDetail.city.cityStats.lastProductionSyncAt) : "-"}</strong></div>
                      <div><span>Places libres</span><strong>{formatNumber(Number(playerDetail.city?.populationFree || playerDetail.city?.cityStats?.populationFree || 0))}</strong></div>
                      <div><span>Croissance population</span><strong>{Number(playerDetail.city?.populationGrowthPerHour || playerDetail.city?.cityStats?.populationGrowthPerHour || 0).toFixed(2)} / h</strong></div>
                      <div><span>Reliquat croissance</span><strong>{Number(playerDetail.city?.populationRemainder || playerDetail.city?.cityStats?.populationRemainder || 0).toFixed(3)}</strong></div>
                      <div><span>Derniere pop</span><strong>{(playerDetail.city?.lastPopulationSyncAt || playerDetail.city?.cityStats?.lastPopulationSyncAt) ? formatDate(playerDetail.city?.lastPopulationSyncAt || playerDetail.city?.cityStats?.lastPopulationSyncAt) : "-"}</strong></div>
                      <div><span>Food prod/conso</span><strong>{Number(playerDetail.city?.foodProduction || playerDetail.city?.cityStats?.foodProduction || 0).toFixed(3)} / {Number(playerDetail.city?.foodConsumption || playerDetail.city?.cityStats?.foodConsumption || 0).toFixed(3)}</strong></div>
                      <div><span>Food balance</span><strong>{Number(playerDetail.city?.foodBalance || playerDetail.city?.cityStats?.foodBalance || 0).toFixed(3)}</strong></div>
                    </div>
                  </div>

                  <div className="panel">
                    <h2>Ressources</h2>
                    <p className="hint">Modifie le stock cible: sauvegarde automatique apres 650 ms. Une valeur plus basse debite le joueur.</p>
                    <div className="table-wrap">
                      <table>
                        <thead><tr><th>Code</th><th>Stock cible</th><th>Capacite</th><th>Prod/tick</th><th>Conso/tick</th><th>Solde</th><th>Etat</th></tr></thead>
                        <tbody>
                          {detailArray(playerDetail.resources).map((resource: any) => {
                            const code = resourceCode(resource);
                            const amount = Math.trunc(resourceAmount(resource));
                            const displayedValue = resourceEdits[code] ?? String(amount);
                            const busy = Boolean(savingResources[code]);
                            return (
                              <tr key={code}>
                                <td>{code}</td>
                                <td>
                                  <div style={{ display: "flex", gap: 6, alignItems: "center", minWidth: 220 }}>
                                    <button
                                      className="secondary"
                                      type="button"
                                      disabled={busy}
                                      onClick={() => scheduleResourceAmountSave(resource, String(Math.max(0, Number(displayedValue || amount) - 100)))}
                                      style={{ padding: "6px 8px" }}
                                    >
                                      -100
                                    </button>
                                    <input
                                      type="number"
                                      min={0}
                                      step={1}
                                      value={displayedValue}
                                      disabled={busy}
                                      onChange={(event) => scheduleResourceAmountSave(resource, event.target.value)}
                                      style={{ width: 110 }}
                                    />
                                    <button
                                      className="secondary"
                                      type="button"
                                      disabled={busy}
                                      onClick={() => scheduleResourceAmountSave(resource, String(Number(displayedValue || amount) + 100))}
                                      style={{ padding: "6px 8px" }}
                                    >
                                      +100
                                    </button>
                                  </div>
                                </td>
                                <td>{formatNumber(Number(resource.capacity || 0))}</td>
                                <td>{formatNumber(Number(resource.productionPerTick || resource.production_per_tick || 0))}</td>
                                <td>{formatNumber(Number(resource.consumptionPerTick || resource.consumption_per_tick || 0))}</td>
                                <td>{formatNumber(Number(resource.balancePerTick || resource.balance_per_tick || 0))}</td>
                                <td>{busy ? "Sauvegarde..." : resourceEdits[code] !== undefined ? "En attente" : "Synchro"}</td>
                              </tr>
                            );
                          })}
                          {detailArray(playerDetail.resources).length === 0 ? <tr><td colSpan={7}>Aucune ressource joueur.</td></tr> : null}
                        </tbody>
                      </table>
                    </div>
                  </div>
                </section>

                <section className="split">
                  <div className="panel">
                    <h2>Batiments construits</h2>
                    <div className="table-wrap">
                      <table>
                        <thead><tr><th>Batiment</th><th>Niveau</th><th>Statut</th><th>Fin</th><th>Ouvriers</th></tr></thead>
                        <tbody>
                          {detailArray(playerDetail.buildings).map((item: any) => {
                            const building = item.playerBuilding || item;
                            return (
                              <tr key={`building-${building.id || contentLabel(item, "playerBuilding")}`}>
                                <td>{contentLabel(item, "playerBuilding")}</td>
                                <td>{building.level || 1}</td>
                                <td><span className={statusClass(building.isConstructing ? "building" : "running")}>{building.isConstructing ? "construction" : "actif"}</span></td>
                                <td>{building.constructionEndsAt ? formatDate(building.constructionEndsAt) : "-"}</td>
                                <td>{formatNumber(Number(building.assignedWorkers || 0))}</td>
                              </tr>
                            );
                          })}
                          {detailArray(playerDetail.buildings).length === 0 ? <tr><td colSpan={5}>Aucun batiment.</td></tr> : null}
                        </tbody>
                      </table>
                    </div>
                  </div>

                  <div className="panel">
                    <h2>Unites</h2>
                    <div className="table-wrap">
                      <table>
                        <thead><tr><th>Unite</th><th>Nombre</th><th>Type</th><th>Attaque</th><th>Defense</th></tr></thead>
                        <tbody>
                          {detailArray(playerDetail.units).map((item: any) => {
                            const unit = item.playerUnit || item;
                            return (
                              <tr key={`unit-${unit.id || contentLabel(item, "playerUnit")}`}>
                                <td>{contentLabel(item, "playerUnit")}</td>
                                <td>{formatNumber(Number(unit.count || 0))}</td>
                                <td>{item.definition?.type || "-"}</td>
                                <td>{formatNumber(Number(item.definition?.attackBase || 0))}</td>
                                <td>{formatNumber(Number(item.definition?.defenseBase || 0))}</td>
                              </tr>
                            );
                          })}
                          {detailArray(playerDetail.units).length === 0 ? <tr><td colSpan={5}>Aucune unite.</td></tr> : null}
                        </tbody>
                      </table>
                    </div>
                  </div>
                </section>

                <section className="split">
                  <div className="panel">
                    <h2>Recherches</h2>
                    <div className="table-wrap">
                      <table>
                        <thead><tr><th>Recherche</th><th>Branche</th><th>Tier</th><th>Terminee</th></tr></thead>
                        <tbody>
                          {detailArray(playerDetail.research).map((item: any) => {
                            const research = item.playerResearch || item;
                            return (
                              <tr key={`research-${research.id || contentLabel(item, "playerResearch")}`}>
                                <td>{contentLabel(item, "playerResearch")}</td>
                                <td>{item.definition?.branch || "-"}</td>
                                <td>{item.definition?.tier || "-"}</td>
                                <td>{research.completedAt ? formatDate(research.completedAt) : "-"}</td>
                              </tr>
                            );
                          })}
                          {detailArray(playerDetail.research).length === 0 ? <tr><td colSpan={4}>Aucune recherche terminee.</td></tr> : null}
                        </tbody>
                      </table>
                    </div>
                  </div>

                  <div className="panel">
                    <h2>Agents, plans et IA serveur</h2>
                    <div className="signal-list">
                      <div><span>Avatars</span><strong>{detailArray(playerDetail.avatars).length}</strong></div>
                      <div><span>Compagnons</span><strong>{detailArray(playerDetail.companions).length}</strong></div>
                      <div><span>Plans journaliers</span><strong>{detailArray(playerDetail.dailyPlans).length}</strong></div>
                      <div><span>Grants journaliers</span><strong>{detailArray(playerDetail.dailyGrantClaims).length}</strong></div>
                      <div><span>Transactions</span><strong>{detailArray(playerDetail.resourceTransactions).length}</strong></div>
                      <div><span>Memoire IA</span><strong>{playerDetail.serverAI?.hasMemory ? "oui" : "non"}</strong></div>
                      <div><span>Attaques IA</span><strong>{detailArray(playerDetail.serverAI?.attacks).length}</strong></div>
                      <div><span>Sabotages / Espionnage</span><strong>{detailArray(playerDetail.serverAI?.sabotages).length} / {detailArray(playerDetail.serverAI?.espionage).length}</strong></div>
                      <div><span>Logs IA / actions admin</span><strong>{detailArray(playerDetail.serverAI?.aiCallLogs).length} / {detailArray(playerDetail.adminActions).length}</strong></div>
                    </div>
                  </div>
                </section>

                <section className="panel">
                  <h2>Historique ressources</h2>
                  <div className="table-wrap">
                    <table>
                      <thead><tr><th>Date</th><th>Ressource</th><th>Delta</th><th>Solde apres</th><th>Type</th><th>Source</th></tr></thead>
                      <tbody>
                        {detailArray(playerDetail.resourceTransactions).slice(0, 12).map((tx: any) => (
                          <tr key={`tx-${tx.id}`}>
                            <td>{tx.createdAt ? formatDate(tx.createdAt) : "-"}</td>
                            <td>{tx.resourceCode || tx.resource_code}</td>
                            <td>{formatNumber(Number(tx.amountDelta || tx.amount_delta || 0))}</td>
                            <td>{formatNumber(Number(tx.balanceAfter || tx.balance_after || 0))}</td>
                            <td>{tx.transactionType || tx.transaction_type || "-"}</td>
                            <td>{tx.source || "-"}</td>
                          </tr>
                        ))}
                        {detailArray(playerDetail.resourceTransactions).length === 0 ? <tr><td colSpan={6}>Aucune transaction recente.</td></tr> : null}
                      </tbody>
                    </table>
                  </div>
                </section>

                <details className="panel" open={false} style={{ marginTop: 16 }}>
                  <summary style={{ cursor: "pointer", fontWeight: 900 }}>Snapshot JSON complet envoye par le backend</summary>
                  <pre style={{ whiteSpace: "pre-wrap", overflow: "auto", maxHeight: 520, fontSize: 12, marginTop: 12 }}>
                    {JSON.stringify(playerDetail, null, 2)}
                  </pre>
                </details>
              </>
            ) : null}
          </div>
        </div>
      ) : null}
    </AdminShell>
  );
}
