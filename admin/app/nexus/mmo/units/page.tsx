"use client";

import { useState, useEffect } from "react";
import { AdminShell } from "../../../components/AdminShell";

interface Unit {
  contentId: string;
  nameKey?: string;
  rarity: string;
  maxLevel: number;
  healthBase?: number;
  attackBase?: number;
  assetId?: string;
  assetsByTier?: Record<string, string>;
}

export default function UnitsAdminPage() {
  const [items, setItems] = useState<Unit[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [modal, setModal] = useState<'create' | 'edit' | 'delete' | null>(null);
  const [current, setCurrent] = useState<Unit | null>(null);
  const [form, setForm] = useState<any>({});
  const [uploading, setUploading] = useState<string | null>(null);

  const fetchItems = async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch("/api/nexus-game/admin/content/units", { credentials: "same-origin" });
      const data = await res.json();
      setItems(data.units || []);
    } catch (e: any) { setError(e.message); } finally { setLoading(false); }
  };
  useEffect(() => { fetchItems(); }, []);

  const openCreate = () => { setCurrent(null); setForm({ contentId: '', nameKey: '', rarity: 'common', maxLevel: 30, healthBase: 100, attackBase: 20 }); setModal('create'); };
  const openEdit = (item: Unit) => { setCurrent(item); setForm({ ...item }); setModal('edit'); };
  const openDelete = (item: Unit) => { setCurrent(item); setModal('delete'); };
  const closeModal = () => { setModal(null); setCurrent(null); setForm({}); setError(null); };

  const submitForm = async () => {
    const isEdit = !!current;
    const url = isEdit ? `/api/nexus-game/admin/content/units/${current.contentId}` : `/api/nexus-game/admin/content/units`;
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
      await fetch(`/api/nexus-game/admin/content/units/${current.contentId}`, { method: 'DELETE', credentials: 'same-origin' });
      closeModal(); await fetchItems();
    } catch (e: any) { setError(e.message); } finally { setLoading(false); }
  };

  const uploadAsset = async (tier: string, file: File) => {
    const cid = form.contentId || current?.contentId;
    if (!cid) { setError("contentId requis"); return; }
    const fd = new FormData();
    fd.append("file", file); fd.append("domain", "unit"); fd.append("contentId", cid);
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
    <AdminShell title="Unités — Catalogue Nexus v2.0" description="CRUD des 15 unités (niveaux 1-30). Upload images tierées. Stats de base + assets.">
      <button onClick={() => window.location.href = '/nexus/mmo'} style={{ marginBottom: 16 }}>← Retour</button>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <h2>Unités ({items.length})</h2>
        <button onClick={openCreate} style={{ background: '#3b82f6', color: 'white', padding: '8px 16px', borderRadius: 6, border: 'none' }}>+ Créer Unité</button>
      </div>

      {loading && <p>Chargement...</p>}
      {error && <p style={{color:'red'}}>{error}</p>}

      <table className="data-table" style={{ width: '100%' }}>
        <thead><tr><th>Preview</th><th>contentId</th><th>Nom</th><th>Rareté</th><th>Niv Max</th><th>Assets</th><th>Actions</th></tr></thead>
        <tbody>
          {items.map(u => {
            const mainAsset = u.assetId || (u.assetsByTier && (u.assetsByTier.tier1 || u.assetsByTier.main));
            const previewUrl = mainAsset ? `/nexus-assets/content/units/${mainAsset}` : null;
            return (
              <tr key={u.contentId} style={{ borderTop: '1px solid #334155' }}>
                <td style={{ padding: 4 }}>
                  {previewUrl ? (
                    <img src={previewUrl} alt={u.contentId} style={{ width: 48, height: 48, objectFit: 'cover', borderRadius: 4, border: '1px solid #334155' }} onError={(e) => (e.currentTarget.style.display = 'none')} />
                  ) : (
                    <div style={{ width: 48, height: 48, background: '#1e2937', borderRadius: 4, display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 10, color: '#64748b' }}>no img</div>
                  )}
                </td>
                <td style={{ fontFamily: 'monospace', fontSize: 12 }}>{u.contentId}</td>
                <td>{u.nameKey}</td>
                <td>{u.rarity}</td>
                <td>{u.maxLevel}</td>
                <td style={{ fontSize: 11 }}>{u.assetId || (u.assetsByTier && Object.keys(u.assetsByTier).join(', '))}</td>
                <td><button onClick={() => openEdit(u)} style={{ fontSize: 12, marginRight: 6 }}>Éditer</button><button onClick={() => openDelete(u)} style={{ fontSize: 12, color: '#f87171' }}>Suppr</button></td>
              </tr>
            );
          })}
        </tbody>
      </table>

      {modal && (
        <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.7)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 100 }}>
          <div className="panel" style={{ width: 580, padding: 24, position: 'relative' }}>
            <button onClick={closeModal} style={{ position: 'absolute', top: 8, right: 12, fontSize: 24, background: 'none', border: 'none' }}>×</button>
            <h3>{modal === 'create' ? 'Créer Unité' : modal === 'edit' ? 'Éditer Unité' : 'Supprimer Unité'}</h3>

            {modal === 'delete' && current ? (
              <><p>Supprimer <strong>{current.contentId}</strong> ?</p><button onClick={doDelete} style={{ background: '#ef4444', color: 'white', padding: '8px 16px' }}>Confirmer</button> <button onClick={closeModal}>Annuler</button></>
            ) : (
              <>
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8 }}>
                  <input placeholder="contentId" value={form.contentId || ''} onChange={e=>setForm({...form, contentId:e.target.value})} />
                  <input placeholder="nameKey" value={form.nameKey || ''} onChange={e=>setForm({...form, nameKey:e.target.value})} />
                  <select value={form.rarity || 'common'} onChange={e=>setForm({...form, rarity:e.target.value})}>
                    <option>common</option><option>uncommon</option><option>rare</option><option>epic</option><option>legendary</option><option>nexus</option>
                  </select>
                  <input type="number" placeholder="maxLevel" value={form.maxLevel || 30} onChange={e=>setForm({...form, maxLevel:parseInt(e.target.value)})} />
                  <input type="number" placeholder="healthBase" value={form.healthBase || ''} onChange={e=>setForm({...form, healthBase:parseInt(e.target.value)})} />
                  <input type="number" placeholder="attackBase" value={form.attackBase || ''} onChange={e=>setForm({...form, attackBase:parseInt(e.target.value)})} />
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

                <div style={{ marginTop: 16 }}>
                  <button onClick={submitForm} disabled={loading || !form.contentId} style={{ background: '#3b82f6', color: 'white', padding: '10px 20px' }}>{modal==='create'?'Créer':'Sauvegarder'}</button>
                  <button onClick={closeModal} style={{ marginLeft: 8 }}>Annuler</button>
                </div>
              </>
            )}
          </div>
        </div>
      )}
    </AdminShell>
  );
}
