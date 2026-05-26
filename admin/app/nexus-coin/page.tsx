"use client";

import { Save, Trash2 } from "lucide-react";
import { useEffect, useState } from "react";
import { AdminShell } from "../components/AdminShell";
import { ErrorState, LoadingState } from "../components/LoadState";
import { MetricGrid } from "../components/MetricGrid";
import { formatNumber, loadAdminData, usdMicros } from "../components/api";
import type { NexusCoinEstimate, NexusCoinPlan, NexusCoinResponse } from "../types";

export default function NexusCoinPage() {
  const [data, setData] = useState<NexusCoinResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [savingId, setSavingId] = useState<number | null>(null);

  const reload = () => {
    loadAdminData<NexusCoinResponse>("nexus-coin").then(setData).catch((err: Error) => setError(err.message));
  };

  useEffect(() => {
    reload();
  }, []);

  async function savePlan(plan: NexusCoinPlan) {
    setSavingId(plan.id);
    setError(null);
    try {
      const response = await fetch(`/admin/api/nexus-coin/plans/${plan.id}`, {
        method: "PUT",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json", Accept: "application/json" },
        body: JSON.stringify(plan),
      });
      if (!response.ok) {
        const payload = await response.json().catch(() => ({}));
        throw new Error(payload.error ?? `HTTP ${response.status}`);
      }
      reload();
    } catch (err) {
      setError(err instanceof Error ? err.message : "save failed");
    } finally {
      setSavingId(null);
    }
  }

  async function deletePlan(plan: NexusCoinPlan) {
    setSavingId(plan.id);
    setError(null);
    try {
      const response = await fetch(`/admin/api/nexus-coin/plans/${plan.id}`, {
        method: "DELETE",
        credentials: "same-origin",
      });
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }
      reload();
    } catch (err) {
      setError(err instanceof Error ? err.message : "delete failed");
    } finally {
      setSavingId(null);
    }
  }

  function updatePlan(index: number, patch: Partial<NexusCoinPlan>) {
    if (!data) {
      return;
    }
    const nextPlans = data.plans.map((plan, current) => (current === index ? { ...plan, ...patch } : plan));
    setData({ ...data, plans: nextPlans });
  }

  return (
    <AdminShell title="Nexus Coin" description="Monnaie du jeu, estimations tokens et plans commerciaux pour credits IA.">
      {error ? <ErrorState message={error} /> : null}
      {!data && !error ? <LoadingState /> : null}
      {data ? (
        <>
          <MetricGrid
            items={[
              { label: "Appels observes", value: formatNumber(data.stats.callCount) },
              { label: "Tokens observes", value: formatNumber(data.stats.totalTokens) },
              { label: "Cout observe", value: usdMicros(data.stats.totalCostMicros) },
              { label: "Tokens / appel", value: formatNumber(data.stats.averageTokensPerCall) },
              { label: "Marge par defaut", value: `${data.stats.marginPercent}%` },
              { label: "Source cout", value: data.stats.costSource },
            ]}
          />

          <section className="panel">
            <h2>Estimations automatiques</h2>
            <p className="hint">Ces cartes sont recalculees depuis les stats tokens et appliquent une marge de 50% par defaut.</p>
            <div className="nexus-card-grid">
              {data.estimations.map((estimate) => (
                <NexusEstimateCard estimate={estimate} key={estimate.slug} />
              ))}
            </div>
          </section>

          <section className="panel">
            <h2>Plans en base</h2>
            <p className="hint">Ces 4 plans sont la version commerciale envoyable plus tard a Flutter via `GET /api/v1/nexus-coin/plans`.</p>
            <div className="nexus-edit-grid">
              {data.plans.map((plan, index) => (
                <PlanEditor
                  key={plan.id}
                  plan={plan}
                  saving={savingId === plan.id}
                  onChange={(patch) => updatePlan(index, patch)}
                  onSave={() => savePlan(plan)}
                  onDelete={() => deletePlan(plan)}
                />
              ))}
            </div>
          </section>
        </>
      ) : null}
    </AdminShell>
  );
}

function NexusEstimateCard({ estimate }: { estimate: NexusCoinEstimate }) {
  return (
    <article className="nexus-card">
      <span className="status published">{estimate.slug}</span>
      <h3>{estimate.name}</h3>
      <p>{estimate.subtitle}</p>
      <strong>{usdMicros(estimate.priceMicros)}</strong>
      <dl>
        <dt>Nexus Coin</dt>
        <dd>{formatNumber(estimate.nexusCoins)}</dd>
        <dt>Tokens IA</dt>
        <dd>{formatNumber(estimate.tokenBudget)}</dd>
        <dt>Cout base</dt>
        <dd>{usdMicros(estimate.baseCostMicros)}</dd>
        <dt>Marge</dt>
        <dd>{estimate.marginPercent}%</dd>
        <dt>Appels estimes</dt>
        <dd>{formatNumber(estimate.estimatedCalls)}</dd>
      </dl>
    </article>
  );
}

function PlanEditor({
  plan,
  saving,
  onChange,
  onSave,
  onDelete,
}: {
  plan: NexusCoinPlan;
  saving: boolean;
  onChange: (patch: Partial<NexusCoinPlan>) => void;
  onSave: () => void;
  onDelete: () => void;
}) {
  return (
    <article className="panel nexus-editor">
      <div className="row">
        <input value={plan.name} onChange={(event) => onChange({ name: event.target.value })} placeholder="Nom" />
        <input value={plan.slug} onChange={(event) => onChange({ slug: event.target.value })} placeholder="Slug" />
        <select value={plan.status} onChange={(event) => onChange({ status: event.target.value })}>
          <option value="active">active</option>
          <option value="draft">draft</option>
          <option value="archived">archived</option>
        </select>
      </div>
      <input value={plan.subtitle} onChange={(event) => onChange({ subtitle: event.target.value })} placeholder="Sous-titre" />
      <textarea value={plan.description} onChange={(event) => onChange({ description: event.target.value })} placeholder="Texte commercial" />
      <div className="row">
        <label>
          Tokens
          <input type="number" value={plan.tokenBudget} onChange={(event) => onChange({ tokenBudget: Number(event.target.value) })} />
        </label>
        <label>
          Nexus Coin
          <input type="number" value={plan.nexusCoins} onChange={(event) => onChange({ nexusCoins: Number(event.target.value) })} />
        </label>
        <label>
          Marge %
          <input type="number" value={plan.marginPercent} onChange={(event) => onChange({ marginPercent: Number(event.target.value) })} />
        </label>
      </div>
      <dl>
        <dt>Prix actuel</dt>
        <dd>{usdMicros(plan.priceMicros)}</dd>
        <dt>Cout base</dt>
        <dd>{usdMicros(plan.baseCostMicros)}</dd>
        <dt>Appels estimes</dt>
        <dd>{formatNumber(plan.estimatedCalls)}</dd>
      </dl>
      <div className="editor-actions">
        <button className="primary" type="button" onClick={onSave} disabled={saving}>
          <Save size={16} aria-hidden />
          {saving ? "Enregistrement" : "Enregistrer"}
        </button>
        <button className="danger" type="button" onClick={onDelete} disabled={saving}>
          <Trash2 size={16} aria-hidden />
          Supprimer
        </button>
      </div>
    </article>
  );
}
