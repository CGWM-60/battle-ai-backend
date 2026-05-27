"use client";

import { useEffect, useState } from "react";
import { AdminShell } from "../../components/AdminShell";
import { ErrorState, LoadingState } from "../../components/LoadState";
import { formatNumber, loadAdminData } from "../../components/api";

type Manifest = { version: number; buildings: { key: string; name: string; category: string; assets: unknown[] }[] };

export default function AssetsPage() {
  const [manifest, setManifest] = useState<Manifest | null>(null);
  const [error, setError] = useState<string | null>(null);
  useEffect(() => {
    loadAdminData<Manifest>("game/assets/manifest").then(setManifest).catch((err: Error) => setError(err.message));
  }, []);
  async function publish() {
    const response = await fetch("/admin/api/game/assets/manifest/publish", { method: "POST", credentials: "same-origin" });
    if (!response.ok) {
      setError(`Publication echouee: HTTP ${response.status}`);
      return;
    }
    loadAdminData<Manifest>("game/assets/manifest").then(setManifest).catch((err: Error) => setError(err.message));
  }
  return (
    <AdminShell title="Manifest assets" description="Version du catalogue et controle de publication pour Flutter.">
      {error ? <ErrorState message={error} /> : null}
      {!manifest && !error ? <LoadingState /> : null}
      {manifest ? (
        <section className="panel">
          <dl>
            <dt>Version</dt>
            <dd>{formatNumber(manifest.version)}</dd>
            <dt>Batiments actifs</dt>
            <dd>{formatNumber(manifest.buildings.length)}</dd>
            <dt>Assets actifs</dt>
            <dd>{formatNumber(manifest.buildings.reduce((total, item) => total + item.assets.length, 0))}</dd>
          </dl>
          <div className="editor-actions">
            <button className="primary" type="button" onClick={publish}>
              Publier
            </button>
          </div>
        </section>
      ) : null}
    </AdminShell>
  );
}
