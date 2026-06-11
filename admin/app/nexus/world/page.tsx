"use client";

import Link from "next/link";
import { useCallback, useEffect, useMemo, useState } from "react";
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

export default function NexusWorldControlPage() {
  const [state, setState] = useState<CockpitState>(emptyState);
  const [loading, setLoading] = useState(true);
  const [manualError, setManualError] = useState("");
  const [actionMessage, setActionMessage] = useState("");
  const [deletingProfileId, setDeletingProfileId] = useState<number | null>(null);

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

  const commandLinks = [
    { href: "/nexus/mmo/", title: "Console MMO", detail: "Avatars, factions, compagnons, mondes et generation manuelle." },
    { href: "/nexus/mmo/buildings", title: "Batiments", detail: "Contraintes, couts, durees, effets et images de construction." },
    { href: "/nexus/mmo/units", title: "Unites", detail: "Stats, prerequisites, recrutement, assets et traductions." },
    { href: "/nexus/mmo/research", title: "Recherches", detail: "Arbre techno, durees, dependances, unlocks et descriptions." },
    { href: "/nexus/ai-server/", title: "IA serveur", detail: "Pression, bastions, generation et compteurs du noyau IA." },
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
                  <tr key={`${player.worldId}-${profileId}`}>
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
                    <td>
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
    </AdminShell>
  );
}
