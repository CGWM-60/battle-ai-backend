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
      if (!res.ok) {
        throw new Error(payload.message || payload.error || `HTTP ${res.status}`);
      }
      // success
      setGenKey(""); // clear key
      reload();
      alert(payload.message || `Génération terminée : ${payload.generated || 0} affaires.`);
    } catch (e: any) {
      setError(e.message || "Génération manuelle échouée");
    } finally {
      setBusy(false);
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
                Générer les 10 affaires
              </button>
              <button className="ghost" type="button" onClick={reload} disabled={busy}>
                <RefreshCw size={16} aria-hidden /> Rafraîchir
              </button>
            </div>
            <p className="hint" style={{ marginTop: 8 }}>Utilise exactement le même mécanisme que le cron (provider/model + appel IA). Les affaires apparaissent prêtes à charger dans le module Tribunal IA du jeu.</p>
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
                        className={selected?.id === c.id ? "selected" : ""}
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
    return <aside className="panel rp-detail empty">Sélectionnez une affaire générée pour voir tous les paramètres et l'histoire complète.</aside>;
  }

  return (
    <aside className="panel rp-detail">
      <header>
        <span className={`status ${cas.status}`}>{cas.status}</span>
        <h2>#{cas.id} — Niveau {cas.level} ({cas.difficulty}) — {cas.title}</h2>
        <p>{cas.summary}</p>
      </header>

      <div className="rp-edit-strip" style={{ fontSize: '0.9em' }}>
        <div><strong>Mode:</strong> {cas.mode}</div>
        <div><strong>Rôle suggéré:</strong> {cas.playerRoleSuggestion}</div>
        <div><strong>Provider:</strong> {cas.providerType} / {cas.providerModel}</div>
        <div><strong>Durée estimée:</strong> {cas.estimatedDurationMinutes} min</div>
        <div><strong>Batch:</strong> {cas.generationBatchID || '-'}</div>
        <div><strong>Générée par cron:</strong> {cas.generatedByCron ? "oui" : "non (manuel)"}</div>
      </div>

      <dl className="rp-facts">
        <dt>Accusation</dt>
        <dd className="prewrap">{cas.accusationPosition}</dd>
        <dt>Défense</dt>
        <dd className="prewrap">{cas.defensePosition}</dd>
        <dt>Type / Ton</dt>
        <dd>{cas.caseType} / {cas.tone}</dd>
        <dt>Créée</dt>
        <dd>{formatDate(cas.createdAt)}</dd>
      </dl>

      <section className="rp-read-block">
        <h3>Témoins ({Array.isArray(cas.witnesses) ? cas.witnesses.length : 0})</h3>
        <div className="rp-arc-list">
          {(Array.isArray(cas.witnesses) ? cas.witnesses : []).map((w: any, idx: number) => (
            <article key={idx} className="rp-arc">
              <h4>{w.name || w.Name || "Témoin"} — {w.role || w.Role}</h4>
              <small>Crédibilité: {w.credibility || w.Credibility || "?"} | Biais: {w.bias || w.Bias}</small>
              <p>{w.knowledge || w.Knowledge || w.personality}</p>
            </article>
          ))}
          {!Array.isArray(cas.witnesses) || cas.witnesses.length === 0 ? <p>Aucun témoin.</p> : null}
        </div>
      </section>

      <section className="rp-read-block">
        <h3>Preuves ({Array.isArray(cas.evidence) ? cas.evidence.length : 0})</h3>
        <ul>
          {(Array.isArray(cas.evidence) ? cas.evidence : []).map((e: any, idx: number) => (
            <li key={idx}>
              <strong>{e.title || e.Title}</strong> — {e.evidenceType || e.EvidenceType} (force {e.strength || e.Strength}, fiabilité {e.reliability || e.Reliability})<br />
              <small>Supporte: {e.supportsSide || e.SupportsSide} | {e.description || e.Description}</small>
            </li>
          ))}
        </ul>
      </section>

      <section className="rp-read-block">
        <h3>Déclarations du témoignage ({Array.isArray(cas.testimonyStatements) ? cas.testimonyStatements.length : 0})</h3>
        <ol>
          {(Array.isArray(cas.testimonyStatements) ? cas.testimonyStatements : []).map((s: any, idx: number) => (
            <li key={idx}>
              {s.content || s.Content || s.witnessName} {s.isAttackable ? "(attaquable)" : ""}
              <small> — {Array.isArray(s.tags) ? s.tags.join(", ") : s.tags}</small>
            </li>
          ))}
        </ol>
      </section>

      <section className="rp-read-block">
        <h3>Contradictions attendues</h3>
        <ul>
          {(Array.isArray(cas.expectedContradictions) ? cas.expectedContradictions : []).map((k: any, idx: number) => (
            <li key={idx}>{k.statementContent || k.StatementContent} ↔ {k.evidenceTitle || k.EvidenceTitle} ({k.contradictionType})</li>
          ))}
          {(!Array.isArray(cas.expectedContradictions) || cas.expectedContradictions.length === 0) && <li>Aucune listée.</li>}
        </ul>
      </section>

      <section className="rp-read-block">
        <h3>Tags</h3>
        <div>{(Array.isArray(cas.tags) ? cas.tags : []).join(", ") || "-"}</div>
      </section>
    </aside>
  );
}
