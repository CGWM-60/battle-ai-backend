"use client";

import { useEffect, useState } from "react";
import { AdminShell } from "../../components/AdminShell";
import { ErrorState, LoadingState } from "../../components/LoadState";
import { formatNumber, loadAdminData } from "../../components/api";

type BuildingAsset = {
  id: number;
  buildingDefinitionId: number;
  level: number;
  variant: string;
  imageUrl: string;
  imageHash: string;
  imageSize: number;
  version: number;
  isActive: boolean;
};

export default function BuildingAssetsPage() {
  const [assets, setAssets] = useState<BuildingAsset[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [buildingId, setBuildingId] = useState("");
  const [level, setLevel] = useState("1");
  const [variant, setVariant] = useState("default");
  const [file, setFile] = useState<File | null>(null);
  const [refresh, setRefresh] = useState(0);

  useEffect(() => {
    setLoading(true);
    loadAdminData<{ items: BuildingAsset[] }>("game/building-assets")
      .then((data) => setAssets(data.items))
      .catch((err: Error) => setError(err.message))
      .finally(() => setLoading(false));
  }, [refresh]);

  async function upload() {
    if (!buildingId || !file) {
      setError("Building ID et image sont requis.");
      return;
    }
    const form = new FormData();
    form.set("level", level);
    form.set("variant", variant);
    form.set("image", file);
    const response = await fetch(`/admin/api/game/buildings/${buildingId}/assets`, {
      method: "POST",
      credentials: "same-origin",
      body: form,
    });
    if (!response.ok) {
      setError(`Upload echoue: HTTP ${response.status}`);
      return;
    }
    setFile(null);
    setRefresh((value) => value + 1);
  }

  return (
    <AdminShell title="Assets de batiments" description="Upload, hash, version et publication des images telechargees par Flutter.">
      {error ? <ErrorState message={error} /> : null}
      <section className="panel form-grid">
        <label>
          Building ID
          <input value={buildingId} onChange={(event) => setBuildingId(event.target.value)} inputMode="numeric" />
        </label>
        <label>
          Niveau
          <input value={level} onChange={(event) => setLevel(event.target.value)} inputMode="numeric" />
        </label>
        <label>
          Variante
          <input value={variant} onChange={(event) => setVariant(event.target.value)} />
        </label>
        <label>
          Image
          <input type="file" accept="image/png,image/jpeg,image/webp,image/gif" onChange={(event) => setFile(event.target.files?.[0] ?? null)} />
        </label>
        <div className="editor-actions">
          <button className="primary" type="button" onClick={upload}>
            Uploader
          </button>
          <button className="secondary" type="button" onClick={() => setRefresh((value) => value + 1)}>
            Actualiser
          </button>
        </div>
      </section>
      {loading ? <LoadingState /> : null}
      {!loading ? (
        <section className="panel">
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>ID</th>
                  <th>Building</th>
                  <th>Niveau</th>
                  <th>Variante</th>
                  <th>URL</th>
                  <th>Hash</th>
                  <th>Taille</th>
                  <th>Version</th>
                  <th>Actif</th>
                </tr>
              </thead>
              <tbody>
                {assets.map((asset) => (
                  <tr key={asset.id}>
                    <td>{asset.id}</td>
                    <td>{asset.buildingDefinitionId}</td>
                    <td>{asset.level}</td>
                    <td>{asset.variant}</td>
                    <td>{asset.imageUrl}</td>
                    <td>{asset.imageHash}</td>
                    <td>{formatNumber(asset.imageSize)}</td>
                    <td>{asset.version}</td>
                    <td>{asset.isActive ? "oui" : "non"}</td>
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
