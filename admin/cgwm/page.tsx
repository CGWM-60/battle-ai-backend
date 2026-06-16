'use client';

import React, { useEffect, useState } from 'react';

/**
 * CGWM Park Monitor — real-time admin dashboard (Next.js)
 * Shows cloud saves, park state, anonymized social events, sync errors, consents.
 * Never shows private conversations or raw player data.
 */

type ParkState = {
  activePlayers: number;
  activeAnimas: number;
  aloneAnimas: number;
  currentMeetings: number;
  atmosphereLevel: string;
};

type SyncRow = {
  animaId: string;
  ownerMasked: string;
  status: string;
  lastSync: string;
};

export default function CgwmAdminPage() {
  const [park, setPark] = useState<ParkState | null>(null);
  const [syncs, setSyncs] = useState<SyncRow[]>([]);
  const [events, setEvents] = useState<any[]>([]);
  const [connected, setConnected] = useState(false);

  useEffect(() => {
    // In real app: connect to /ws/admin/cgwm or SSE
    const timer = setInterval(() => {
      setPark({
        activePlayers: 14,
        activeAnimas: 53,
        aloneAnimas: 22,
        currentMeetings: 9,
        atmosphereLevel: 'luminousForest',
      });
      setSyncs([
        { animaId: 'anima_1724...', ownerMasked: 'u_****8f', status: 'synced', lastSync: new Date().toISOString() },
      ]);
      setEvents([
        { time: new Date().toISOString(), topic: 'absence', safety: 0.91, accepted: true },
      ]);
      setConnected(true);
    }, 2500);

    return () => clearInterval(timer);
  }, []);

  return (
    <div className="p-8 bg-black text-white min-h-screen font-mono">
      <h1 className="text-3xl mb-8">CGWM Park Monitor — ANIMA Cloud World Memory</h1>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {/* Global stats */}
        <div className="border border-white/20 p-4 rounded">
          <h2 className="text-xl mb-4">État global du Parc</h2>
          {park ? (
            <ul className="space-y-1 text-sm">
              <li>Joueurs actifs : {park.activePlayers}</li>
              <li>Amina présentes : {park.activeAnimas}</li>
              <li>Amina seules au parc : {park.aloneAnimas}</li>
              <li>Rencontres en cours : {park.currentMeetings}</li>
              <li>Atmosphère : {park.atmosphereLevel}</li>
            </ul>
          ) : <p>Connexion...</p>}
        </div>

        {/* Sync table */}
        <div className="border border-white/20 p-4 rounded col-span-1 md:col-span-2">
          <h2 className="text-xl mb-4">Synchronisations Cloud (masqué)</h2>
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left border-b border-white/20">
                <th className="py-1">Anima</th><th>Utilisateur</th><th>Statut</th><th>Dernière sync</th>
              </tr>
            </thead>
            <tbody>
              {syncs.map((s, i) => (
                <tr key={i} className="border-b border-white/10">
                  <td>{s.animaId}</td>
                  <td>{s.ownerMasked}</td>
                  <td>{s.status}</td>
                  <td>{s.lastSync}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {/* Social events (anonymized) */}
        <div className="border border-white/20 p-4 rounded">
          <h2 className="text-xl mb-4">Événements sociaux anonymisés</h2>
          <ul className="text-sm space-y-2">
            {events.map((e, i) => (
              <li key={i}>{e.time} — {e.topic} (safety {e.safety}) {e.accepted ? '✓' : '✗'}</li>
            ))}
          </ul>
          <p className="text-[10px] text-white/40 mt-4">Aucune conversation privée n’est jamais visible ici.</p>
        </div>

        {/* Real-time note */}
        <div className="col-span-1 md:col-span-2 text-xs text-white/50">
          {connected ? 'WebSocket / SSE connecté — mise à jour temps réel' : 'Connexion temps réel...'}
          <br />
          RGPD : seul l’admin technique peut voir ces tableaux. Les joueurs contrôlent tout via l’app (opt-in, export, suppression).
        </div>
      </div>
    </div>
  );
}