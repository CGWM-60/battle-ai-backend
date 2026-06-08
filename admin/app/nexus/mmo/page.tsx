"use client";

import { useState } from "react";
import { AdminShell } from "../../../components/AdminShell";

export default function NexusMmoPage() {
  const [avatarName, setAvatarName] = useState("");
  const [avatarFile, setAvatarFile] = useState<File | null>(null);
  const [uploading, setUploading] = useState(false);
  const [result, setResult] = useState<any>(null);
  const [error, setError] = useState<string | null>(null);

  // Fake stats for now (Etape 2)
  const fakeStats = {
    players: 1247,
    activeCities: 892,
    totalAvatars: 1156,
    online: 342,
  };

  const handleAvatarUpload = async () => {
    if (!avatarName || !avatarFile) {
      setError("Nom et image requis");
      return;
    }
    setUploading(true);
    setError(null);
    setResult(null);

    const formData = new FormData();
    formData.append("name", avatarName);
    formData.append("image", avatarFile);

    try {
      // Call the backend directly (in real setup via /admin/api proxy if configured)
      // For now using the /api/nexus-game path
      const res = await fetch("/api/nexus-game/assets/avatar", {
        method: "POST",
        body: formData,
        credentials: "same-origin",
      });

      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || `HTTP ${res.status}`);
      }

      const data = await res.json();
      setResult(data);
      setAvatarName("");
      setAvatarFile(null);
    } catch (e: any) {
      setError(e.message || "Erreur upload");
    } finally {
      setUploading(false);
    }
  };

  return (
    <AdminShell
      title="Nexus MMO"
      description="Gestion du module MMO (stats fake + cartes de gestion). Etape 2 : création d'avatar avec conversion WebP."
    >
      {/* Fake Stats */}
      <section className="panel" style={{ marginBottom: 24 }}>
        <h2>Stats (fake pour l'instant)</h2>
        <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(140px, 1fr))", gap: 16 }}>
          <div className="stat-card">
            <div className="label">Joueurs</div>
            <div className="value">{fakeStats.players}</div>
          </div>
          <div className="stat-card">
            <div className="label">Villes actives</div>
            <div className="value">{fakeStats.activeCities}</div>
          </div>
          <div className="stat-card">
            <div className="label">Avatars créés</div>
            <div className="value">{fakeStats.totalAvatars}</div>
          </div>
          <div className="stat-card">
            <div className="label">En ligne</div>
            <div className="value">{fakeStats.online}</div>
          </div>
        </div>
      </section>

      {/* Management Cards */}
      <section className="panel" style={{ marginBottom: 24 }}>
        <h2>Cartes de gestion</h2>
        <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(320px, 1fr))", gap: 16 }}>

          {/* Avatar Creation Card - First one */}
          <div className="card" style={{ border: "1px solid #334155", padding: 16, borderRadius: 8 }}>
            <h3>Création d'Avatar</h3>
            <p style={{ fontSize: 13, color: "#64748b" }}>
              Nom + image → conversion WebP obligatoire → stockage persistant (volume) → enregistré en base.
            </p>

            <div style={{ marginTop: 12 }}>
              <input
                type="text"
                placeholder="Nom de l'avatar"
                value={avatarName}
                onChange={(e) => setAvatarName(e.target.value)}
                style={{ width: "100%", padding: 8, marginBottom: 8 }}
              />
              <input
                type="file"
                accept="image/*"
                onChange={(e) => setAvatarFile(e.target.files?.[0] || null)}
                style={{ marginBottom: 8 }}
              />
              <button
                onClick={handleAvatarUpload}
                disabled={uploading || !avatarName || !avatarFile}
                style={{ padding: "8px 16px" }}
              >
                {uploading ? "Upload en cours..." : "Créer Avatar (WebP)"}
              </button>
            </div>

            {error && <div style={{ color: "red", marginTop: 8 }}>{error}</div>}
            {result && (
              <div style={{ marginTop: 12, padding: 8, background: "#0f172a", borderRadius: 4 }}>
                <strong>Avatar créé !</strong>
                <pre style={{ fontSize: 12, whiteSpace: "pre-wrap" }}>
                  {JSON.stringify(result, null, 2)}
                </pre>
              </div>
            )}
          </div>

          {/* Other fake management cards */}
          <div className="card" style={{ border: "1px solid #334155", padding: 16, borderRadius: 8, opacity: 0.7 }}>
            <h3>Gestion des Quêtes (fake)</h3>
            <p>Liste des quêtes monde / perso. Boutons "Générer", "Clôturer".</p>
            <button disabled>TODO</button>
          </div>

          <div className="card" style={{ border: "1px solid #334155", padding: 16, borderRadius: 8, opacity: 0.7 }}>
            <h3>Gestion des Agents IA (fake)</h3>
            <p>Assignation, budget tokens, plans IA.</p>
            <button disabled>TODO</button>
          </div>
        </div>
      </section>

      <p style={{ fontSize: 12, color: "#64748b" }}>
        Note: L'upload envoie vers /api/nexus-game/assets/avatar. L'image est convertie en WebP côté serveur.
        Le dossier est monté via volume (compose.prod.yml) pour persistance.
      </p>
    </AdminShell>
  );
}