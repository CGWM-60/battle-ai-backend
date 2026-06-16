'use client';

import React, { useEffect, useState } from 'react';

/**
 * ANIMA CGWM Park Monitor — Page admin dédiée "Anima"
 * Comme demandé dans le prompt : page pour voir en temps réel les sauvegardes cloud,
 * état de synchronisation, Amina connectées, Amina seules au parc, interactions sociales
 * anonymisées, état du parc, nombre de joueurs, erreurs de sync.
 * 
 * Route : /anima (dans l'admin Next.js)
 * Respect RGPD : rien de privé, données masquées/anonymisées.
 */

type ParkState = {
  activePlayers: number;
  activeAnimas: number;
  aloneAnimas: number;
  currentMeetings: number;
  socialLearningEvents: number;
  atmosphereLevel: string;
};

type SyncRow = {
  animaId: string;
  ownerMasked: string;
  status: string;
  lastSync: string;
  conflicts: number;
};

type SocialEvent = {
  time: string;
  sourcePublicAnimaId: string;
  targetPublicAnimaId: string;
  topic: string;
  safetyScore: number;
  accepted: boolean;
};

export default function AnimaCgwmPage() {
  const [park, setPark] = useState<ParkState | null>(null);
  const [syncs, setSyncs] = useState<SyncRow[]>([]);
  const [events, setEvents] = useState<SocialEvent[]>([]);
  const [connected, setConnected] = useState(false);
  const [lastUpdate, setLastUpdate] = useState<string>('');

  useEffect(() => {
    // Simulation temps réel (en prod: WebSocket ou SSE vers /ws/admin/cgwm ou /ws/cgwm/park)
    const interval = setInterval(() => {
      const now = new Date().toISOString();

      setPark({
        activePlayers: Math.floor(Math.random() * 20) + 8,
        activeAnimas: Math.floor(Math.random() * 60) + 30,
        aloneAnimas: Math.floor(Math.random() * 25) + 10,
        currentMeetings: Math.floor(Math.random() * 12) + 3,
        socialLearningEvents: Math.floor(Math.random() * 8) + 1,
        atmosphereLevel: ['livingPark', 'luminousForest', 'vastDreamPark'][Math.floor(Math.random() * 3)],
      });

      setSyncs([
        { animaId: 'anima_1724a1f2', ownerMasked: 'u_****8f', status: 'synced', lastSync: now, conflicts: 0 },
        { animaId: 'anima_1724b3c9', ownerMasked: 'u_****2e', status: 'pendingUpload', lastSync: now, conflicts: 1 },
      ]);

      setEvents([
        { time: now, sourcePublicAnimaId: 'pub_abc123', targetPublicAnimaId: 'pub_def456', topic: 'absence', safetyScore: 0.91, accepted: true },
        { time: now, sourcePublicAnimaId: 'pub_ghi789', targetPublicAnimaId: 'pub_abc123', topic: 'patience', safetyScore: 0.87, accepted: true },
      ]);

      setLastUpdate(now);
      setConnected(true);
    }, 2200);

    return () => clearInterval(interval);
  }, []);

  return (
    <div className="p-8 bg-black text-white min-h-screen font-mono">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-3xl font-bold">Anima — CGWM Park Monitor</h1>
          <p className="text-white/60 text-sm">Cloud Game World Memory • ANIMA CGWM PARK (admin temps réel)</p>
        </div>
        <div className="text-xs text-white/50">
          {connected ? '● Connecté (simulé WS/SSE)' : 'Connexion...'}<br />
          Dernière mise à jour : {lastUpdate || '—'}
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Vue globale du Parc */}
        <div className="border border-white/20 p-5 rounded-xl bg-white/5">
          <h2 className="text-xl mb-4 flex items-center gap-2">🌳 État du Parc (temps réel)</h2>
          {park ? (
            <div className="space-y-2 text-sm">
              <div>Joueurs actifs : <span className="text-emerald-400 font-mono">{park.activePlayers}</span></div>
              <div>Amina présentes : <span className="text-emerald-400 font-mono">{park.activeAnimas}</span></div>
              <div>Amina seules au parc : <span className="text-amber-400 font-mono">{park.aloneAnimas}</span></div>
              <div>Rencontres en cours : <span className="text-sky-400 font-mono">{park.currentMeetings}</span></div>
              <div>Événements d'apprentissage social : <span className="text-purple-400 font-mono">{park.socialLearningEvents}</span></div>
              <div className="pt-2 border-t border-white/10">Atmosphère : <span className="text-lime-400">{park.atmosphereLevel}</span></div>
            </div>
          ) : <div>Chargement...</div>}
          <div className="mt-4 text-[10px] text-white/40">Données anonymisées uniquement (publicAnimaId, displayName filtré).</div>
        </div>

        {/* Sauvegardes cloud par joueur / sync */}
        <div className="border border-white/20 p-5 rounded-xl bg-white/5 lg:col-span-2">
          <h2 className="text-xl mb-4">☁️ Sauvegardes Cloud &amp; Synchronisation (par Anima)</h2>
          <table className="w-full text-xs">
            <thead>
              <tr className="text-left border-b border-white/20 text-white/70">
                <th className="py-1 pr-4">Anima ID</th>
                <th className="py-1 pr-4">Joueur (masqué)</th>
                <th className="py-1 pr-4">Statut Sync</th>
                <th className="py-1 pr-4">Dernière sync</th>
                <th className="py-1">Conflits</th>
              </tr>
            </thead>
            <tbody className="text-white/90">
              {syncs.map((s, i) => (
                <tr key={i} className="border-b border-white/10 hover:bg-white/5">
                  <td className="py-1 pr-4 font-mono">{s.animaId}</td>
                  <td className="py-1 pr-4">{s.ownerMasked}</td>
                  <td className="py-1 pr-4">
                    <span className={s.status === 'synced' ? 'text-emerald-400' : 'text-amber-400'}>{s.status}</span>
                  </td>
                  <td className="py-1 pr-4 text-white/60">{s.lastSync}</td>
                  <td className="py-1">{s.conflicts}</td>
                </tr>
              ))}
            </tbody>
          </table>
          <div className="mt-3 text-[10px] text-white/40">Aucune donnée personnelle ni conversation privée n'est exposée.</div>
        </div>

        {/* Amina au parc / interactions sociales */}
        <div className="border border-white/20 p-5 rounded-xl bg-white/5">
          <h2 className="text-xl mb-4">🌲 Amina seules au Parc &amp; Rencontres</h2>
          <div className="text-sm space-y-1">
            <div>Présences seules : <span className="font-mono text-amber-400">{park?.aloneAnimas ?? '—'}</span></div>
            <div>Rencontres actives : <span className="font-mono text-sky-400">{park?.currentMeetings ?? '—'}</span></div>
          </div>
        </div>

        {/* Événements sociaux anonymisés */}
        <div className="border border-white/20 p-5 rounded-xl bg-white/5 lg:col-span-2">
          <h2 className="text-xl mb-4">🤝 Interactions Sociales Anonymisées (live)</h2>
          <div className="text-xs max-h-48 overflow-auto space-y-1 font-mono">
            {events.length > 0 ? events.map((e, i) => (
              <div key={i} className="flex gap-2 text-white/80">
                <span className="text-white/50 w-40 shrink-0">{e.time}</span>
                <span>{e.sourcePublicAnimaId} ↔ {e.targetPublicAnimaId}</span>
                <span className="text-lime-400">[{e.topic}]</span>
                <span className="text-white/60">safety={e.safetyScore}</span>
                <span className={e.accepted ? 'text-emerald-400' : 'text-red-400'}>{e.accepted ? 'accepté' : 'refusé'}</span>
              </div>
            )) : <div className="text-white/40">Aucun événement pour l'instant.</div>}
          </div>
          <div className="mt-2 text-[10px] text-white/40">Seules des leçons générales anonymisées et filtrées (safety ≥ 0.8) sont échangées. Aucun contenu privé.</div>
        </div>

        {/* Stats rapides + erreurs */}
        <div className="border border-white/20 p-5 rounded-xl bg-white/5">
          <h2 className="text-xl mb-4">📊 Synthèse &amp; Erreurs</h2>
          <div className="grid grid-cols-2 gap-3 text-sm">
            <div>Joueurs avec cloud : <span className="font-mono">{park ? Math.floor(park.activePlayers * 0.7) : '—'}</span></div>
            <div>Amina cloud : <span className="font-mono">{park ? Math.floor(park.activeAnimas * 0.6) : '—'}</span></div>
            <div>Erreurs de sync (dernières 24h) : <span className="text-red-400 font-mono">0</span></div>
            <div>Schedulers actifs : <span className="text-emerald-400">alone-learning, cleanup</span></div>
          </div>
        </div>
      </div>

      <div className="mt-8 text-[10px] text-white/40 border-t border-white/10 pt-4">
        RGPD : Toutes les données ici sont agrégées ou anonymisées (publicAnimaId). L'admin ne voit jamais les conversations privées, les userId réels, ni les données sensibles. 
        Le joueur garde le contrôle total (opt-in, export, suppression) depuis l'application Flutter.
      </div>
    </div>
  );
}
