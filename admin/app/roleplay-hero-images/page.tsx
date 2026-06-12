"use client";

import { useEffect, useMemo, useState } from "react";
import { AdminShell } from "../components/AdminShell";
import { formatDate, formatNumber, loadAdminData } from "../components/api";
import type { AdminRolePlayHeroImage, AdminRolePlayHeroImagesResponse } from "../types";

type SexFilter = "all" | "h" | "f";

export default function RolePlayHeroImagesPage() {
  const [data, setData] = useState<AdminRolePlayHeroImagesResponse | null>(null);
  const [filter, setFilter] = useState<SexFilter>("all");
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);

  async function reload() {
    setError("");
    try {
      const payload = await loadAdminData<AdminRolePlayHeroImagesResponse>("roleplay-hero-images");
      setData(payload);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Chargement impossible");
    }
  }

  useEffect(() => {
    reload();
  }, []);

  const images = useMemo(() => {
    const list = data?.images ?? [];
    if (filter === "all") {
      return list;
    }
    return list.filter((image) => image.sex === filter);
  }, [data, filter]);

  async function createImage(formData: FormData) {
    setSaving(true);
    setError("");
    try {
      const response = await fetch("/admin/api/roleplay-hero-images", {
        method: "POST",
        credentials: "same-origin",
        body: formData,
      });
      if (!response.ok) {
        const payload = await response.json().catch(() => ({}));
        throw new Error(payload.error ?? `HTTP ${response.status}`);
      }
      await reload();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Creation impossible");
    } finally {
      setSaving(false);
    }
  }

  async function patchImage(image: AdminRolePlayHeroImage, patch: Partial<AdminRolePlayHeroImage>) {
    setError("");
    try {
      const response = await fetch(`/admin/api/roleplay-hero-images/${image.id}`, {
        method: "PATCH",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(patch),
      });
      if (!response.ok) {
        const payload = await response.json().catch(() => ({}));
        throw new Error(payload.error ?? `HTTP ${response.status}`);
      }
      await reload();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Mise a jour impossible");
    }
  }

  async function deleteImage(image: AdminRolePlayHeroImage) {
    if (!window.confirm(`Supprimer l'image "${image.name}" ?`)) {
      return;
    }
    setError("");
    try {
      const response = await fetch(`/admin/api/roleplay-hero-images/${image.id}`, {
        method: "DELETE",
        credentials: "same-origin",
      });
      if (!response.ok) {
        const payload = await response.json().catch(() => ({}));
        throw new Error(payload.error ?? `HTTP ${response.status}`);
      }
      await reload();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Suppression impossible");
    }
  }

  return (
    <AdminShell title="Images heros RP" description="Catalogue d'avatars selectionnables dans le builder de heros.">
      {error ? <div className="alert error">{error}</div> : null}

      <section className="panel hero-image-create-panel">
        <h2>Ajouter une image</h2>
        <HeroImageForm saving={saving} onSubmit={createImage} />
      </section>

      <section className="rp-toolbar">
        <div>
          <strong>{formatNumber(images.length)} images affichees</strong>
          <span>{formatNumber(data?.images.length ?? 0)} images au total</span>
        </div>
        <div className="hero-image-filter">
          {(["all", "h", "f"] as SexFilter[]).map((value) => (
            <button className={filter === value ? "active" : ""} type="button" key={value} onClick={() => setFilter(value)}>
              {value === "all" ? "Tous" : value.toUpperCase()}
            </button>
          ))}
        </div>
      </section>

      <section className="hero-image-grid">
        {images.map((image) => (
          <article className="panel hero-image-card" key={image.id}>
            <img src={image.imageUrl} alt={image.name} />
            <div className="hero-image-card-body">
              <div>
                <span className={`status ${image.isActive ? "published" : "archived"}`}>
                  {image.isActive ? "active" : "inactive"}
                </span>
                <h3>{image.name}</h3>
                <p>{image.sex.toUpperCase()} / v{image.version} / {formatNumber(image.imageSize)} o</p>
                <small>{formatDate(image.createdAt)}</small>
              </div>
              <div className="hero-image-card-actions">
                <button type="button" onClick={() => patchImage(image, { isActive: !image.isActive })}>
                  {image.isActive ? "Desactiver" : "Activer"}
                </button>
                <button className="danger" type="button" onClick={() => deleteImage(image)}>
                  Supprimer
                </button>
              </div>
            </div>
          </article>
        ))}
        {!images.length ? <div className="panel empty-panel">Aucune image hero.</div> : null}
      </section>
    </AdminShell>
  );
}

function HeroImageForm({
  saving,
  onSubmit,
}: {
  saving: boolean;
  onSubmit: (formData: FormData) => Promise<void>;
}) {
  return (
    <form
      className="hero-image-form"
      onSubmit={async (event) => {
        event.preventDefault();
        await onSubmit(new FormData(event.currentTarget));
        event.currentTarget.reset();
      }}
    >
      <label>
        Nom
        <input required name="name" minLength={2} placeholder="Ex: Mira Vale" />
      </label>
      <label>
        Sexe
        <select required name="sex" defaultValue="h">
          <option value="h">H</option>
          <option value="f">F</option>
        </select>
      </label>
      <label>
        Image
        <input required name="image" type="file" accept="image/png,image/jpeg,image/webp,image/gif" />
      </label>
      <button className="primary" type="submit" disabled={saving}>
        {saving ? "Upload..." : "Creer"}
      </button>
    </form>
  );
}
