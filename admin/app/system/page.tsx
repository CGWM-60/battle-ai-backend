"use client";

import { useEffect, useState } from "react";
import { AdminShell } from "../components/AdminShell";
import { ErrorState, LoadingState } from "../components/LoadState";
import { MetricGrid } from "../components/MetricGrid";
import { bytes, formatDate, formatNumber, loadAdminData } from "../components/api";
import type { SystemResponse } from "../types";

export default function SystemPage() {
  const [data, setData] = useState<SystemResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadAdminData<SystemResponse>("system").then(setData).catch((err: Error) => setError(err.message));
  }, []);

  return (
    <AdminShell title="Charge systeme" description="Statistiques de connexion, requetes HTTP, base de donnees et runtime Go.">
      {error ? <ErrorState message={error} /> : null}
      {!data && !error ? <LoadingState /> : null}
      {data ? <SystemContent data={data} /> : null}
    </AdminShell>
  );
}

function SystemContent({ data }: { data: SystemResponse }) {
  return (
    <>
      <MetricGrid
        items={[
          { label: "Database", value: data.health.Database, tone: data.health.DatabaseOK ? "good" : "bad" },
          { label: "Requetes actives", value: formatNumber(data.requests.activeRequests), tone: data.requests.activeRequests > 0 ? "good" : "neutral" },
          { label: "Requetes total", value: formatNumber(data.requests.totalRequests) },
          { label: "Latence moyenne", value: `${data.requests.averageLatencyMs.toFixed(1)} ms` },
          { label: "Lives streaming", value: formatNumber(data.network.liveStreaming) },
          { label: "Viewers live", value: formatNumber(data.network.liveViewers) },
          { label: "Connexions DB", value: `${data.database.openConnections}/${data.database.maxOpenConnections || "-"}` },
          { label: "Goroutines", value: formatNumber(data.runtime.numGoroutine) },
        ]}
      />

      <section className="split">
        <article className="panel">
          <h2>HTTP</h2>
          <dl>
            <dt>2xx</dt>
            <dd>{formatNumber(data.requests.status2xx)}</dd>
            <dt>3xx</dt>
            <dd>{formatNumber(data.requests.status3xx)}</dd>
            <dt>4xx</dt>
            <dd>{formatNumber(data.requests.status4xx)}</dd>
            <dt>5xx</dt>
            <dd>{formatNumber(data.requests.status5xx)}</dd>
            <dt>Latence max</dt>
            <dd>{data.requests.maxLatencyMs.toFixed(1)} ms</dd>
          </dl>
        </article>

        <article className="panel">
          <h2>Reseau applicatif</h2>
          <dl>
            <dt>Live sessions</dt>
            <dd>{formatNumber(data.network.liveSessions)}</dd>
            <dt>Streaming</dt>
            <dd>{formatNumber(data.network.liveStreaming)}</dd>
            <dt>Ended</dt>
            <dd>{formatNumber(data.network.liveEnded)}</dd>
            <dt>Arenes</dt>
            <dd>{formatNumber(data.network.arenas)}</dd>
            <dt>Coop parties</dt>
            <dd>{formatNumber(data.network.coopParties)}</dd>
          </dl>
        </article>
      </section>

      <section className="split">
        <article className="panel">
          <h2>Base de donnees</h2>
          <dl>
            <dt>Open</dt>
            <dd>{formatNumber(data.database.openConnections)}</dd>
            <dt>In use</dt>
            <dd>{formatNumber(data.database.inUse)}</dd>
            <dt>Idle</dt>
            <dd>{formatNumber(data.database.idle)}</dd>
            <dt>Wait count</dt>
            <dd>{formatNumber(data.database.waitCount)}</dd>
            <dt>Max idle closed</dt>
            <dd>{formatNumber(data.database.maxIdleClosed)}</dd>
            <dt>Max lifetime closed</dt>
            <dd>{formatNumber(data.database.maxLifetimeClosed)}</dd>
          </dl>
        </article>

        <article className="panel">
          <h2>Runtime Go</h2>
          <dl>
            <dt>Demarrage</dt>
            <dd>{formatDate(data.runtime.startedAt)}</dd>
            <dt>Uptime</dt>
            <dd>{formatNumber(data.runtime.uptimeSeconds)} s</dd>
            <dt>Version</dt>
            <dd>{data.runtime.goVersion}</dd>
            <dt>Plateforme</dt>
            <dd>{data.runtime.goos}/{data.runtime.goarch}</dd>
            <dt>CPU</dt>
            <dd>{formatNumber(data.runtime.numCpu)}</dd>
            <dt>Memoire alloc</dt>
            <dd>{bytes(data.runtime.allocBytes)}</dd>
            <dt>Heap</dt>
            <dd>{bytes(data.runtime.heapAllocBytes)}</dd>
            <dt>GC</dt>
            <dd>{formatNumber(data.runtime.numGc)}</dd>
          </dl>
        </article>
      </section>
    </>
  );
}
