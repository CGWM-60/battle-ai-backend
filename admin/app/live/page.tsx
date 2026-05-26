"use client";

import { Square } from "lucide-react";
import { useEffect, useState } from "react";
import { AdminShell } from "../components/AdminShell";
import { ErrorState, LoadingState } from "../components/LoadState";
import { MetricGrid } from "../components/MetricGrid";
import { formatDate, formatNumber, loadAdminData } from "../components/api";
import type { LiveSession, StatsData } from "../types";

type LiveResponse = {
  stats: StatsData;
  liveSessions: LiveSession[];
};

export default function LivePage() {
  const [data, setData] = useState<LiveResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadAdminData<LiveResponse>("live").then(setData).catch((err: Error) => setError(err.message));
  }, []);

  return (
    <AdminShell title="Live" description="Sessions live, viewers et actions de cloture des streams actifs.">
      {error ? <ErrorState message={error} /> : null}
      {!data && !error ? <LoadingState /> : null}
      {data ? (
        <>
          <MetricGrid
            items={[
              { label: "Sessions", value: formatNumber(data.stats.LiveSessions) },
              { label: "Streaming", value: formatNumber(data.stats.LiveStreaming), tone: data.stats.LiveStreaming > 0 ? "good" : "neutral" },
              { label: "Ended", value: formatNumber(data.stats.LiveEnded) },
            ]}
          />
          <section className="panel">
            <h2>Sessions recentes</h2>
            <div className="table-wrap">
              <table>
                <thead>
                  <tr>
                    <th>ID</th>
                    <th>Channel</th>
                    <th>Mode</th>
                    <th>Status</th>
                    <th>Viewers</th>
                    <th>Replay</th>
                    <th>Maj</th>
                    <th>Action</th>
                  </tr>
                </thead>
                <tbody>
                  {data.liveSessions.map((session) => (
                    <tr key={session.Id}>
                      <td>#{session.Id}</td>
                      <td>{session.ChannelKey}</td>
                      <td>{session.Mode}</td>
                      <td><span className={`status ${session.Status}`}>{session.Status}</span></td>
                      <td>{formatNumber(session.ViewerCount)}</td>
                      <td>{session.AllowReplay ? "oui" : "non"}</td>
                      <td>{formatDate(session.UpdatedAt)}</td>
                      <td>
                        {session.Status !== "ended" ? (
                          <form method="post" action={`/admin/live/${session.Id}/end`}>
                            <button className="danger" type="submit" title="Terminer le live">
                              <Square size={14} aria-hidden />
                              End
                            </button>
                          </form>
                        ) : (
                          "-"
                        )}
                      </td>
                    </tr>
                  ))}
                  {!data.liveSessions.length ? (
                    <tr>
                      <td colSpan={8}>Aucun live.</td>
                    </tr>
                  ) : null}
                </tbody>
              </table>
            </div>
          </section>
        </>
      ) : null}
    </AdminShell>
  );
}
