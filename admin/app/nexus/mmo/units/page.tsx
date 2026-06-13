"use client";

import { useState, useEffect } from "react";
import { AdminShell } from "../../../components/AdminShell";
import { AssetPreview, CatalogIdentity, CatalogTable, CostSummary, DescriptionSummary, EffectsPreview, LevelDescriptionsPreview, TranslationMap, fetchCatalogTranslations } from "../contentDisplay";

const API_BASE = (process.env.NEXT_PUBLIC_NEXUS_API_BASE || "").replace(/\/$/, "");

interface Unit {
  id: number;
  contentId: string;
  type?: string;
  domain?: string;
  nameKey?: string;
  descriptionKey?: string;
  flavorTextKey?: string;
  levelDescriptionKeys?: Record<string, string>;
  rarity: string;
  maxLevel: number;
  healthBase?: number;
  attackBase?: number;
  defenseBase?: number;
  speedBase?: number;
  trainingTimeBaseSeconds?: number;
  upkeepBase?: number;
  assetId?: string;
  assetsByTier?: Record<string, string>;
  effects?: string;
  isPublished?: boolean;
}

const ASSET_KEYS = ["main", "tier1", "tier2", "tier3", "tier4"] as const;
const UNIT_TYPES = ["infantry", "drone", "support", "special", "mecha", "artillerie", "defense", "commander", "officer"] as const;

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
  const [translations, setTranslations] = useState<TranslationMap>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [modal, setModal] = useState<'create' | 'edit' | 'delete' | null>(null);
  const [current, setCurrent] = useState<Unit | null>(null);
  const [form, setForm] = useState<any>({});
  const [uploading, setUploading] = useState<string | null>(null);

  const fetchItems = async () => {
    setLoading(true); setError(null);
    try {
      const [res, catalogTranslations] = await Promise.all([
        fetch(`${API_BASE}/api/nexus-game/admin/content/units`, { credentials: "same-origin" }),
        fetchCatalogTranslations(API_BASE, ["unit"]),
      ]);
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      setTranslations(catalogTranslations);
      setItems(data.units || []);
    } catch (e: any) { setError(e.message); } finally { setLoading(false); }
  };
  useEffect(() => { fetchItems(); }, []);

  const openCreate = () => { setCurrent(null); setForm({ contentId: '', domain: 'unit', type: 'infantry', nameKey: '', descriptionKey: '', flavorTextKey: '', levelDescriptionKeys: '{}', rarity: 'common', maxLevel: 30, healthBase: 100, attackBase: 20, defenseBase: 10, speedBase: 10, trainingTimeBaseSeconds: 300, upkeepBase: 1, isPublished: true }); setModal('create'); };
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

      <CatalogTable>
        <thead><tr><th style={{ width: 260 }}>Fiche</th><th style={{ width: 310 }}>Description</th><th style={{ width: 210 }}>Stats / coûts</th><th style={{ width: 300 }}>Profil tactique</th><th style={{ width: 280 }}>Niveaux</th><th style={{ width: 210 }}>Médias</th><th style={{ width: 130 }}>Actions</th></tr></thead>
        <tbody>
          {items.map(u => {
            const assets = collectAssets(u);
            return (
              <tr key={u.contentId || `unit-${u.id}`} style={{ borderTop: '1px solid #334155', verticalAlign: 'top' }}>
                <td>
                  <CatalogIdentity
                    translations={translations}
                    titleKey={u.nameKey}
                    contentId={u.contentId}
                    badges={[u.type ? `type ${u.type}` : 'type manquant', u.rarity || '-', `lvl ${u.maxLevel || '-'}`, u.isPublished === false ? 'draft' : 'published']}
                  />
                </td>
                <td><DescriptionSummary translations={translations} descriptionKey={u.descriptionKey} flavorTextKey={u.flavorTextKey} /></td>
                <td><CostSummary item={u as any} kind="unit" /></td>
                <td><EffectsPreview effects={u.effects} /></td>
                <td><LevelDescriptionsPreview keys={u.levelDescriptionKeys} translations={translations} /></td>
                <td><AssetPreview assets={assets} fallbackLabel={u.contentId || `unit-${u.id}`} /></td>
                <td><button onClick={() => openEdit(u)} style={{ fontSize: 12, marginRight: 6 }}>Éditer</button><button onClick={() => openDelete(u)} style={{ fontSize: 12, color: '#f87171' }}>Suppr</button></td>
              </tr>
            );
          })}
        </tbody>
      </CatalogTable>

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
                  <select value={form.type || ''} onChange={e=>setForm({...form, type:e.target.value})}>
                    <option value="" disabled>type tactique requis</option>
                    {UNIT_TYPES.map((type) => <option value={type} key={type}>{type}</option>)}
                  </select>
                  <input placeholder="nameKey" value={form.nameKey || ''} onChange={e=>setForm({...form, nameKey:e.target.value})} />
                  <input placeholder="descriptionKey" value={form.descriptionKey || ''} onChange={e=>setForm({...form, descriptionKey:e.target.value})} />
                  <input placeholder="flavorTextKey (optionnel)" value={form.flavorTextKey || ''} onChange={e=>setForm({...form, flavorTextKey:e.target.value})} />
                  <select value={form.rarity || 'common'} onChange={e=>setForm({...form, rarity:e.target.value})}>
                    <option>common</option><option>uncommon</option><option>rare</option><option>epic</option><option>legendary</option><option>nexus</option>
                  </select>
                  <input type="number" placeholder="maxLevel" value={form.maxLevel || 30} onChange={e=>setForm({...form, maxLevel:parseInt(e.target.value)})} />
                  <input type="number" placeholder="healthBase" value={form.healthBase || ''} onChange={e=>setForm({...form, healthBase:parseInt(e.target.value)})} />
                  <input type="number" placeholder="attackBase" value={form.attackBase || ''} onChange={e=>setForm({...form, attackBase:parseInt(e.target.value)})} />
                  <input type="number" placeholder="defenseBase" value={form.defenseBase || ''} onChange={e=>setForm({...form, defenseBase:parseInt(e.target.value)})} />
                  <input type="number" placeholder="speedBase" value={form.speedBase || ''} onChange={e=>setForm({...form, speedBase:parseInt(e.target.value)})} />
                  <input type="number" placeholder="trainingTimeBaseSeconds" value={form.trainingTimeBaseSeconds || ''} onChange={e=>setForm({...form, trainingTimeBaseSeconds:parseInt(e.target.value)})} />
                  <input type="number" placeholder="upkeepBase" value={form.upkeepBase || ''} onChange={e=>setForm({...form, upkeepBase:parseInt(e.target.value)})} />
                  <label style={{ display: 'flex', alignItems: 'center', gap: 8, color: '#cbd5e1', fontSize: 13 }}>
                    <input type="checkbox" checked={form.isPublished !== false} onChange={e=>setForm({...form, isPublished:e.target.checked})} />
                    Publiée / utilisable
                  </label>
                </div>

                <p style={{ marginTop: 8, color: '#94a3b8', fontSize: 12 }}>
                  Le type tactique est utilisé par les slots d'armée: infantry, drone, support, special, mecha, artillerie, defense, commander, officer.
                </p>

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
                  <button onClick={submitForm} disabled={loading || !form.contentId || !form.type} style={{ background: '#3b82f6', color: 'white', padding: '10px 20px' }}>{modal==='create'?'Créer':'Sauvegarder'}</button>
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
