"use client";

import { useEffect, useState } from "react";
import { AdminShell } from "./components/AdminShell";
import { ErrorState, LoadingState } from "./components/LoadState";
import { MetricGrid } from "./components/MetricGrid";
import { formatDate, formatNumber, loadAdminData, usdMicros } from "./components/api";
import type { DashboardData } from "./types";

export default function DashboardPage() {
  const [data, setData] = useState<DashboardData | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadAdminData<DashboardData>("dashboard").then(setData).catch((err: Error) => setError(err.message));
  }, []);

  return (
    <AdminShell title="Vue generale" description="Etat rapide du backend, de la base, des lives et de la consommation IA.">
      {error ? <ErrorState message={error} /> : null}
      {!data && !error ? <LoadingState /> : null}
      {data ? <DashboardContent data={data} /> : null}
    </AdminShell>
  );
}

function DashboardContent({ data }: { data: DashboardData }) {
  return (
    <>
      <MetricGrid
        items={[
          { label: "Database", value: data.Health.Database, tone: data.Health.DatabaseOK ? "good" : "bad" },
          { label: "Comptes", value: formatNumber(data.Stats.Users) },
          { label: "Battles", value: formatNumber(data.Stats.Battles) },
          { label: "Quetes Battle", value: formatNumber(data.Stats.BattleQuests) },
          { label: "Quetes RP", value: formatNumber(data.Stats.RolePlayQuests) },
          { label: "Lives", value: formatNumber(data.Stats.LiveSessions) },
          { label: "Streaming", value: formatNumber(data.Stats.LiveStreaming), tone: data.Stats.LiveStreaming > 0 ? "good" : "neutral" },
          { label: "Cout IA estime", value: usdMicros(data.Usage.Total.EstimatedCostMicros) },
        ]}
      />

      <section className="split">
        <article className="panel">
          <h2>Sante backend</h2>
          <dl>
            <dt>Horodatage</dt>
            <dd>{formatDate(data.Health.Now)}</dd>
            <dt>Port</dt>
            <dd>{data.Config.AppPort}</dd>
            <dt>GIN_MODE</dt>
            <dd>{data.Config.GinMode}</dd>
            <dt>Requetes simultanees</dt>
            <dd>{data.Config.MaxConcurrentRequests}</dd>
          </dl>
        </article>

        <article className="panel">
          <h2>Cron quetes IA</h2>
          <dl>
            <dt>Etat</dt>
            <dd className={data.Cron.Enabled ? "good" : "bad"}>{data.Cron.Enabled ? "actif" : "inactif"}</dd>
            <dt>Timezone</dt>
            <dd>{data.Cron.Timezone}</dd>
            <dt>Fenetre</dt>
            <dd>{data.Cron.Window}</dd>
            <dt>Prochain run</dt>
            <dd>{data.Cron.NextRun || "-"}</dd>
          </dl>
        </article>
      </section>

      <section className="triple">
        <RecentList title="Dernieres quetes Battle" items={data.Recent.BattleQuests.map((item) => `#${item.Id} ${item.Title} - ${item.Status}`)} />
        <RecentList title="Dernieres quetes RP" items={data.Recent.RolePlayQuests.map((item) => `#${item.Id} ${item.Title} - ${item.Status}`)} />
        <RecentList title="Dernieres battles" items={data.Recent.Battles.map((item) => `#${item.Id} ${item.Title} - ${item.Status}`)} />
      </section>
    </>
  );
}

function RecentList({ title, items }: { title: string; items: string[] }) {
  return (
    <article className="panel">
      <h2>{title}</h2>
      {items.length ? (
        <ul>
          {items.map((item) => (
            <li key={item}>{item}</li>
          ))}
        </ul>
      ) : (
        <p className="hint">Aucune donnee.</p>
      )}
    </article>
  );
}
