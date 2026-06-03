"use client";

import { Sparkles, RefreshCw, Eye } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { AdminShell } from "../components/AdminShell";
import { ErrorState, LoadingState } from "../components/LoadState";
import { MetricGrid } from "../components/MetricGrid";
import { formatDate, formatNumber } from "../components/api";
import type { TribunalGeneratedCaseAdmin, TribunalGeneratedAdminResponse } from "../types";

const providers = ["mistral", "openai", "openrouter", "xia", "claude", "gemini"];

// New narrative quality tracking (from corrective prompt)
type NarrativeStats = {
  totalGenerated: number;
  narrativePlayable: number;
  withCrisis: number;
  withFinalReveal: number;
};

export default function TribunalAIPage() {
  const [data, setData] = useState<TribunalGeneratedAdminResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [query, setQuery] = useState("");
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [busy, setBusy] = useState(false);

  // Manual generate form state
  const [genProvider, setGenProvider] = useState("mistral");
  const [genModel, setGenModel] = useState("");
  const [genCount, setGenCount] = useState(10);
  const [genKey, setGenKey] = useState("");
  const [narrativeStats, setNarrativeStats] = useState<NarrativeStats | null>(null);

  const reload = () => {
    // Use the tribunal public list (auth protected) + batches via admin api if available, fallback to direct
    Promise.all([
      fetch("/api/nexus-tribunal/generated-cases?limit=100", { credentials: "same-origin" }).then(r => r.json()),
      fetch("/admin/api/tribunal-generated", { credentials: "same-origin" }).then(r => r.json().catch(() => null)),
      fetch("/api/nexus-tribunal/admin/generated-cases/narrative-stats", { credentials: "same-origin" }).then(r => r.json().catch(() => null)),
    ]).then(([listPayload, adminPayload, narrStats]) => {
      if (narrStats) setNarrativeStats(narrStats);
      const cases: TribunalGeneratedCaseAdmin[] = (adminPayload?.cases || listPayload?.data || []).map((c: any) => ({
        id: c.id,
        createdAt: c.createdAt || c.CreatedAt,
        title: c.title,
        summary: c.summary,
        caseType: c.caseType,
        level: c.level,
        difficulty: c.difficulty,
        estimatedDurationMinutes: c.estimatedDurationMinutes,
        mode: c.mode,
        tone: c.tone,
        playerRoleSuggestion: c.playerRoleSuggestion,
        accusationPosition: c.accusationPosition,
        defensePosition: c.defensePosition,
        tags: c.tags || [],
        witnesses: c.witnesses || [],
        evidence: c.evidence || [],
        testimonyStatements: c.testimonyStatements || c["testimonyStatements"] || [],
        expectedContradictions: c.expectedContradictions || c["expectedContradictions"] || [],
        status: c.status,
        isPlayable: c.isPlayable,
        isPublished: c.isPublished,
        generatedByCron: c.generatedByCron,
        providerType: c.providerType,
        providerModel: c.model || c.providerModel,
        generationBatchID: c.generationBatchID || c.GenerationBatchID,
        // Preserve full narrative data for the detail view
        ...c, // cast, acts, scenes, progressionRules, failureRules, realTruth, etc. will be available
      }));
      const batches = adminPayload?.batches || [];
      const stats = adminPayload?.stats || { totalGenerated: cases.length, published: cases.filter(c => c.isPublished).length };
      setData({ cases, batches, stats });
      setSelectedId((current) => current ?? cases[0]?.id ?? null);
    }).catch((err: Error) => setError(err.message));
  };

  useEffect(() => {
    reload();
  }, []);

  const filtered = useMemo(() => {
    const needle = query.trim().toLowerCase();
    const list = data?.cases ?? [];
    if (!needle) return list;
    return list.filter((c) =>
      [c.title, String(c.id), c.difficulty, c.caseType, c.mode, c.providerType].some(v =>
        String(v).toLowerCase().includes(needle)
      )
    );
  }, [data, query]);

  const selected = filtered.find((c) => c.id === selectedId) ?? filtered[0] ?? null;

  async function generateNow() {
    if (!genModel || !genKey) {
      setError("Provider, modèle et clé API sont requis pour générer manuellement.");
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const res = await fetch("/admin/generate/tribunal", {
        method: "POST",
        credentials: "same-origin",
        headers: { "Content-Type": "application/x-www-form-urlencoded" },
        body: new URLSearchParams({
          provider: genProvider,
          model: genModel,
          api_key: genKey,
          count: String(genCount),
        }).toString(),
      });
      const payload = await res.json().catch(() => ({}));
      if (!res.ok || payload.success === false) {
        const msg = payload.error || payload.message || `HTTP ${res.status}`;
        const fullMsg = payload.batchId ? `${msg} (batch #${payload.batchId})` : msg;
        throw new Error(fullMsg);
      }
      // success
      setGenKey(""); // clear key
      reload();
      alert(payload.message || `Génération terminée : ${payload.generated || 0} affaires.`);
    } catch (e: any) {
      const base = e.message || "Génération manuelle échouée";
      setError(`${base}\n\n→ Vérifie les logs Dokploy (recherche [tribunal-generate]) pour la réponse brute de l'IA et le détail précis. Le batch créé contient aussi l'ErrorMessage.`);
    } finally {
      setBusy(false);
      reload(); // always refresh batches so user can see detailed ErrorMessage in the list
    }
  }

  return (
    <AdminShell title="Tribunal IA" description="Affaires générées automatiquement (10 par cycle aux horaires du cron) ou manuellement. Chaque histoire avec ses paramètres complets (témoins, preuves, témoignages, contradictions).">
      {error ? <ErrorState message={error} /> : null}
      {!data && !error ? <LoadingState /> : null}

      {data ? (
        <>
          <MetricGrid
            items={[
              { label: "Affaires générées", value: formatNumber(data.stats.totalGenerated) },
              { label: "Publiées / Prêtes", value: formatNumber(data.stats.published), tone: "good" },
              { label: "Batches", value: formatNumber(data.batches?.length || 0) },
            ]}
          />

          {narrativeStats && (
            <div className="rounded border border-cyan-500/30 bg-[#0b0f1a] p-3 mb-3 text-xs">
              <div className="text-cyan-400 mb-1 font-semibold">Qualité Narrative (Phoenix-like / Correctif)</div>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-x-4 gap-y-1">
                <div>Narratifs jouables: <span className="font-mono text-white">{narrativeStats.narrativePlayable}</span></div>
                <div>Avec crisis: <span className="font-mono text-white">{narrativeStats.withCrisis}</span></div>
                <div>Avec final reveal: <span className="font-mono text-white">{narrativeStats.withFinalReveal}</span></div>
                <div>Total: <span className="font-mono text-white">{narrativeStats.totalGenerated}</span></div>
              </div>
              <div className="text-[10px] text-white/40 mt-1">Voir specs: hasIntro, actsCount, scenesCount, progressionRulesCount, hasCrisis, hasNexusBridge etc.</div>
            </div>
          )}

          {/* Manual generation - like the quest generate buttons, but prominent for Tribunal */}
          <section className="panel" style={{ marginBottom: 16 }}>
            <h2>Générer 10 affaires manuellement</h2>
            <div className="rp-edit-strip" style={{ display: 'flex', gap: 8, flexWrap: 'wrap', alignItems: 'flex-end' }}>
              <label>
                Provider
                <select value={genProvider} onChange={(e) => setGenProvider(e.target.value)}>
                  {providers.map(p => <option key={p} value={p}>{p}</option>)}
                </select>
              </label>
              <label>
                Modèle
                <input value={genModel} onChange={(e) => setGenModel(e.target.value)} placeholder="mistral-large-latest" />
              </label>
              <label>
                Nombre
                <input type="number" value={genCount} onChange={(e) => setGenCount(parseInt(e.target.value) || 10)} min={1} max={20} />
              </label>
              <label style={{ flex: 1, minWidth: 220 }}>
                Clé API (non stockée)
                <input type="password" value={genKey} onChange={(e) => setGenKey(e.target.value)} placeholder="sk-..." />
              </label>
              <button className="secondary" type="button" onClick={generateNow} disabled={busy || !genModel || !genKey}>
                <Sparkles size={16} aria-hidden />
                Générer les 10 affaires (1 par 1)
              </button>
              <button
                className="ghost"
                type="button"
                onClick={async () => {
                  if (!confirm("Purger toutes les affaires générées ?")) return;
                  setBusy(true);
                  try {
                    const res = await fetch("/admin/generate/tribunal/purge", { method: "POST", credentials: "same-origin" });
                    const p = await res.json().catch(() => ({}));
                    if (!res.ok || p.success === false) throw new Error(p.error || "Purge failed");
                    alert(p.message || "Affaires purgées.");
                  } catch (e: any) {
                    alert("Erreur purge: " + (e.message || e));
                  } finally {
                    setBusy(false);
                    reload();
                  }
                }}
                disabled={busy}
              >
                Purger les affaires
              </button>
              <button className="ghost" type="button" onClick={reload} disabled={busy}>
                <RefreshCw size={16} aria-hidden /> Rafraîchir
              </button>
            </div>
            <p className="hint" style={{ marginTop: 8 }}>Utilise exactement le même mécanisme que le cron (provider/model + appel IA). Les affaires apparaissent prêtes à charger dans le module Tribunal IA du jeu.</p>
            {/* Precise errors: show recent batches with their ErrorMessage (very useful for 0 cases / AI response issues) */}
            {data.batches && data.batches.length > 0 && (
              <div className="panel" style={{ marginTop: 12, fontSize: '0.85em' }}>
                <strong>Derniers batches (clique Rafraîchir après generate pour voir ErrorMessage détaillé)</strong>
                <ul style={{ margin: '8px 0', paddingLeft: 16 }}>
                  {data.batches.slice(0, 5).map((b: any) => (
                    <li key={b.id}>
                      #{b.id} — {b.status} — {b.providerType}/{b.providerModel} — générés: {b.generatedCount || 0}
                      {b.errorMessage && <div style={{ color: '#f66', whiteSpace: 'pre-wrap' }}>Erreur: {b.errorMessage}</div>}
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </section>

          <section className="rp-toolbar">
            <div>
              <Eye size={18} aria-hidden />
              <strong>{formatNumber(filtered.length)} affaires affichées</strong>
            </div>
            <input
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Filtrer par id, titre, niveau, difficulté, provider..."
            />
          </section>

          <section className="rp-admin-layout">
            <div className="panel rp-table-panel">
              <div className="table-wrap">
                <table className="rp-table">
                  <thead>
                    <tr>
                      <th>ID</th>
                      <th>Niv / Diff</th>
                      <th>Titre / Histoire</th>
                      <th>Mode</th>
                      <th>Provider</th>
                      <th>Statut</th>
                      <th>Créée</th>
                    </tr>
                  </thead>
                  <tbody>
                    {filtered.map((c) => (
                      <tr
                        key={c.id}
                        className={`cursor-pointer hover:bg-white/5 ${selected?.id === c.id ? "selected" : ""}`}
                        onClick={() => setSelectedId(c.id)}
                      >
                        <td>#{c.id}</td>
                        <td><strong>{c.level}</strong> <small>{c.difficulty}</small></td>
                        <td>
                          <strong>{c.title}</strong>
                          <small>{c.caseType} • {c.tone}</small>
                        </td>
                        <td>{c.mode}</td>
                        <td><small>{c.providerType} / {c.providerModel}</small></td>
                        <td><span className={`status ${c.status}`}>{c.status}</span></td>
                        <td>{formatDate(c.createdAt)}</td>
                      </tr>
                    ))}
                    {!filtered.length && (
                      <tr><td colSpan={7}>Aucune affaire générée pour le moment. Utilisez le bouton ci-dessus ou attendez le cron (8h/12h/20h).</td></tr>
                    )}
                  </tbody>
                </table>
              </div>
            </div>

            <TribunalCaseDetail cas={selected} />
          </section>
        </>
      ) : null}
    </AdminShell>
  );
}

function TribunalCaseDetail({ cas }: { cas: TribunalGeneratedCaseAdmin | null }) {
  if (!cas) {
    return <aside className="panel rp-detail empty">Sélectionnez une affaire générée (cliquez une ligne du tableau) pour voir la structure narrative complète (1-by-1).</aside>;
  }

  const anyCas = cas as any;

  const cast = anyCas.cast || anyCas.Cast || anyCas.characterCast || [];
  const acts = anyCas.acts || anyCas.Acts || [];
  const scenes = anyCas.scenes || anyCas.Scenes || [];
  const progRules = anyCas.progressionRules || anyCas.ProgressionRules || [];
  const failRules = anyCas.failureRules || anyCas.FailureRules || [];
  const crisis = anyCas.crisisMoment || anyCas.CrisisMoment;
  const verdicts = anyCas.possibleVerdicts || anyCas.PossibleVerdicts || [];
  const epilogue = anyCas.epilogue || anyCas.Epilogue || anyCas.finalReveal || "";
  const nexus = anyCas.nexusBridgeHints || anyCas.NexusBridgeHints || [];
  const realTruth = anyCas.realTruth || anyCas.RealTruth || "";
  const publicTruth = anyCas.publicTruth || anyCas.PublicTruth || "";

  return (
    <aside className="panel rp-detail" style={{ maxHeight: '82vh', overflowY: 'auto', fontSize: '0.82em' }}>
      <header>
        <span className={`status ${cas.status}`}>{cas.status}</span>
        <h2>#{cas.id} — Niv.{cas.level} ({cas.difficulty}) — {cas.title}</h2>
        <p style={{ margin: '4px 0' }}>{cas.summary || anyCas.synopsis || anyCas.Synopsis}</p>
      </header>

      <div className="rp-edit-strip" style={{ fontSize: '0.78em', gap: '4px 12px' }}>
        <div><strong>Mode:</strong> {cas.mode}</div>
        <div><strong>Provider:</strong> {cas.providerType}/{cas.providerModel}</div>
        <div><strong>Durée:</strong> {cas.estimatedDurationMinutes}min</div>
        <div><strong>Batch:</strong> {cas.generationBatchID || '-'}</div>
        <div><strong>Cron:</strong> {cas.generatedByCron ? 'oui' : 'manuel'}</div>
      </div>

      {realTruth && <div style={{ marginTop: 6 }}><strong>Vérité réelle:</strong> <span style={{ opacity: .85 }}>{realTruth}</span></div>}
      {publicTruth && <div><strong>Vérité publique:</strong> <span style={{ opacity: .85 }}>{publicTruth}</span></div>}

      {/* Cast */}
      {cast.length > 0 && (
        <div style={{ marginTop: 8 }}>
          <strong>Cast ({cast.length})</strong>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4, marginTop: 2 }}>
            {cast.map((a: any, i: number) => (
              <span key={i} style={{ background: '#222', padding: '1px 6px', borderRadius: 3, fontSize: '0.75em' }}>
                {a.actorType}: <strong>{a.name}</strong>
              </span>
            ))}
          </div>
        </div>
      )}

      {/* Acts & Scenes summary */}
      {(acts.length > 0 || scenes.length > 0) && (
        <div style={{ marginTop: 6 }}>
          <strong>Structure</strong> — {acts.length} acte(s), {scenes.length} scène(s)
          <div style={{ fontSize: '0.72em', marginTop: 2, maxHeight: 90, overflow: 'auto', background: '#111', padding: 4, borderRadius: 2 }}>
            {scenes.slice(0, 6).map((s: any, i: number) => (
              <div key={i}>{s.sceneId} [{s.sceneType}] {s.title}</div>
            ))}
            {scenes.length > 6 && <div>... +{scenes.length - 6}</div>}
          </div>
        </div>
      )}

      {/* Rules */}
      <div style={{ marginTop: 6, fontSize: '0.78em' }}>
        <strong>Règles</strong>: {progRules.length} progression (dont {progRules.filter((r: any) => r.isCritical).length} critiques) / {failRules.length} échecs
      </div>

      {crisis && <div style={{ marginTop: 4 }}><strong>Crisis:</strong> {crisis.sceneId} — {crisis.trigger}</div>}
      {verdicts.length > 0 && <div><strong>Verdicts:</strong> {verdicts.join(', ')}</div>}
      {epilogue && <div style={{ marginTop: 4, fontSize: '0.78em' }}><strong>Épilogue:</strong> {epilogue}</div>}

      {nexus.length > 0 && (
        <div style={{ marginTop: 4, fontSize: '0.75em' }}>
          <strong>Nexus hints:</strong> {nexus.map((h: any) => `${h.type}→${h.targetId || h.delta}`).join(' ; ')}
        </div>
      )}

      <div style={{ marginTop: 10, fontSize: '0.7em', opacity: 0.6, borderTop: '1px solid #333', paddingTop: 4 }}>
        Données complètes (1-by-1 generation). Sélectionnez dans le tableau pour rafraîchir.
      </div>
    </aside>
  );
}
