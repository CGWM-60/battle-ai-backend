"use client";

import { useEffect, useState } from "react";
import { AdminShell } from "../../../components/AdminShell";
import { ErrorState, LoadingState } from "../../../components/LoadState";
import { loadAdminData } from "../../../components/api";

type ProviderStatus = { name: string; displayName: string; configured: boolean; primary: boolean; fallback: boolean; keyEnv: string; modelEnv: string; model: string; secretPreview: string };

export default function AIProvidersPage() {
  const [providers, setProviders] = useState<ProviderStatus[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  useEffect(() => {
    loadAdminData<{ providers: ProviderStatus[] }>("game/ai/providers").then((data) => setProviders(data.providers)).catch((err: Error) => setError(err.message));
  }, []);
  return (
    <AdminShell title="Providers IA" description="Etat de configuration du provider principal et du fallback. Les secrets restent masques.">
      {error ? <ErrorState message={error} /> : null}
      {!providers && !error ? <LoadingState /> : null}
      {providers ? (
        <section className="panel">
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Provider</th>
                  <th>Etat</th>
                  <th>Role</th>
                  <th>Modele</th>
                  <th>Env</th>
                  <th>Cle</th>
                </tr>
              </thead>
              <tbody>
                {providers.map((provider) => (
                  <tr key={provider.name}>
                    <td>{provider.displayName}</td>
                    <td className={provider.configured ? "good" : "bad"}>{provider.configured ? "configure" : "incomplet"}</td>
                    <td>{provider.primary ? "principal" : provider.fallback ? "fallback" : "-"}</td>
                    <td>{provider.model || "-"}</td>
                    <td>{provider.keyEnv} / {provider.modelEnv}</td>
                    <td>{provider.secretPreview || "-"}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      ) : null}
    </AdminShell>
  );
}
