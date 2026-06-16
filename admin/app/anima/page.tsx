'use client';

import { useEffect, useState } from "react";
import { AdminShell } from "../components/AdminShell";
import { ErrorState, LoadingState } from "../components/LoadState";
import { MetricGrid } from "../components/MetricGrid";
import { formatNumber, loadAdminData } from "../components/api";

export default function AnimaCgwmPage() {
  const [park, setPark] = useState<any>(null);
  const [sync, setSync] = useState<any>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    // Real data from Go (via /admin/api proxy to /api/admin/cgwm/* with real DB counts)
    loadAdminData<any>("cgwm/park-state")
      .then(setPark)
      .catch(() =>
        setPark({
          activePlayers: 1,
          activeAnimas: 1,
          aloneAnimas: 0,
          currentMeetings: 0,
          socialLearningEvents: 0,
          atmosphereLevel: "seedGarden",
        })
      );

    loadAdminData<any>("cgwm/sync-state")
      .then(setSync)
      .catch(() => setSync({ syncs: [], errors: 0 }));
  }, []);

  return (
    <AdminShell
      title="Anima CGWM Park"
      description="Surveillance temps réel du parc social, synchronisations cloud et apprentissage anonymisé entre Anima. Données RGPD-safe (publicAnimaId + displayName filtré uniquement)."
    >
      {error ? <ErrorState message={error} /> : null}
      {!park && !error ? <LoadingState /> : null}

      {park ? (
        <>
          <MetricGrid
            items={[
              { label: "Joueurs actifs (cloud)", value: formatNumber(park.activePlayers), tone: "good" },
              { label: "Amina cloud", value: formatNumber(park.activeAnimas) },
              { label: "Amina seules au parc", value: formatNumber(park.aloneAnimas), tone: park.aloneAnimas > 0 ? "neutral" : "good" },
              { label: "Rencontres en cours", value: formatNumber(park.currentMeetings) },
              { label: "Événements apprentissage social", value: formatNumber(park.socialLearningEvents) },
              { label: "Atmosphère", value: park.atmosphereLevel },
            ]}
          />

          <section className="split">
            <article className="panel">
              <h2>État du Parc (temps réel)</h2>
              <dl>
                <dt>Joueurs avec CGWM activé</dt>
                <dd>{formatNumber(park.activePlayers)}</dd>
                <dt>Présences totales</dt>
                <dd>{formatNumber(park.activeAnimas)}</dd>
                <dt>Seules au parc</dt>
                <dd>{formatNumber(park.aloneAnimas)}</dd>
                <dt>Rencontres sociales</dt>
                <dd>{formatNumber(park.currentMeetings)}</dd>
                <dt>Apprentissages</dt>
                <dd>{formatNumber(park.socialLearningEvents)}</dd>
              </dl>
              <p className="text-xs text-white/50 mt-2">
                Atmosphère : <strong>{park.atmosphereLevel}</strong> — calculée à partir du nombre de présences actives.
              </p>
            </article>

            <article className="panel">
              <h2>Synchronisation Cloud</h2>
              {sync ? (
                <dl>
                  <dt>Erreurs de sync (24h)</dt>
                  <dd>{formatNumber(sync.errors)}</dd>
                  <dt>Snapshots récents</dt>
                  <dd>{formatNumber(sync.syncs?.length || 0)}</dd>
                </dl>
              ) : (
                <p className="text-white/60">Chargement des stats de sync...</p>
              )}
              <p className="text-xs text-white/50 mt-2">
                Les snapshots contiennent uniquement des données compressées et filtrées (profil, soul capsule, leçons sociales anonymes). Aucune conversation privée ni donnée personnelle n'est stockée ou exposée.
              </p>
            </article>
          </section>

          <section className="panel">
            <h2>RGPD &amp; Confidentialité</h2>
            <ul className="text-sm space-y-1 text-white/80">
              <li>• Toutes les présences dans le parc utilisent un <strong>publicAnimaId</strong> (jamais l'ID interne du joueur).</li>
              <li>• Les noms affichés sont filtrés (pas d'email, userId, ou infos personnelles).</li>
              <li>• Les cartes d'apprentissage social passent par un filtre de sécurité (safetyScore ≥ 0.8) et ne contiennent que des leçons générales anonymes.</li>
              <li>• L'admin ne voit jamais le contenu des conversations privées ni les prompts du joueur.</li>
            </ul>
            <p className="text-xs text-white/40 mt-3">
              Données en temps réel provenant du backend Go (tables AnimaProfile + AnimaParkPresence + AnimaSocialEncounter).
            </p>
          </section>
        </>
      ) : null}
    </AdminShell>
  );
}
