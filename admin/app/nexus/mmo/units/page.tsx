"use client";

import { useState, useEffect } from "react";
import { AdminShell } from "../../../components/AdminShell";

const API_BASE = (process.env.NEXT_PUBLIC_NEXUS_API_BASE || "").replace(/\/$/, "");

interface Unit {
  id: number;
  contentId: string;
  nameKey?: string;
  descriptionKey?: string;
  flavorTextKey?: string;
  levelDescriptionKeys?: Record<string, string>;
  rarity: string;
  maxLevel: number;
  healthBase?: number;
  attackBase?: number;
  assetId?: string;
  assetsByTier?: Record<string, string>;
}

const ASSET_KEYS = ["main", "tier1", "tier2", "tier3", "tier4"] as const;

function buildAssetUrl(folder: string, fileName?: string) {
  if (!fileName) return null;
  if (/^https?:\/\//.test(fileName)) return fileName;
  if (fileName.startsWith("/")) return `${API_BASE}${fileName}`;
  return `${API_BASE}/nexus-assets/content/${folder}/${encodeURIComponent(fileName)}`;
}

function collectAssets(item: Pick<Unit, "assetId" | "assetsByTier">) {
  const assetsByTier = item.assetsByTier && typeof item.assetsByTier === "object"
    ? item.assetsByTier
    : {};

  return ASSET_KEYS.map((key) => {
    const fileName = key === "main" ? (item.assetId || assetsByTier.main) : assetsByTier[key];
    return { key, fileName, url: buildAssetUrl("units", fileName) };
  }).filter((asset) => Boolean(asset.fileName));
}

function hasMeaningfulUnitData(item: Unit) {
  return Boolean(item.contentId || item.nameKey || item.descriptionKey || item.flavorTextKey || item.assetId || (item.levelDescriptionKeys && Object.keys(item.levelDescriptionKeys).length > 0) || (item.assetsByTier && Object.keys(item.assetsByTier).length > 0));
}

function stringifyRecord(value: unknown) {
  if (!value) return "{}";
  if (typeof value === "string") return value;
  return JSON.stringify(value, null, 2);
}

function parseRecord(value: unknown) {
  if (!value) return {};
  if (typeof value === "object") return value;
  if (typeof value !== "string") return {};
  try {
    const parsed = JSON.parse(value);
    return parsed && typeof parsed === "object" && !Array.isArray(parsed) ? parsed : {};
  } catch {
    throw new Error("levelDescriptionKeys doit être un objet JSON valide");
  }
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
      const res = await fetch(`${API_BASE}/api/nexus-game/admin/content/units`, { credentials: "same-origin" });
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      setItems(data.units || []);
    } catch (e: any) { setError(e.message); } finally { setLoading(false); }
  };
  useEffect(() => { fetchItems(); }, []);

  const openCreate = () => { setCurrent(null); setForm({ contentId: '', nameKey: '', descriptionKey: '', flavorTextKey: '', levelDescriptionKeys: '{}', rarity: 'common', maxLevel: 30, healthBase: 100, attackBase: 20 }); setModal('create'); };
  const openEdit = (item: Unit) => { setCurrent(item); setForm({ ...item, levelDescriptionKeys: stringifyRecord(item.levelDescriptionKeys) }); setModal('edit'); };
  const openDelete = (item: Unit) => { setCurrent(item); setModal('delete'); };
  const closeModal = () => { setModal(null); setCurrent(null); setForm({}); setError(null); };

  const submitForm = async () => {
    const isEdit = !!current;
    const url = isEdit ? `${API_BASE}/api/nexus-game/admin/content/units/${current.contentId}` : `${API_BASE}/api/nexus-game/admin/content/units`;
    const method = isEdit ? 'PUT' : 'POST';
    setLoading(true);
    try {
      const payload = { ...form, levelDescriptionKeys: parseRecord(form.levelDescriptionKeys) };
      const res = await fetch(url, { method, headers: { 'Content-Type': 'application/json' }, credentials: 'same-origin', body: JSON.stringify(payload) });
      if (!res.ok) throw new Error(await res.text());
      closeModal(); await fetchItems();
    } catch (e: any) { setError(e.message); } finally { setLoading(false); }
  };

  const doDelete = async () => {
    if (!current) return;
    setLoading(true);
    try {
      const target = current.contentId
        ? `${API_BASE}/api/nexus-game/admin/content/units/${encodeURIComponent(current.contentId)}`
        : `${API_BASE}/api/nexus-game/admin/content/units/by-id/${current.id}`;
      const res = await fetch(target, { method: 'DELETE', credentials: 'same-origin' });
      if (!res.ok) throw new Error(await res.text());
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
      const res = await fetch(`${API_BASE}/api/nexus-game/admin/content/upload-asset`, { method: "POST", body: fd, credentials: "same-origin" });
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      const saved = data.url || data.urlHint || data.publicPath || data.savedAs || '';
      const key = tier ? `tier${tier}` : 'main';
      const newAssets = { ...(form.assetsByTier || {}) };
      if (tier) newAssets[key] = saved; else setForm((f:any)=>({...f, assetId: saved}));
      setForm((f:any)=>({...f, assetsByTier: newAssets}));
    } catch(e:any){ setError(e.message); } finally { setUploading(null); }
  };

  const invalidItems = items.filter((item) => !item.contentId && hasMeaningfulUnitData(item));

  return (
    <AdminShell title="Unités — Catalogue Nexus v2.0" description="CRUD des 15 unités (niveaux 1-30). Upload images tierées. Stats de base + assets.">
      <button onClick={() => window.location.href = '/admin/nexus/mmo'} style={{ marginBottom: 16 }}>← Retour Nexus MMO</button>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <h2>Unités ({items.length})</h2>
        <button onClick={openCreate} style={{ background: '#3b82f6', color: 'white', padding: '8px 16px', borderRadius: 6, border: 'none' }}>+ Créer Unité</button>
      </div>

      {invalidItems.length > 0 && <p style={{ marginBottom: 12, color: '#fca5a5', fontSize: 13 }}>{invalidItems.length} entrée(s) invalide(s) sans <code>contentId</code> détectée(s).</p>}

      {loading && <p>Chargement...</p>}
      {error && <p style={{color:'red'}}>{error}</p>}

      <table className="data-table" style={{ width: '100%' }}>
        <thead><tr><th>Preview</th><th>contentId</th><th>Nom</th><th>Description</th><th>Rareté</th><th>Niv Max</th><th>Assets</th><th>Actions</th></tr></thead>
        <tbody>
          {items.map(u => {
            const assets = collectAssets(u);
            const previewAssets = assets.slice(0, 4);
            return (
              <tr key={u.contentId || `unit-${u.id}`} style={{ borderTop: '1px solid #334155' }}>
                <td style={{ padding: 4 }}>
                  {previewAssets.length > 0 ? (
                    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, 48px)', gap: 4 }}>
                      {previewAssets.map((asset) => (
                        <div key={asset.key} style={{ position: 'relative' }}>
                          <img src={asset.url!} alt={`${u.contentId || `unit-${u.id}`}-${asset.key}`} style={{ width: 48, height: 48, objectFit: 'cover', borderRadius: 4, border: '1px solid #334155', background: '#0f172a' }} onError={(e) => (e.currentTarget.style.opacity = '0.18')} />
                          <span style={{ position: 'absolute', left: 2, bottom: 2, fontSize: 9, padding: '1px 3px', borderRadius: 3, background: 'rgba(15,23,42,0.82)', color: '#cbd5e1' }}>{asset.key}</span>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <div style={{ width: 48, height: 48, background: '#1e2937', borderRadius: 4, display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 10, color: '#64748b' }}>no img</div>
                  )}
                </td>
                <td style={{ fontFamily: 'monospace', fontSize: 12 }}>{u.contentId || <span style={{ color: '#fca5a5' }}>#{u.id} — contentId manquant</span>}</td>
                <td>{u.nameKey || '—'}</td>
                <td style={{ fontFamily: 'monospace', fontSize: 11 }}>{u.descriptionKey || '—'}</td>
                <td>{u.rarity || '—'}</td>
                <td>{u.maxLevel || '—'}</td>
                <td style={{ fontSize: 11 }}>{u.assetId || (u.assetsByTier && Object.keys(u.assetsByTier).join(', '))}</td>
                <td><button onClick={() => openEdit(u)} style={{ fontSize: 12, marginRight: 6 }}>Éditer</button><button onClick={() => openDelete(u)} style={{ fontSize: 12, color: '#f87171' }}>Suppr</button></td>
              </tr>
            );
          })}
        </tbody>
      </table>

      {modal && (
        <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.7)', display: 'flex', alignItems: 'flex-start', justifyContent: 'center', zIndex: 100, overflowY: 'auto', padding: '24px 12px' }}>
          <div className="panel" style={{ width: 580, maxWidth: 'calc(100vw - 24px)', maxHeight: 'calc(100vh - 48px)', overflowY: 'auto', padding: 24, position: 'relative' }}>
            <button onClick={closeModal} style={{ position: 'absolute', top: 8, right: 12, fontSize: 24, background: 'none', border: 'none' }}>×</button>
            <h3>{modal === 'create' ? 'Créer Unité' : modal === 'edit' ? 'Éditer Unité' : 'Supprimer Unité'}</h3>

            {modal === 'delete' && current ? (
              <><p>Supprimer <strong>{current.contentId || `#${current.id}`}</strong> ?</p><button onClick={doDelete} style={{ background: '#ef4444', color: 'white', padding: '8px 16px' }}>Confirmer</button> <button onClick={closeModal}>Annuler</button></>
            ) : (
              <>
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))', gap: 8 }}>
                  <input placeholder="contentId" value={form.contentId || ''} onChange={e=>setForm({...form, contentId:e.target.value})} />
                  <input placeholder="nameKey" value={form.nameKey || ''} onChange={e=>setForm({...form, nameKey:e.target.value})} />
                  <input placeholder="descriptionKey" value={form.descriptionKey || ''} onChange={e=>setForm({...form, descriptionKey:e.target.value})} />
                  <input placeholder="flavorTextKey (optionnel)" value={form.flavorTextKey || ''} onChange={e=>setForm({...form, flavorTextKey:e.target.value})} />
                  <select value={form.rarity || 'common'} onChange={e=>setForm({...form, rarity:e.target.value})}>
                    <option>common</option><option>uncommon</option><option>rare</option><option>epic</option><option>legendary</option><option>nexus</option>
                  </select>
                  <input type="number" placeholder="maxLevel" value={form.maxLevel || 30} onChange={e=>setForm({...form, maxLevel:parseInt(e.target.value)})} />
                  <input type="number" placeholder="healthBase" value={form.healthBase || ''} onChange={e=>setForm({...form, healthBase:parseInt(e.target.value)})} />
                  <input type="number" placeholder="attackBase" value={form.attackBase || ''} onChange={e=>setForm({...form, attackBase:parseInt(e.target.value)})} />
                </div>

                <div style={{ marginTop: 12 }}>
                  <label style={{ fontSize: 12, color: '#94a3b8' }}>Descriptions par niveau (clés i18n JSON, ex: niveau 1 → nexus_game.unit.xxx.level_1.description)</label>
                  <textarea placeholder='{"1":"nexus_game.unit.example.level_1.description","2":"nexus_game.unit.example.level_2.description"}' value={stringifyRecord(form.levelDescriptionKeys)} onChange={e=>setForm({...form, levelDescriptionKeys:e.target.value})} style={{ width: '100%', height: 110, marginTop: 6, fontFamily: 'monospace', fontSize: 12 }} />
                </div>

                <div style={{ marginTop: 12 }}>
                  <div style={{ fontSize: 12, color: '#94a3b8', marginBottom: 4 }}>Assets (upload par tier)</div>
                  {['','1','2','3','4'].map(t => {
                    const key = t ? `tier${t}` : 'main';
                    const cur = (form.assetsByTier && form.assetsByTier[key]) || (t==='' ? form.assetId : '');
                    return (
                      <div key={key} style={{ display: 'inline-block', marginRight: 8, marginBottom: 8, border: '1px solid #334155', padding: 4, borderRadius: 4, maxWidth: '100%' }}>
                        <div style={{ fontSize: 11 }}>{key}</div>
                        <input type="file" style={{ maxWidth: '100%' }} onChange={e=>{ if (e.target.files?.[0]) uploadAsset(t, e.target.files[0]); }} disabled={uploading===key} />
                        {cur && <div style={{ fontSize: 10, wordBreak: 'break-all' }}>{cur}</div>}
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
