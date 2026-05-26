"use client";

import { useEffect, useMemo, useState } from "react";
import { AdminShell } from "../components/AdminShell";
import { ErrorState, LoadingState } from "../components/LoadState";
import { MetricGrid } from "../components/MetricGrid";
import { formatDate, formatNumber, loadAdminData } from "../components/api";
import type { Account, AccountsResponse } from "../types";

export default function AccountsPage() {
  const [data, setData] = useState<AccountsResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [query, setQuery] = useState("");

  useEffect(() => {
    loadAdminData<AccountsResponse>("accounts").then(setData).catch((err: Error) => setError(err.message));
  }, []);

  const accounts = useMemo(() => {
    const needle = query.trim().toLowerCase();
    if (!data || !needle) {
      return data?.accounts ?? [];
    }
    return data.accounts.filter((account) =>
      [account.email, account.pseudo, String(account.id)].some((value) => value.toLowerCase().includes(needle)),
    );
  }, [data, query]);

  return (
    <AdminShell title="Comptes actuels" description="Vue operationnelle des comptes joueurs et de leur activite applicative.">
      {error ? <ErrorState message={error} /> : null}
      {!data && !error ? <LoadingState /> : null}
      {data ? (
        <>
          <MetricGrid
            items={[
              { label: "Comptes", value: formatNumber(data.summary.totalAccounts) },
              { label: "Actifs 7 jours", value: formatNumber(data.summary.updatedLast7Days), tone: data.summary.updatedLast7Days > 0 ? "good" : "neutral" },
              { label: "Actifs 30 jours", value: formatNumber(data.summary.updatedLast30Days) },
              { label: "XP total", value: formatNumber(data.summary.totalXp) },
              { label: "Coins total", value: formatNumber(data.summary.totalCoins) },
            ]}
          />

          <section className="panel">
            <h2>Liste des comptes</h2>
            <input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Filtrer par id, pseudo ou email" />
            <div className="table-wrap">
              <table>
                <thead>
                  <tr>
                    <th>ID</th>
                    <th>Pseudo</th>
                    <th>Email</th>
                    <th>XP</th>
                    <th>Coins</th>
                    <th>Battles</th>
                    <th>RP</th>
                    <th>Profils IA</th>
                    <th>Lives</th>
                    <th>Creation</th>
                    <th>Maj</th>
                  </tr>
                </thead>
                <tbody>
                  {accounts.map((account) => (
                    <AccountRow account={account} key={account.id} />
                  ))}
                  {!accounts.length ? (
                    <tr>
                      <td colSpan={11}>Aucun compte trouve.</td>
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

function AccountRow({ account }: { account: Account }) {
  return (
    <tr>
      <td>#{account.id}</td>
      <td>{account.pseudo || "-"}</td>
      <td>{account.email || "-"}</td>
      <td>{formatNumber(account.xp)}</td>
      <td>{formatNumber(account.coin)}</td>
      <td>{formatNumber(account.battleCount)}</td>
      <td>{formatNumber(account.rolePlayCount)}</td>
      <td>{formatNumber(account.iaProfileCount)}</td>
      <td>{formatNumber(account.liveSessionCount)}</td>
      <td>{formatDate(account.createdAt)}</td>
      <td>{formatDate(account.updatedAt)}</td>
    </tr>
  );
}
