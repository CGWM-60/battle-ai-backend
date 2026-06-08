"use client";

import { useState, useEffect } from "react";
import { AdminShell } from "../../components/AdminShell";

interface Avatar {
  id: number;
  name: string;
  url: string;
  filename: string;
  created_at: string;
}

export default function NexusMmoPage() {
  const [avatars, setAvatars] = useState<Avatar[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Modal state for CRUD (create / edit / delete)
  const [modal, setModal] = useState<'create' | 'edit' | 'delete' | null>(null);
  const [currentAvatar, setCurrentAvatar] = useState<Avatar | null>(null);

  // Form states for create/edit
  const [formName, setFormName] = useState("");
  const [formFile, setFormFile] = useState<File | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const count = avatars.length;

  // Fetch all avatars
  const fetchAvatars = async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch("/api/nexus-game/assets/avatars", {
        credentials: "same-origin",
      });
      if (!res.ok) throw new Error("Failed to load avatars");
      const data = await res.json();
      setAvatars(data.avatars || []);
    } catch (e: any) {
      setError(e.message || "Erreur de chargement des avatars");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchAvatars();
  }, []);

  // Open modals
  const openCreate = () => {
    setModal('create');
    setFormName("");
    setFormFile(null);
    setCurrentAvatar(null);
  };

  const openEdit = (avatar: Avatar) => {
    setModal('edit');
    setCurrentAvatar(avatar);
    setFormName(avatar.name);
    setFormFile(null);
  };

  const openDelete = (avatar: Avatar) => {
    setModal('delete');
    setCurrentAvatar(avatar);
  };

  const closeModal = () => {
    setModal(null);
    setCurrentAvatar(null);
    setFormName("");
    setFormFile(null);
    setError(null);
  };

  // Create
  const handleCreate = async () => {
    if (!formName || !formFile) {
      setError("Nom et image sont requis");
      return;
    }
    setSubmitting(true);
    const formData = new FormData();
    formData.append("name", formName);
    formData.append("image", formFile);

    try {
      const res = await fetch("/api/nexus-game/assets/avatar", {
        method: "POST",
        body: formData,
        credentials: "same-origin",
      });
      if (!res.ok) throw new Error(await res.text());
      closeModal();
      await fetchAvatars();
    } catch (e: any) {
      setError(e.message || "Erreur lors de la création");
    } finally {
      setSubmitting(false);
    }
  };

  // Edit (name + optional new image)
  const handleEdit = async () => {
    if (!currentAvatar) return;
    setSubmitting(true);
    const formData = new FormData();
    formData.append("name", formName);
    if (formFile) formData.append("image", formFile);

    try {
      const res = await fetch(`/api/nexus-game/assets/avatars/${currentAvatar.id}`, {
        method: "PUT",
        body: formData,
        credentials: "same-origin",
      });
      if (!res.ok) throw new Error(await res.text());
      closeModal();
      await fetchAvatars();
    } catch (e: any) {
      setError(e.message || "Erreur lors de la modification");
    } finally {
      setSubmitting(false);
    }
  };

  // Delete
  const handleDelete = async () => {
    if (!currentAvatar) return;
    setSubmitting(true);
    try {
      const res = await fetch(`/api/nexus-game/assets/avatars/${currentAvatar.id}`, {
        method: "DELETE",
        credentials: "same-origin",
      });
      if (!res.ok) throw new Error(await res.text());
      closeModal();
      await fetchAvatars();
    } catch (e: any) {
      setError(e.message || "Erreur lors de la suppression");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <AdminShell
      title="Nexus MMO - Gestion des Avatars"
      description="Liste complète des avatars avec previews. Création, modification et suppression via popins (CRUD)."
    >
      {/* Summary Card - shows count, click to create */}
      <div 
        className="panel" 
        style={{ 
          marginBottom: 24, 
          cursor: 'pointer',
          border: '2px solid #7C3AED',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between'
        }}
        onClick={openCreate}
      >
        <div>
          <h2 style={{ margin: 0 }}>Création &amp; Gestion d'Avatars</h2>
          <p style={{ margin: '4px 0 0', color: '#64748b' }}>
            Cliquez pour créer un nouvel avatar
          </p>
        </div>
        <div style={{ textAlign: 'right' }}>
          <div style={{ fontSize: 42, fontWeight: 700, lineHeight: 1 }}>{count}</div>
          <div style={{ fontSize: 14, color: '#64748b' }}>avatars disponibles</div>
        </div>
      </div>

      {/* Table with all avatars + previews */}
      <section className="panel">
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
          <h2 style={{ margin: 0 }}>Tous les avatars ({count})</h2>
          <button 
            onClick={openCreate}
            style={{ 
              background: '#7C3AED', 
              color: 'white', 
              padding: '8px 16px', 
              borderRadius: 6, 
              border: 'none',
              cursor: 'pointer'
            }}
          >
            + Créer un avatar
          </button>
        </div>

        {loading ? (
          <p>Chargement des avatars...</p>
        ) : error ? (
          <p style={{ color: 'red' }}>{error}</p>
        ) : avatars.length === 0 ? (
          <p>Aucun avatar pour le moment. Utilisez le bouton ci-dessus pour en créer un.</p>
        ) : (
          <table className="data-table" style={{ width: '100%', borderCollapse: 'collapse' }}>
            <thead>
              <tr>
                <th style={{ textAlign: 'left', padding: 8 }}>Preview</th>
                <th style={{ textAlign: 'left', padding: 8 }}>Nom</th>
                <th style={{ textAlign: 'left', padding: 8 }}>URL</th>
                <th style={{ textAlign: 'left', padding: 8 }}>Créé le</th>
                <th style={{ textAlign: 'right', padding: 8 }}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {avatars.map((a) => (
                <tr key={a.id} style={{ borderTop: '1px solid #334155' }}>
                  <td style={{ padding: 8 }}>
                    <img 
                      src={a.url} 
                      alt={a.name} 
                      style={{ 
                        width: 64, 
                        height: 64, 
                        objectFit: 'cover', 
                        borderRadius: 6, 
                        border: '1px solid #1e2937' 
                      }} 
                    />
                  </td>
                  <td style={{ padding: 8, fontWeight: 500 }}>{a.name}</td>
                  <td style={{ padding: 8 }}>
                    <a 
                      href={a.url} 
                      target="_blank" 
                      rel="noreferrer"
                      style={{ color: '#a5b4fc', fontSize: 12, wordBreak: 'break-all' }}
                    >
                      {a.url.length > 55 ? a.url.substring(0, 52) + '...' : a.url}
                    </a>
                  </td>
                  <td style={{ padding: 8, fontSize: 13, color: '#64748b' }}>
                    {new Date(a.created_at).toLocaleDateString('fr-FR')}
                  </td>
                  <td style={{ padding: 8, textAlign: 'right' }}>
                    <button 
                      onClick={() => openEdit(a)}
                      style={{ marginRight: 8, padding: '4px 10px', fontSize: 12 }}
                    >
                      Modifier
                    </button>
                    <button 
                      onClick={() => openDelete(a)}
                      style={{ color: '#f87171', padding: '4px 10px', fontSize: 12 }}
                    >
                      Supprimer
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>

      {/* CRUD Popins / Modals */}
      {modal && (
        <div style={{
          position: 'fixed', top: 0, left: 0, right: 0, bottom: 0,
          background: 'rgba(0,0,0,0.75)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1000
        }}>
          <div className="panel" style={{ width: 460, maxWidth: '92%', position: 'relative', padding: 24 }}>
            <button 
              onClick={closeModal} 
              style={{ position: 'absolute', top: 12, right: 16, background: 'none', border: 'none', fontSize: 24, cursor: 'pointer', lineHeight: 1 }}
            >
              ×
            </button>

            {/* CREATE MODAL */}
            {modal === 'create' && (
              <>
                <h3>Créer un nouvel avatar</h3>
                <p style={{ fontSize: 13, color: '#64748b', marginBottom: 16 }}>
                  L'image sera automatiquement convertie en WebP côté serveur.
                </p>
                <div>
                  <label style={{ display: 'block', marginBottom: 6, fontSize: 13, fontWeight: 500 }}>Nom de l'avatar</label>
                  <input 
                    type="text" 
                    value={formName} 
                    onChange={e => setFormName(e.target.value)} 
                    placeholder="Ex: Guerrier Néon" 
                    style={{ width: '100%', padding: 10, marginBottom: 14 }} 
                  />
                  <label style={{ display: 'block', marginBottom: 6, fontSize: 13, fontWeight: 500 }}>Image (JPG / PNG)</label>
                  <input 
                    type="file" 
                    accept="image/*" 
                    onChange={e => setFormFile(e.target.files?.[0] || null)} 
                    style={{ marginBottom: 20 }} 
                  />
                  <button 
                    onClick={handleCreate} 
                    disabled={submitting || !formName || !formFile}
                    style={{ width: '100%', padding: 12, background: '#7C3AED', color: 'white', border: 'none', borderRadius: 6, fontWeight: 600 }}
                  >
                    {submitting ? 'Création en cours...' : 'Créer l\'avatar (conversion WebP)'}
                  </button>
                </div>
              </>
            )}

            {/* EDIT MODAL */}
            {modal === 'edit' && currentAvatar && (
              <>
                <h3>Modifier l'avatar #{currentAvatar.id}</h3>
                <div style={{ marginBottom: 14 }}>
                  <div style={{ marginBottom: 8, fontSize: 13, color: '#64748b' }}>Aperçu actuel</div>
                  <img src={currentAvatar.url} alt="" style={{ width: 90, height: 90, objectFit: 'cover', borderRadius: 8, border: '1px solid #334155' }} />
                </div>
                <div>
                  <label style={{ display: 'block', marginBottom: 6, fontSize: 13, fontWeight: 500 }}>Nom</label>
                  <input 
                    type="text" 
                    value={formName} 
                    onChange={e => setFormName(e.target.value)} 
                    style={{ width: '100%', padding: 10, marginBottom: 14 }} 
                  />
                  <label style={{ display: 'block', marginBottom: 6, fontSize: 13, fontWeight: 500 }}>Nouvelle image (optionnel)</label>
                  <input 
                    type="file" 
                    accept="image/*" 
                    onChange={e => setFormFile(e.target.files?.[0] || null)} 
                    style={{ marginBottom: 20 }} 
                  />
                  <button 
                    onClick={handleEdit} 
                    disabled={submitting || !formName}
                    style={{ width: '100%', padding: 12, background: '#7C3AED', color: 'white', border: 'none', borderRadius: 6, fontWeight: 600 }}
                  >
                    {submitting ? 'Mise à jour...' : 'Enregistrer les modifications'}
                  </button>
                </div>
              </>
            )}

            {/* DELETE CONFIRM POPIN */}
            {modal === 'delete' && currentAvatar && (
              <>
                <h3>Supprimer l'avatar</h3>
                <p style={{ margin: '12px 0 20px' }}>
                  Êtes-vous sûr de vouloir supprimer <strong>{currentAvatar.name}</strong> ?<br />
                  Le fichier image sera également supprimé du stockage.
                </p>
                <div style={{ display: 'flex', gap: 12 }}>
                  <button 
                    onClick={closeModal} 
                    style={{ flex: 1, padding: 11, background: '#334155', color: 'white', border: 'none', borderRadius: 6 }}
                  >
                    Annuler
                  </button>
                  <button 
                    onClick={handleDelete} 
                    disabled={submitting}
                    style={{ flex: 1, padding: 11, background: '#dc2626', color: 'white', border: 'none', borderRadius: 6, fontWeight: 600 }}
                  >
                    {submitting ? 'Suppression...' : 'Confirmer la suppression'}
                  </button>
                </div>
              </>
            )}
          </div>
        </div>
      )}

      {error && <div style={{ color: '#f87171', marginTop: 12, fontSize: 13 }}>{error}</div>}
    </AdminShell>
  );
}