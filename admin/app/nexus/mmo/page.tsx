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
  const [factions, setFactions] = useState<any[]>([]);
  const [companions, setCompanions] = useState<any[]>([]);
  const [worlds, setWorlds] = useState<any[]>([]);
  const [prompts, setPrompts] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [activeView, setActiveView] = useState<'overview' | 'avatars' | 'factions' | 'companions' | 'worlds' | 'prompts' | 'stats' | 'ia'>('overview');

  // Modal state for CRUD (create / edit / delete) - per type for simplicity
  const [modal, setModal] = useState<'create' | 'edit' | 'delete' | null>(null);
  const [modalType, setModalType] = useState<'avatars' | 'factions' | 'companions' | null>(null);
  const [currentItem, setCurrentItem] = useState<any>(null);

  // For Prompts real popin
  const [promptModal, setPromptModal] = useState(false);
  const [promptForm, setPromptForm] = useState({ prompt_id: '', version: '', domain: '', purpose: '', system_prompt: '' });
  const [editingPrompt, setEditingPrompt] = useState<any>(null);

  // Form states
  const [formName, setFormName] = useState("");
  const [formFile, setFormFile] = useState<File | null>(null);
  const [formDesc, setFormDesc] = useState("");
  const [formColor, setFormColor] = useState("#FF0000");
  const [formRole, setFormRole] = useState("Gouverneur");
  const [formLevel, setFormLevel] = useState(1);
  const [submitting, setSubmitting] = useState(false);

  const avatarCount = avatars.length;
  const factionCount = factions.length;
  const companionCount = companions.length;
  const worldCount = worlds.length;
  const promptCount = prompts.length;

  const fetchAll = async () => {
    setLoading(true);
    setError(null);
    try {
      const [avRes, facRes, comRes, worldRes, promptRes] = await Promise.all([
        fetch("/api/nexus-game/assets/avatars", { credentials: "same-origin" }),
        fetch("/api/nexus-game/factions", { credentials: "same-origin" }),
        fetch("/api/nexus-game/ia-companions", { credentials: "same-origin" }),
        fetch("/api/nexus-game/worlds", { credentials: "same-origin" }),
        fetch("/api/nexus-game/prompts", { credentials: "same-origin" }),
      ]);

      if (avRes.ok) {
        const avData = await avRes.json();
        setAvatars(avData.avatars || []);
      }
      if (facRes.ok) {
        const facData = await facRes.json();
        setFactions(facData.factions || []);
      }
      if (comRes.ok) {
        const comData = await comRes.json();
        setCompanions(comData.ia_companions || []);
      }
      if (worldRes.ok) {
        const wData = await worldRes.json();
        setWorlds(wData.worlds || []);
      }
      if (promptRes.ok) {
        const pData = await promptRes.json();
        setPrompts(pData.prompts || []);
      }
    } catch (e: any) {
      setError(e.message || "Erreur de chargement");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchAll();
  }, []);

  // Open modals (updated for new state)
  const openCreate = (type: 'avatars' | 'factions' | 'companions' = 'avatars') => {
    setModalType(type);
    setModal('create');
    setFormName("");
    setFormFile(null);
    setCurrentItem(null);
    if (type === 'factions') { setFormDesc(''); setFormColor('#FF0000'); }
    if (type === 'companions') { setFormRole('Gouverneur'); setFormLevel(1); }
  };

  const openEdit = (item: any, type: 'avatars' | 'factions' | 'companions') => {
    setModalType(type);
    setModal('edit');
    setCurrentItem(item);
    setFormName(item.name || '');
    setFormFile(null);
    if (type === 'factions') { setFormDesc(item.description || ''); setFormColor(item.color || '#FF0000'); }
    if (type === 'companions') { setFormRole(item.role || 'Gouverneur'); setFormLevel(item.level || 1); }
  };

  const openDelete = (item: any, type: 'avatars' | 'factions' | 'companions') => {
    setModalType(type);
    setModal('delete');
    setCurrentItem(item);
  };

  const closeModal = () => {
    setModal(null);
    setModalType(null);
    setCurrentItem(null);
    setFormName("");
    setFormFile(null);
    setFormDesc("");
    setFormColor("#FF0000");
    setFormRole("Gouverneur");
    setFormLevel(1);
    setError(null);
  };

  // Prompt popin handlers
  const openPromptModal = (p: any = null) => {
    if (p) {
      setPromptForm({
        prompt_id: p.prompt_id || '',
        version: p.version || '',
        domain: p.domain || '',
        purpose: p.purpose || '',
        system_prompt: p.system_prompt || '',
      });
      setEditingPrompt(p);
    } else {
      setPromptForm({ prompt_id: '', version: '', domain: '', purpose: '', system_prompt: '' });
      setEditingPrompt(null);
    }
    setPromptModal(true);
  };

  const closePromptModal = () => {
    setPromptModal(false);
    setPromptForm({ prompt_id: '', version: '', domain: '', purpose: '', system_prompt: '' });
    setEditingPrompt(null);
  };

  const submitPrompt = async () => {
    setSubmitting(true);
    try {
      if (editingPrompt) {
        await fetch(`/api/nexus-game/prompts/${editingPrompt.id}`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ system_prompt: promptForm.system_prompt }),
          credentials: 'same-origin',
        });
      } else {
        await fetch('/api/nexus-game/prompts', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(promptForm),
          credentials: 'same-origin',
        });
      }
      closePromptModal();
      await fetchAll();
    } catch (e: any) {
      setError(e.message || 'Erreur prompt');
    } finally {
      setSubmitting(false);
    }
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
      await fetchAll();
    } catch (e: any) {
      setError(e.message || "Erreur lors de la création");
    } finally {
      setSubmitting(false);
    }
  };

  // Old avatar-specific handleEdit/handleDelete removed (conflicted with new state).
  // Avatar CRUD is handled via the generic modals when modalType==='avatars' (port the previous full modals here if needed for full avatar functionality).

  const goToView = (view: 'avatars' | 'factions' | 'companions' | 'worlds' | 'prompts' | 'stats' | 'ia') => {
    setActiveView(view);
    // scroll to content if needed
    window.scrollTo({ top: 400, behavior: 'smooth' });
  };

  const backToOverview = () => {
    setActiveView('overview');
    closeModal();
  };

  // Reusable simple modal wrapper
  const renderModal = (title: string, children: React.ReactNode) => (
    <div style={{
      position: 'fixed', top: 0, left: 0, right: 0, bottom: 0,
      background: 'rgba(0,0,0,0.75)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1000
    }}>
      <div className="panel" style={{ width: 480, maxWidth: '92%', position: 'relative', padding: 24 }}>
        <button 
          onClick={closeModal} 
          style={{ position: 'absolute', top: 12, right: 16, background: 'none', border: 'none', fontSize: 24, cursor: 'pointer', lineHeight: 1 }}
        >
          ×
        </button>
        <h3>{title}</h3>
        {children}
      </div>
    </div>
  );

  return (
    <AdminShell
      title="Nexus MMO"
      description="Statistiques fake et points d'entrée vers la gestion des Avatars, Factions et IA Compagnons."
    >
      {/* Overview - Fake Stats + 3 Entry Points */}
      {activeView === 'overview' && (
        <>
          {/* Fake Stats */}
          <section className="panel" style={{ marginBottom: 24 }}>
            <h2>Statistiques (fake pour l'instant)</h2>
            <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(140px, 1fr))", gap: 16 }}>
              <div className="stat-card">
                <div className="label">Joueurs actifs</div>
                <div className="value">1247</div>
              </div>
              <div className="stat-card">
                <div className="label">Villes</div>
                <div className="value">892</div>
              </div>
              <div className="stat-card">
                <div className="label">En ligne</div>
                <div className="value">342</div>
              </div>
            </div>
          </section>

          {/* 3 Entry Points Cards */}
          <section className="panel">
            <h2>Points d'entrée</h2>
            <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(280px, 1fr))", gap: 16 }}>

              {/* Avatar Card */}
              <div 
                className="card" 
                style={{ border: '1px solid #7C3AED', padding: 16, borderRadius: 8, cursor: 'pointer' }}
                onClick={() => goToView('avatars')}
              >
                <h3>Avatars</h3>
                <div style={{ fontSize: 32, fontWeight: 700, color: '#7C3AED' }}>{avatarCount}</div>
                <p style={{ fontSize: 13, color: '#64748b' }}>Gestion des avatars des joueurs (nom + image WebP)</p>
                <button style={{ marginTop: 8, width: '100%' }}>Gérer les Avatars →</button>
              </div>

              {/* Faction Card */}
              <div 
                className="card" 
                style={{ border: '1px solid #10b981', padding: 16, borderRadius: 8, cursor: 'pointer' }}
                onClick={() => goToView('factions')}
              >
                <h3>Factions</h3>
                <div style={{ fontSize: 32, fontWeight: 700, color: '#10b981' }}>{factionCount}</div>
                <p style={{ fontSize: 13, color: '#64748b' }}>Factions du monde et réputation des joueurs</p>
                <button style={{ marginTop: 8, width: '100%' }}>Gérer les Factions →</button>
              </div>

              {/* IA Compagnons Card */}
              <div 
                className="card" 
                style={{ border: '1px solid #f59e0b', padding: 16, borderRadius: 8, cursor: 'pointer' }}
                onClick={() => goToView('companions')}
              >
                <h3>IA Compagnons</h3>
                <div style={{ fontSize: 32, fontWeight: 700, color: '#f59e0b' }}>{companionCount}</div>
                <p style={{ fontSize: 13, color: '#64748b' }}>Compagnons IA (Gouverneur, Stratège...)</p>
                <button style={{ marginTop: 8, width: '100%' }}>Gérer les IA Compagnons →</button>
              </div>

              {/* Worlds Card */}
              <div 
                className="card" 
                style={{ border: '1px solid #3b82f6', padding: 16, borderRadius: 8, cursor: 'pointer' }}
                onClick={() => goToView('worlds')}
              >
                <h3>Mondes & Continents</h3>
                <div style={{ fontSize: 32, fontWeight: 700, color: '#3b82f6' }}>{worldCount}</div>
                <p style={{ fontSize: 13, color: '#64748b' }}>Gestion des mondes (5 continents, 500 joueurs max, 3 factions max par continent, IA events)</p>
                <button style={{ marginTop: 8, width: '100%' }}>Gérer les Mondes →</button>
              </div>

              {/* Prompts Card */}
              <div 
                className="card" 
                style={{ border: '1px solid #8b5cf6', padding: 16, borderRadius: 8, cursor: 'pointer' }}
                onClick={() => goToView('prompts')}
              >
                <h3>Prompts IA Serveur</h3>
                <div style={{ fontSize: 32, fontWeight: 700, color: '#8b5cf6' }}>{promptCount}</div>
                <p style={{ fontSize: 13, color: '#64748b' }}>CRUD prompts pour IA serveur (versionnés, optimisés, modifiables)</p>
                <button style={{ marginTop: 8, width: '100%' }}>Gérer les Prompts →</button>
              </div>

              {/* Stats Card */}
              <div 
                className="card" 
                style={{ border: '1px solid #ef4444', padding: 16, borderRadius: 8, cursor: 'pointer' }}
                onClick={() => goToView('stats')}
              >
                <h3>Stats & Visualisation</h3>
                <div style={{ fontSize: 32, fontWeight: 700, color: '#ef4444' }}>📊</div>
                <p style={{ fontSize: 13, color: '#64748b' }}>Distribution joueurs par continent, capacité, full worlds, IA logs</p>
                <button style={{ marginTop: 8, width: '100%' }}>Voir les Stats →</button>
              </div>

              {/* IA Tools Card */}
              <div 
                className="card" 
                style={{ border: '1px solid #14b8a6', padding: 16, borderRadius: 8, cursor: 'pointer' }}
                onClick={() => goToView('ia')}
              >
                <h3>Outils IA Serveur</h3>
                <div style={{ fontSize: 32, fontWeight: 700, color: '#14b8a6' }}>🤖</div>
                <p style={{ fontSize: 13, color: '#64748b' }}>Déclencher génération events, ticks, prompts live</p>
                <button style={{ marginTop: 8, width: '100%' }}>Outils IA →</button>
              </div>

            </div>
          </section>
        </>
      )}

      {/* Avatars View */}
      {activeView === 'avatars' && (
        <section className="panel">
          <button onClick={backToOverview} style={{ marginBottom: 16 }}>← Retour aux points d'entrée</button>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
            <h2>Tous les avatars ({avatarCount})</h2>
            <button onClick={() => openCreate('avatars')} style={{ background: '#7C3AED', color: 'white', padding: '8px 16px', borderRadius: 6, border: 'none' }}>
              + Créer un avatar
            </button>
          </div>

          {loading ? <p>Chargement...</p> : error ? <p style={{color:'red'}}>{error}</p> : (
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
                      <img src={a.url} alt={a.name} style={{ width: 64, height: 64, objectFit: 'cover', borderRadius: 6, border: '1px solid #1e2937' }} />
                    </td>
                    <td style={{ padding: 8, fontWeight: 500 }}>{a.name}</td>
                    <td style={{ padding: 8 }}>
                      <a href={a.url} target="_blank" rel="noreferrer" style={{ color: '#a5b4fc', fontSize: 12 }}>{a.url.length > 50 ? a.url.substring(0,47)+'...' : a.url}</a>
                    </td>
                    <td style={{ padding: 8, fontSize: 13, color: '#64748b' }}>{new Date(a.created_at).toLocaleDateString('fr-FR')}</td>
                    <td style={{ padding: 8, textAlign: 'right' }}>
                      <button onClick={() => openEdit(a, 'avatars')} style={{ marginRight: 8, padding: '4px 10px', fontSize: 12 }}>Modifier</button>
                      <button onClick={() => openDelete(a, 'avatars')} style={{ color: '#f87171', padding: '4px 10px', fontSize: 12 }}>Supprimer</button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </section>
      )}

      {/* Factions View - same principle */}
      {activeView === 'factions' && (
        <section className="panel">
          <button onClick={backToOverview} style={{ marginBottom: 16 }}>← Retour aux points d'entrée</button>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
            <h2>Toutes les factions ({factionCount})</h2>
            <button onClick={() => { setModalType('factions'); setModal('create'); setFormName(''); setFormDesc(''); setFormColor('#FF0000'); }} style={{ background: '#10b981', color: 'white', padding: '8px 16px', borderRadius: 6, border: 'none' }}>
              + Créer une faction
            </button>
          </div>

          <table className="data-table" style={{ width: '100%', borderCollapse: 'collapse' }}>
            <thead>
              <tr>
                <th style={{ textAlign: 'left', padding: 8 }}>Preview</th>
                <th style={{ textAlign: 'left', padding: 8 }}>Nom</th>
                <th style={{ textAlign: 'left', padding: 8 }}>Description</th>
                <th style={{ textAlign: 'left', padding: 8 }}>Couleur</th>
                <th style={{ textAlign: 'right', padding: 8 }}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {factions.map((f: any) => (
                <tr key={f.id} style={{ borderTop: '1px solid #334155' }}>
                  <td style={{ padding: 8 }}>
                    {f.url ? (
                      <img src={f.url} alt={f.name} style={{ width: 64, height: 64, objectFit: 'cover', borderRadius: 6, border: '1px solid #1e2937' }} />
                    ) : (
                      <span style={{ color: '#64748b', fontSize: 12 }}>—</span>
                    )}
                  </td>
                  <td style={{ padding: 8, fontWeight: 500 }}>{f.name}</td>
                  <td style={{ padding: 8, fontSize: 13 }}>{f.description}</td>
                  <td style={{ padding: 8 }}>
                    <span style={{ display: 'inline-block', width: 24, height: 24, background: f.color || '#ccc', borderRadius: 4, border: '1px solid #334155' }} title={f.color}></span>
                  </td>
                  <td style={{ padding: 8, textAlign: 'right' }}>
                    <button onClick={() => { setModalType('factions'); setModal('edit'); setCurrentItem(f); setFormName(f.name); setFormDesc(f.description || ''); setFormColor(f.color || '#FF0000'); }} style={{ marginRight: 8, padding: '4px 10px', fontSize: 12 }}>Modifier</button>
                    <button onClick={() => { setModalType('factions'); setModal('delete'); setCurrentItem(f); }} style={{ color: '#f87171', padding: '4px 10px', fontSize: 12 }}>Supprimer</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </section>
      )}

      {/* IA Companions View */}
      {activeView === 'companions' && (
        <section className="panel">
          <button onClick={backToOverview} style={{ marginBottom: 16 }}>← Retour aux points d'entrée</button>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
            <h2>Tous les IA Compagnons ({companionCount})</h2>
            <button onClick={() => { setModalType('companions'); setModal('create'); setFormName(''); setFormRole('Gouverneur'); setFormLevel(1); }} style={{ background: '#f59e0b', color: 'white', padding: '8px 16px', borderRadius: 6, border: 'none' }}>
              + Créer un compagnon IA
            </button>
          </div>

          <table className="data-table" style={{ width: '100%', borderCollapse: 'collapse' }}>
            <thead>
              <tr>
                <th style={{ textAlign: 'left', padding: 8 }}>Preview</th>
                <th style={{ textAlign: 'left', padding: 8 }}>Nom</th>
                <th style={{ textAlign: 'left', padding: 8 }}>Rôle</th>
                <th style={{ textAlign: 'left', padding: 8 }}>Niveau</th>
                <th style={{ textAlign: 'right', padding: 8 }}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {companions.map((c: any) => (
                <tr key={c.id} style={{ borderTop: '1px solid #334155' }}>
                  <td style={{ padding: 8 }}>
                    {c.url ? (
                      <img src={c.url} alt={c.name} style={{ width: 64, height: 64, objectFit: 'cover', borderRadius: 6, border: '1px solid #1e2937' }} />
                    ) : (
                      <span style={{ color: '#64748b', fontSize: 12 }}>—</span>
                    )}
                  </td>
                  <td style={{ padding: 8, fontWeight: 500 }}>{c.name}</td>
                  <td style={{ padding: 8 }}>{c.role}</td>
                  <td style={{ padding: 8 }}>{c.level}</td>
                  <td style={{ padding: 8, textAlign: 'right' }}>
                    <button onClick={() => { setModalType('companions'); setModal('edit'); setCurrentItem(c); setFormName(c.name); setFormRole(c.role); setFormLevel(c.level); }} style={{ marginRight: 8, padding: '4px 10px', fontSize: 12 }}>Modifier</button>
                    <button onClick={() => { setModalType('companions'); setModal('delete'); setCurrentItem(c); }} style={{ color: '#f87171', padding: '4px 10px', fontSize: 12 }}>Supprimer</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </section>
      )}

      {/* Worlds View - Gestion complète des Mondes (backend Go + IA serveur) */}
      {activeView === 'worlds' && (
        <section className="panel">
          <button onClick={backToOverview} style={{ marginBottom: 16 }}>← Retour aux points d'entrée</button>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
            <h2>Gestion des Mondes ({worldCount}) - 5 Continents, 500 joueurs max, 3 factions max</h2>
            <button onClick={async () => { await fetch('/api/nexus-game/worlds', { method: 'POST', credentials: 'same-origin' }); await fetchAll(); }} style={{ background: '#3b82f6', color: 'white', padding: '8px 16px', borderRadius: 6, border: 'none' }}>
              + Créer Nouveau Monde
            </button>
          </div>
          {loading ? <p>Chargement...</p> : error ? <p style={{color:'red'}}>{error}</p> : (
            <div>
              {worlds.length === 0 && <p>Aucun monde. Créez-en un (backend crée 5 continents automatiquement, assignation proportionnelle).</p>}
              {worlds.map((w: any, wi: number) => (
                <div key={wi} style={{ border: '1px solid #334155', borderRadius: 8, padding: 12, marginBottom: 12 }}>
                  <h4>{w.name || `Monde ${w.id}`} (ID {w.id}) - {w.is_active ? 'Actif' : 'Inactif'}</h4>
                  <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))', gap: 8, marginTop: 8 }}>
                    {(w.continents || []).map((c: any, ci: number) => {
                      const p = parseInt(c.players || '0');
                      const m = c.max_players || 500;
                      const f = parseInt(c.factions || '0');
                      const mf = c.max_factions || 3;
                      const pPct = m > 0 ? Math.min(100, Math.round((p / m) * 100)) : 0;
                      const fPct = mf > 0 ? Math.min(100, Math.round((f / mf) * 100)) : 0;
                      return (
                        <div key={ci} style={{ background: '#0f172a', padding: 8, borderRadius: 6, color: 'white' }}>
                          <div style={{ fontWeight: 600 }}>{c.name}</div>
                          <div>Joueurs: {p}/{m} ({pPct}%)</div>
                          <div style={{ height: 6, background: '#1e2937', borderRadius: 3, margin: '4px 0' }}><div style={{ width: `${pPct}%`, height: '100%', background: pPct > 80 ? '#ef4444' : '#3b82f6', borderRadius: 3 }} /></div>
                          <div>Factions: {f}/{mf} ({fPct}%)</div>
                          <div style={{ height: 6, background: '#1e2937', borderRadius: 3, margin: '4px 0' }}><div style={{ width: `${fPct}%`, height: '100%', background: '#10b981', borderRadius: 3 }} /></div>
                          {c.players_list && c.players_list.length > 0 && (
                            <div style={{ fontSize: 11, marginTop: 4, color: '#94a3b8' }}>Joueurs: {c.players_list.join(', ')}</div>
                          )}
                        </div>
                      );
                    })}
                  </div>
                  <div style={{ marginTop: 8, display: 'flex', gap: 8, flexWrap: 'wrap' }}>
                    <button onClick={async () => { await fetch(`/api/nexus-game/worlds/${w.id}/trigger-tick`, { method: 'POST', credentials: 'same-origin' }); alert('Tick + IA résumé exécuté (voir logs backend)'); }} style={{ padding: '4px 8px', fontSize: 12 }}>Déclencher Tick Monde + IA Serveur</button>
                    <button onClick={async () => { const r = await fetch(`/api/nexus-game/worlds/${w.id}/generate-event`, { method: 'POST', credentials: 'same-origin' }); const j = await r.json(); alert('Événement IA Serveur proposé (prompt optimisé): ' + (j.proposed_event?.title || JSON.stringify(j))); }} style={{ padding: '4px 8px', fontSize: 12 }}>Générer Événement IA (prompt optimisé)</button>
                  </div>
                </div>
              ))}
            </div>
          )}
          <p style={{ fontSize: 12, color: '#64748b', marginTop: 12 }}>Règle backend: 5 continents fixes. Max 500 joueurs/continent, max 3 factions/continent. Si plein → nouveau monde prioritaire pour nouvelles factions. Assignation auto sur création profil (basée sur faction). Redis pour counts/locks.</p>
        </section>
      )}

      {/* Prompts View - CRUD complet pour IA Serveur (optimisés, versionnés, modifiables) */}
      {activeView === 'prompts' && (
        <section className="panel">
          <button onClick={backToOverview} style={{ marginBottom: 16 }}>← Retour aux points d'entrée</button>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
            <h2>Prompts IA Serveur ({promptCount}) - Modifiables, versionnés, optimisés coût/rapidité/enrichissant</h2>
            <button onClick={() => {
              setPromptForm({ prompt_id: 'quest_seed_generation', version: 'v1.3', domain: 'quest_seed_generation', purpose: 'Génération de seeds de quêtes', system_prompt: 'System: You are the Nexus server IA. Generate a controlled quest seed. Output ONLY valid JSON. Optimized for low cost and fast response. Constructive, detailed and enriching with lore hooks and balanced outcomes. Respect max rewards and policies. User: [player report and world state]' });
              setEditingPrompt(null);
              setPromptModal(true);
            }} style={{ background: '#8b5cf6', color: 'white', padding: '8px 16px', borderRadius: 6, border: 'none' }}>
              + Créer Prompt (exemple 1)
            </button>
            <button onClick={() => {
              setPromptForm({ prompt_id: 'event_generation', version: 'v1.1', domain: 'event_generation', purpose: 'Génération d\'événements mondiaux', system_prompt: 'System: Generate a world event proposal. Max 4 per day. Linked to region/faction. Output JSON with title, summary (enriching narrative), duration, difficulty, rewards_cap. Evolves with current tensions. User: [world state]' });
              setEditingPrompt(null);
              setPromptModal(true);
            }} style={{ background: '#8b5cf6', color: 'white', padding: '8px 16px', borderRadius: 6, border: 'none' }}>
              + Créer Prompt (exemple 2)
            </button>
          </div>
          {loading ? <p>Chargement...</p> : error ? <p style={{color:'red'}}>{error}</p> : (
            <table className="data-table" style={{ width: '100%', borderCollapse: 'collapse' }}>
              <thead>
                <tr>
                  <th style={{ textAlign: 'left', padding: 8 }}>prompt_id @ version</th>
                  <th style={{ textAlign: 'left', padding: 8 }}>Domain / Purpose</th>
                  <th style={{ textAlign: 'left', padding: 8 }}>Prompt (tronqué)</th>
                  <th style={{ textAlign: 'right', padding: 8 }}>Actions</th>
                </tr>
              </thead>
              <tbody>
                {prompts.map((p: any, pi: number) => (
                  <tr key={pi} style={{ borderTop: '1px solid #334155' }}>
                    <td style={{ padding: 8, fontWeight: 500 }}>{p.prompt_id} @ {p.version}</td>
                    <td style={{ padding: 8, fontSize: 13 }}>{p.domain} / {p.purpose}</td>
                    <td style={{ padding: 8, fontSize: 12, color: '#64748b', maxWidth: 280, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{(p.system_prompt || '').substring(0, 70)}...</td>
                    <td style={{ padding: 8, textAlign: 'right' }}>
                      <button onClick={() => openPromptModal(p)} style={{ marginRight: 8, padding: '4px 10px', fontSize: 12 }}>Modifier (évoluer)</button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
          <p style={{ fontSize: 12, color: '#64748b', marginTop: 12 }}>Prompts utilisés par l'IA serveur (world tick, events, lore, tribunal, seeds). Modifiables ici. Évoluent automatiquement avec l'état du monde/jour/univers. Optimisés coût, rapidité, constructif/enrichissant.</p>
        </section>
      )}

      {/* Stats & Visualisation */}
      {activeView === 'stats' && (
        <section className="panel">
          <button onClick={backToOverview} style={{ marginBottom: 16 }}>← Retour aux points d'entrée</button>
          <h2>Stats & Visualisation Mondes (capacité 5 continents × 500 = 2500 joueurs max par monde)</h2>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(160px, 1fr))', gap: 16, marginTop: 16 }}>
            <div className="stat-card"><div className="label">Mondes</div><div className="value">{worldCount}</div></div>
            <div className="stat-card"><div className="label">Factions</div><div className="value">{factionCount}</div></div>
            <div className="stat-card"><div className="label">Capacité max / monde</div><div className="value">2500</div></div>
          </div>
          <div style={{ marginTop: 24 }}>
            <h3>Visualisation Distribution Joueurs par Continent (barres proportionnelles)</h3>
            {worlds.length > 0 ? worlds.map((w: any, wi: number) => (
              <div key={wi} style={{ marginBottom: 16 }}>
                <strong>{w.name} (priorité assignations)</strong>
                <div style={{ display: 'flex', gap: 4, marginTop: 4, height: 28 }}>
                  {(w.continents || []).map((c: any, ci: number) => {
                    const p = parseInt(c.players || 0);
                    const pct = Math.round((p / (c.max_players || 500)) * 100);
                    return <div key={ci} title={`${c.name}: ${p} joueurs (${pct}%)`} style={{ flex: 1, background: '#1e2937', position: 'relative', borderRadius: 4, overflow: 'hidden', border: '1px solid #334155' }}>
                      <div style={{ width: `${pct}%`, height: '100%', background: pct > 80 ? '#ef4444' : '#3b82f6' }} />
                      <div style={{ position: 'absolute', top: 4, left: 4, fontSize: 11, color: 'white', textShadow: '0 1px 1px black' }}>{c.name?.substring(0,8)}: {pct}%</div>
                    </div>;
                  })}
                </div>
              </div>
            )) : <p>Créez des mondes pour visualiser la distribution proportionnelle (max 3 factions/continent).</p>}
          </div>
          <p style={{ fontSize: 12, color: '#64748b', marginTop: 12 }}>Si tous continents pleins → nouveau monde auto. Assignation auto sur création profil (basée sur faction choisie). Redis pour counts/locks.</p>
        </section>
      )}

      {/* IA Server Tools */}
      {activeView === 'ia' && (
        <section className="panel">
          <button onClick={backToOverview} style={{ marginBottom: 16 }}>← Retour aux points d'entrée</button>
          <h2>Outils IA Serveur (gestion complète des prompts, ticks, events)</h2>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 12 }}>
            <button onClick={async () => { if (worlds[0]) { const r = await fetch(`/api/nexus-game/worlds/${worlds[0].id}/generate-event`, {method:'POST', credentials:'same-origin'}); const j = await r.json(); alert('IA Event proposé (prompt optimisé): ' + JSON.stringify(j.proposed_event || j)); } else alert('Créez un monde d\'abord'); }} style={{ padding: '10px 16px' }}>Générer Événement IA Serveur</button>
            <button onClick={async () => { if (worlds[0]) { await fetch(`/api/nexus-game/worlds/${worlds[0].id}/trigger-tick`, {method:'POST', credentials:'same-origin'}); alert('World Tick + IA résumé exécuté (logs backend)'); } else alert('Créez un monde d\'abord'); }} style={{ padding: '10px 16px' }}>Déclencher Tick + IA Serveur</button>
            <button onClick={() => goToView('prompts')} style={{ padding: '10px 16px' }}>Gérer Prompts (CRUD, versions, évolution)</button>
          </div>
          <p style={{ marginTop: 12, fontSize: 13, color: '#64748b' }}>Prompts optimisés (coût, rapidité, constructif/enrichissant). Évoluent automatiquement avec l'état du monde/jour/univers. Logs et coûts traçables. Tous les appels IA serveur respectent les limits (pas de bypass policies).</p>
        </section>
      )}

      {/* Generic CRUD Popins - simplified for factions and companions, full for avatars */}
      {modal && (
        <div style={{ position: 'fixed', top: 0, left: 0, right: 0, bottom: 0, background: 'rgba(0,0,0,0.75)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1000 }}>
          <div className="panel" style={{ width: 460, maxWidth: '92%', position: 'relative', padding: 24 }}>
            <button onClick={closeModal} style={{ position: 'absolute', top: 12, right: 16, background: 'none', border: 'none', fontSize: 24, cursor: 'pointer' }}>×</button>

            {/* Full avatar create/edit with name + image (WebP) */}
            {modalType === 'avatars' && (modal === 'create' || modal === 'edit') && (
              <>
                <h3>{modal === 'create' ? 'Créer un Avatar' : 'Modifier l\'Avatar'} {currentItem ? '#' + currentItem.id : ''}</h3>
                <p style={{ fontSize: 13, color: '#64748b' }}>Nom + image → conversion WebP obligatoire côté serveur.</p>
                <div style={{ marginTop: 12 }}>
                  <label style={{ display: 'block', marginBottom: 6, fontSize: 13, fontWeight: 500 }}>Nom</label>
                  <input type="text" value={formName} onChange={e => setFormName(e.target.value)} placeholder="Nom de l'avatar" style={{ width: '100%', padding: 10, marginBottom: 12 }} />
                  <label style={{ display: 'block', marginBottom: 6, fontSize: 13, fontWeight: 500 }}>Image (jpg/png)</label>
                  <input type="file" accept="image/*" onChange={e => setFormFile(e.target.files?.[0] || null)} style={{ marginBottom: 16 }} />
                  {modal === 'edit' && currentItem && currentItem.url && <img src={currentItem.url} alt="" style={{ width: 80, height: 80, objectFit: 'cover', borderRadius: 6, marginBottom: 12 }} />}
                  <button onClick={async () => {
                    setSubmitting(true);
                    const formData = new FormData();
                    formData.append("name", formName);
                    if (formFile) formData.append("image", formFile);
                    const url = modal === 'create' ? '/api/nexus-game/assets/avatar' : `/api/nexus-game/assets/avatars/${currentItem?.id}`;
                    const method = modal === 'create' ? 'POST' : 'PUT';
                    const res = await fetch(url, { method, body: formData, credentials: 'same-origin' });
                    if (res.ok) { closeModal(); await fetchAll(); } else setError(await res.text());
                    setSubmitting(false);
                  }} disabled={submitting || !formName} style={{ width: '100%', padding: 12, background: '#7C3AED', color: 'white', border: 'none', borderRadius: 6, fontWeight: 600 }}>
                    {submitting ? '...' : (modal === 'create' ? 'Créer Avatar (WebP)' : 'Enregistrer')}
                  </button>
                </div>
              </>
            )}

            {/* Factions - now with name + image (WebP) like avatar */}
            {modalType === 'factions' && (modal === 'create' || modal === 'edit') && (
              <>
                <h3>{modal === 'create' ? 'Créer une Faction' : 'Modifier la Faction'}</h3>
                <p style={{ fontSize: 13, color: '#64748b' }}>Nom + image → WebP obligatoire.</p>
                <div style={{ marginTop: 12 }}>
                  <label style={{ display: 'block', marginBottom: 6, fontSize: 13, fontWeight: 500 }}>Nom</label>
                  <input type="text" value={formName} onChange={e => setFormName(e.target.value)} placeholder="Nom de la faction" style={{ width: '100%', padding: 10, marginBottom: 12 }} />
                  <label style={{ display: 'block', marginBottom: 6, fontSize: 13, fontWeight: 500 }}>Description</label>
                  <input type="text" value={formDesc} onChange={e => setFormDesc(e.target.value)} placeholder="Description" style={{ width: '100%', padding: 10, marginBottom: 12 }} />
                  <label style={{ display: 'block', marginBottom: 6, fontSize: 13, fontWeight: 500 }}>Couleur</label>
                  <input type="color" value={formColor} onChange={e => setFormColor(e.target.value)} style={{ marginBottom: 12 }} />
                  <label style={{ display: 'block', marginBottom: 6, fontSize: 13, fontWeight: 500 }}>Image (WebP)</label>
                  <input type="file" accept="image/*" onChange={e => setFormFile(e.target.files?.[0] || null)} style={{ marginBottom: 16 }} />
                  {modal === 'edit' && currentItem && currentItem.url && <img src={currentItem.url} alt="" style={{ width: 80, height: 80, objectFit: 'cover', borderRadius: 6, marginBottom: 12 }} />}
                  <button onClick={async () => {
                    setSubmitting(true);
                    const formData = new FormData();
                    formData.append("name", formName);
                    formData.append("description", formDesc);
                    formData.append("color", formColor);
                    if (formFile) formData.append("image", formFile);
                    const url = modal === 'create' ? '/api/nexus-game/factions' : `/api/nexus-game/factions/${currentItem?.id}`;
                    const method = modal === 'create' ? 'POST' : 'PUT';
                    const res = await fetch(url, { method, body: formData, credentials: 'same-origin' });
                    if (res.ok) { closeModal(); await fetchAll(); } else setError(await res.text());
                    setSubmitting(false);
                  }} disabled={submitting || !formName} style={{ width: '100%', padding: 12, background: '#10b981', color: 'white', border: 'none', borderRadius: 6, fontWeight: 600 }}>
                    {submitting ? '...' : (modal === 'create' ? 'Créer Faction (WebP)' : 'Enregistrer')}
                  </button>
                </div>
              </>
            )}

            {/* IA Compagnons - now with name + image (WebP) like avatar */}
            {modalType === 'companions' && (modal === 'create' || modal === 'edit') && (
              <>
                <h3>{modal === 'create' ? 'Créer un IA Compagnon' : 'Modifier l\'IA Compagnon'}</h3>
                <p style={{ fontSize: 13, color: '#64748b' }}>Nom + image → WebP obligatoire.</p>
                <div style={{ marginTop: 12 }}>
                  <label style={{ display: 'block', marginBottom: 6, fontSize: 13, fontWeight: 500 }}>Nom</label>
                  <input type="text" value={formName} onChange={e => setFormName(e.target.value)} placeholder="Nom du compagnon" style={{ width: '100%', padding: 10, marginBottom: 12 }} />
                  <label style={{ display: 'block', marginBottom: 6, fontSize: 13, fontWeight: 500 }}>Rôle</label>
                  <input type="text" value={formRole} onChange={e => setFormRole(e.target.value)} placeholder="Gouverneur / Stratège..." style={{ width: '100%', padding: 10, marginBottom: 12 }} />
                  <label style={{ display: 'block', marginBottom: 6, fontSize: 13, fontWeight: 500 }}>Niveau</label>
                  <input type="number" value={formLevel} onChange={e => setFormLevel(parseInt(e.target.value)||1)} style={{ width: '100%', padding: 10, marginBottom: 12 }} />
                  <label style={{ display: 'block', marginBottom: 6, fontSize: 13, fontWeight: 500 }}>Image (WebP)</label>
                  <input type="file" accept="image/*" onChange={e => setFormFile(e.target.files?.[0] || null)} style={{ marginBottom: 16 }} />
                  {modal === 'edit' && currentItem && currentItem.url && <img src={currentItem.url} alt="" style={{ width: 80, height: 80, objectFit: 'cover', borderRadius: 6, marginBottom: 12 }} />}
                  <button onClick={async () => {
                    setSubmitting(true);
                    const formData = new FormData();
                    formData.append("name", formName);
                    formData.append("role", formRole);
                    formData.append("level", String(formLevel));
                    formData.append("player_id", "1");
                    if (formFile) formData.append("image", formFile);
                    const url = modal === 'create' ? '/api/nexus-game/ia-companions' : `/api/nexus-game/ia-companions/${currentItem?.id}`;
                    const method = modal === 'create' ? 'POST' : 'PUT';
                    const res = await fetch(url, { method, body: formData, credentials: 'same-origin' });
                    if (res.ok) { closeModal(); await fetchAll(); } else setError(await res.text());
                    setSubmitting(false);
                  }} disabled={submitting || !formName} style={{ width: '100%', padding: 12, background: '#f59e0b', color: 'white', border: 'none', borderRadius: 6, fontWeight: 600 }}>
                    {submitting ? '...' : (modal === 'create' ? 'Créer Compagnon (WebP)' : 'Enregistrer')}
                  </button>
                </div>
              </>
            )}

            {/* Note: full avatar modals from previous version should be here for consistency when activeView==='avatars' */}
          </div>
        </div>
      )}

      {error && <div style={{ color: '#f87171', marginTop: 12 }}>{error}</div>}

      {/* Real Popin for Prompts CRUD */}
      {promptModal && (
        <div style={{ position: 'fixed', top: 0, left: 0, right: 0, bottom: 0, background: 'rgba(0,0,0,0.75)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1000 }}>
          <div className="panel" style={{ width: 520, maxWidth: '92%', position: 'relative', padding: 24 }}>
            <button onClick={closePromptModal} style={{ position: 'absolute', top: 12, right: 16, background: 'none', border: 'none', fontSize: 24, cursor: 'pointer' }}>×</button>
            <h3>{editingPrompt ? 'Modifier le Prompt IA' : 'Créer un Prompt IA Serveur'}</h3>
            <p style={{ fontSize: 13, color: '#64748b' }}>Versionné, optimisé coût/rapidité/enrichissant. Utilisé par l'IA serveur pour le tick, events, lore, etc.</p>
            <div style={{ marginTop: 12 }}>
              <label style={{ display: 'block', marginBottom: 6, fontSize: 13, fontWeight: 500 }}>Prompt ID</label>
              <input type="text" value={promptForm.prompt_id} onChange={e => setPromptForm({...promptForm, prompt_id: e.target.value})} placeholder="ex: quest_seed_generation" style={{ width: '100%', padding: 10, marginBottom: 12 }} disabled={!!editingPrompt} />
              <label style={{ display: 'block', marginBottom: 6, fontSize: 13, fontWeight: 500 }}>Version</label>
              <input type="text" value={promptForm.version} onChange={e => setPromptForm({...promptForm, version: e.target.value})} placeholder="v1.0" style={{ width: '100%', padding: 10, marginBottom: 12 }} disabled={!!editingPrompt} />
              <label style={{ display: 'block', marginBottom: 6, fontSize: 13, fontWeight: 500 }}>Domain</label>
              <input type="text" value={promptForm.domain} onChange={e => setPromptForm({...promptForm, domain: e.target.value})} placeholder="quest_seed_generation" style={{ width: '100%', padding: 10, marginBottom: 12 }} />
              <label style={{ display: 'block', marginBottom: 6, fontSize: 13, fontWeight: 500 }}>Purpose</label>
              <input type="text" value={promptForm.purpose} onChange={e => setPromptForm({...promptForm, purpose: e.target.value})} placeholder="Génération de seeds" style={{ width: '100%', padding: 10, marginBottom: 12 }} />
              <label style={{ display: 'block', marginBottom: 6, fontSize: 13, fontWeight: 500 }}>System Prompt (détaillé, optimisé)</label>
              <textarea value={promptForm.system_prompt} onChange={e => setPromptForm({...promptForm, system_prompt: e.target.value})} style={{ width: '100%', padding: 10, marginBottom: 16, height: 120 }} placeholder="System: ..." />
              <button onClick={submitPrompt} disabled={submitting} style={{ width: '100%', padding: 12, background: '#8b5cf6', color: 'white', border: 'none', borderRadius: 6, fontWeight: 600 }}>
                {submitting ? '...' : (editingPrompt ? 'Mettre à jour Prompt' : 'Créer Prompt')}
              </button>
            </div>
          </div>
        </div>
      )}
    </AdminShell>
  );
}