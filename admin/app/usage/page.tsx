"use client";

import { useEffect, useState } from "react";
import { AdminShell } from "../components/AdminShell";
import { ErrorState, LoadingState } from "../components/LoadState";
import { MetricGrid } from "../components/MetricGrid";
import { formatDate, formatNumber, loadAdminData, usdMicros } from "../components/api";
import type { UsageData, UsageSummary } from "../types";

export default function UsagePage() {
  const [data, setData] = useState<UsageData | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadAdminData<UsageData>("usage").then(setData).catch((err: Error) => setError(err.message));
  }, []);

  return (
    <AdminShell title="IA & couts" description="Consommation tokens, estimation couts et derniers appels providers.">
      {error ? <ErrorState message={error} /> : null}
      {!data && !error ? <LoadingState /> : null}
      {data ? <UsageContent data={data} /> : null}
    </AdminShell>
  );
}

function UsageContent({ data }: { data: UsageData }) {
  return (
    <>
      <MetricGrid
        items={[
          { label: "Appels IA", value: formatNumber(data.Total.CallCount) },
          { label: "Tokens total", value: formatNumber(data.Total.TotalTokens) },
          { label: "Prompt tokens", value: formatNumber(data.Total.PromptTokens) },
          { label: "Completion tokens", value: formatNumber(data.Total.CompletionTokens) },
          { label: "Cout estime", value: usdMicros(data.Total.EstimatedCostMicros) },
        ]}
      />

      <section className="panel">
        <h2>Resume par mode</h2>
        <p className="hint">{data.PricingHint}</p>
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Mode</th>
                <th>Appels</th>
                <th>Prompt</th>
                <th>Completion</th>
                <th>Total tokens</th>
                <th>Cout estime</th>
              </tr>
            </thead>
            <tbody>
              <SummaryRow label="battle_ia" summary={data.Battle} />
              <SummaryRow label="roleplay_ia" summary={data.RolePlay} />
            </tbody>
          </table>
        </div>
      </section>

      <section className="panel">
        <h2>Derniers appels IA</h2>
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Date</th>
                <th>Mode</th>
                <th>Provider</th>
                <th>Operation</th>
                <th>Acteur</th>
                <th>Tokens</th>
                <th>Cout</th>
                <th>Source</th>
              </tr>
            </thead>
            <tbody>
              {data.Recent.map((item) => (
                <tr key={item.Id}>
                  <td>{formatDate(item.CreatedAt)}</td>
                  <td>{item.SessionMode || "-"}</td>
                  <td>{item.ProviderName} / {item.ModelName}</td>
                  <td>{item.Operation}{item.Phase ? ` (${item.Phase})` : ""}</td>
                  <td>{item.ActorName || "-"}</td>
                  <td>{formatNumber(item.TotalTokens)} ({formatNumber(item.PromptTokens)} in / {formatNumber(item.CompletionTokens)} out)</td>
                  <td>{usdMicros(item.EstimatedCostMicros)}</td>
                  <td>{item.BillingSource}{item.Estimated ? " - estime" : ""}</td>
                </tr>
              ))}
              {!data.Recent.length ? (
                <tr>
                  <td colSpan={8}>Aucun usage IA enregistre.</td>
                </tr>
              ) : null}
            </tbody>
          </table>
        </div>
      </section>
    </>
  );
}

function SummaryRow({ label, summary }: { label: string; summary: UsageSummary }) {
  return (
    <tr>
      <td>{label}</td>
      <td>{formatNumber(summary.CallCount)}</td>
      <td>{formatNumber(summary.PromptTokens)}</td>
      <td>{formatNumber(summary.CompletionTokens)}</td>
      <td>{formatNumber(summary.TotalTokens)}</td>
      <td>{usdMicros(summary.EstimatedCostMicros)}</td>
    </tr>
  );
}
