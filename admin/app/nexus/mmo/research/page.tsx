"use client";

import { useState, useEffect } from "react";
import { AdminShell } from "../../../components/AdminShell";

interface Research {
  contentId: string;
  nameKey?: string;
  branch?: string;
  tier?: number;
  rarity: string;
  costBaseCredits?: number;
  durationBaseSeconds?: number;
  effectsJSON?: string;
  prerequisitesJSON?: string;
  assetId?: string;
  assetsByTier?: Record<string, string>;
}

export default function ResearchAdminPage() {
  const [items, setItems] = useState<Research[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [modal, setModal] = useState<'create' | 'edit' | 'delete' | null>(null);
  const [current, setCurrent] = useState<Research | null>(null);
  const [form, setForm] = useState<any>({});
  const [uploading, setUploading] = useState<string | null>(null);

  const fetchItems = async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch("/api/nexus-game/admin/content/research", { credentials: "same-origin" });
      const data = await res.json();
      setItems(data.research || []);
    } catch (e: any) { setError(e.message); } finally { setLoading(false); }
  };
  useEffect(() => { fetchItems(); }, []);

  const openCreate = () => { setCurrent(null); setForm({ contentId: '', nameKey: '', branch: 'economy', tier: 1, rarity: 'common', costBaseCredits: 500, durationBaseSeconds: 3600, effectsJSON: '[]', prerequisitesJSON: '[]' }); setModal('create'); };
  const openEdit = (item: Research) => { setCurrent(item); setForm({ ...item }); setModal('edit'); };
  const openDelete = (item: Research) => { setCurrent(item); setModal('delete'); };
  const closeModal = () => { setModal(null); setCurrent(null); setForm({}); setError(null); };

  const submitForm = async () => {
    const isEdit = !!current;
    const url = isEdit ? `/api/nexus-game/admin/content/research/${current.contentId}` : `/api/nexus-game/admin/content/research`;
    const method = isEdit ? 'PUT' : 'POST';
    setLoading(true);
    try {
      const res = await fetch(url, { method, headers: { 'Content-Type': 'application/json' }, credentials: 'same-origin', body: JSON.stringify(form) });
      if (!res.ok) throw new Error(await res.text());
      closeModal(); await fetchItems();
    } catch (e: any) { setError(e.message); } finally { setLoading(false); }
  };

  const doDelete = async () => {
    if (!current) return;
    setLoading(true);
    try {
      await fetch(`/api/nexus-game/admin/content/research/${current.contentId}`, { method: 'DELETE', credentials: 'same-origin' });
      closeModal(); await fetchItems();
    } catch (e: any) { setError(e.message); } finally { setLoading(false); }
  };

  const uploadAsset = async (tier: string, file: File) => {
    const cid = form.contentId || current?.contentId;
    if (!cid) { setError("contentId requis"); return; }
    const fd = new FormData();
    fd.append("file", file); fd.append("domain", "research"); fd.append("contentId", cid);
    if (tier) fd.append("tier", tier);
    setUploading(tier || 'main');
    try {
      const res = await fetch("/api/nexus-game/admin/content/upload-asset", { method: "POST", body: fd, credentials: "same-origin" });
      const data = await res.json();
      const saved = data.savedAs || '';
      const key = tier ? `tier${tier}` : 'main';
      const newAssets = { ...(form.assetsByTier || {}) };
      if (tier) newAssets[key] = saved; else setForm((f:any)=>({...f, assetId: saved}));
      setForm((f:any)=>({...f, assetsByTier: newAssets}));
    } catch(e:any){ setError(e.message); } finally { setUploading(null); }
  };

  return (
    <AdminShell title="Recherches — Arbre Nexus v2.0" description="11 branches × 7 tiers. CRUD + upload assets + dépendances (prerequisitesJSON).">
      <button onClick={() => window.location.href = '/nexus/mmo'} style={{ marginBottom: 16 }}>← Retour</button>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <h2>Recherches ({items.length})</h2>
        <button onClick={openCreate} style={{ background: '#8b5cf6', color: 'white', padding: '8px 16px', borderRadius: 6, border: 'none' }}>+ Créer Nœud</button>
      </div>

      {loading && <p>Chargement...</p>}
      {error && <p style={{color:'red'}}>{error}</p>}

      <table className="data-table" style={{ width: '100%' }}>
        <thead><tr><th>contentId</th><th>Nom</th><th>Branche</th><th>Tier</th><th>Rareté</th><th>Assets</th><th>Actions</th></tr></thead>
        <tbody>
          {items.map(r => (
            <tr key={r.contentId} style={{ borderTop: '1px solid #334155' }}>
              <td style={{ fontFamily: 'monospace', fontSize: 12 }}>{r.contentId}</td>
              <td>{r.nameKey}</td>
              <td>{r.branch}</td>
              <td>{r.tier}</td>
              <td>{r.rarity}</td>
              <td style={{ fontSize: 11 }}>{r.assetId || (r.assetsByTier && Object.keys(r.assetsByTier).join(', '))}</td>
              <td><button onClick={() => openEdit(r)} style={{ fontSize: 12, marginRight: 6 }}>Éditer</button><button onClick={() => openDelete(r)} style={{ fontSize: 12, color: '#f87171' }}>Suppr</button></td>
            </tr>
          ))}
        </tbody>
      </table>

      {modal && (
        <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.7)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 100 }}>
          <div className="panel" style={{ width: 620, padding: 24, position: 'relative' }}>
            <button onClick={closeModal} style={{ position: 'absolute', top: 8, right: 12, fontSize: 24, background: 'none', border: 'none' }}>×</button>
            <h3>{modal === 'create' ? 'Créer Nœud Recherche' : modal === 'edit' ? 'Éditer Nœud' : 'Supprimer Nœud'}</h3>

            {modal === 'delete' && current ? (
              <><p>Supprimer <strong>{current.contentId}</strong> ?</p><button onClick={doDelete} style={{ background: '#ef4444', color: 'white', padding: '8px 16px' }}>Confirmer</button> <button onClick={closeModal}>Annuler</button></>
            ) : (
              <>
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8 }}>
                  <input placeholder="contentId (ex: research_economy_tier1)" value={form.contentId || ''} onChange={e=>setForm({...form, contentId:e.target.value})} />
                  <input placeholder="nameKey" value={form.nameKey || ''} onChange={e=>setForm({...form, nameKey:e.target.value})} />
                  <input placeholder="branch (economy, military...)" value={form.branch || ''} onChange={e=>setForm({...form, branch:e.target.value})} />
                  <input type="number" placeholder="tier (1-7)" value={form.tier || 1} onChange={e=>setForm({...form, tier:parseInt(e.target.value)})} />
                  <select value={form.rarity || 'common'} onChange={e=>setForm({...form, rarity:e.target.value})}>
                    <option>common</option><option>uncommon</option><option>rare</option><option>epic</option><option>legendary</option><option>nexus</option>
                  </select>
                  <input type="number" placeholder="costBaseCredits" value={form.costBaseCredits || ''} onChange={e=>setForm({...form, costBaseCredits:parseInt(e.target.value)})} />
                </div>

                <div style={{ marginTop: 12 }}>
                  <div style={{ fontSize: 12, color: '#94a3b8', marginBottom: 4 }}>Assets (upload par tier)</div>
                  {['','1','2','3','4'].map(t => {
                    const key = t ? `tier${t}` : 'main';
                    const cur = (form.assetsByTier && form.assetsByTier[key]) || (t==='' ? form.assetId : '');
                    return (
                      <div key={key} style={{ display: 'inline-block', marginRight: 8, border: '1px solid #334155', padding: 4, borderRadius: 4 }}>
                        <div style={{ fontSize: 11 }}>{key}</div>
                        <input type="file" onChange={e=>{ if (e.target.files?.[0]) uploadAsset(t, e.target.files[0]); }} disabled={uploading===key} />
                        {cur && <div style={{ fontSize: 10 }}>{cur}</div>}
                      </div>
                    );
                  })}
                </div>

                <div style={{ marginTop: 12, display: 'flex', gap: 8 }}>
                  <button onClick={submitForm} disabled={loading || !form.contentId} style={{ background: '#8b5cf6', color: 'white', padding: '10px 20px' }}>{modal==='create'?'Créer':'Sauvegarder'}</button>
                  <button onClick={closeModal}>Annuler</button>
                </div>
                <p style={{ fontSize: 11, color: '#64748b', marginTop: 8 }}>Utilisez prerequisitesJSON pour les dépendances (ex: ["research_economy_tier1"]).</p>
              </>
            )}
          </div>
        </div>
      )}
    </AdminShell>
  );
}
