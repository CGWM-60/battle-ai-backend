"use client";

import { useState, useEffect } from "react";
import { AdminShell } from "../../../components/AdminShell";
import { AssetPreview, CatalogIdentity, CatalogTable, CostSummary, DescriptionSummary, EffectsPreview, LevelDescriptionsPreview, TranslationMap, fetchCatalogTranslations } from "../contentDisplay";

// Configurable backend base for when admin static is served separately from Go API (e.g. different port or Dokploy static).
// Set NEXT_PUBLIC_NEXUS_API_BASE=http://localhost:8080 (or prod URL) at build/dev time.
const API_BASE = (process.env.NEXT_PUBLIC_NEXUS_API_BASE || "").replace(/\/$/, "");

interface Research {
  id: number;
  contentId: string;
  nameKey?: string;
  descriptionKey?: string;
  flavorTextKey?: string;
  levelDescriptionKeys?: Record<string, string>;
  branch?: string;
  tier?: number;
  rarity: string;
  costBaseCredits?: number;
  costBaseData?: number;
  durationBaseSeconds?: number;
  effects?: string;
  effectsJSON?: string;
  prerequisitesJSON?: string;
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

function collectAssets(item: Pick<Research, "assetId" | "assetsByTier">) {
  const assetsByTier = item.assetsByTier && typeof item.assetsByTier === "object"
    ? item.assetsByTier
    : {};

  return ASSET_KEYS.map((key) => {
    const fileName = key === "main" ? (item.assetId || assetsByTier.main) : assetsByTier[key];
    return { key, fileName, url: buildAssetUrl("research", fileName) };
  }).filter((asset) => Boolean(asset.fileName));
}

function hasMeaningfulResearchData(item: Research) {
  return Boolean(item.contentId || item.nameKey || item.descriptionKey || item.flavorTextKey || item.branch || item.assetId || (item.levelDescriptionKeys && Object.keys(item.levelDescriptionKeys).length > 0) || (item.assetsByTier && Object.keys(item.assetsByTier).length > 0));
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

export default function ResearchAdminPage() {
  const [items, setItems] = useState<Research[]>([]);
  const [translations, setTranslations] = useState<TranslationMap>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [modal, setModal] = useState<'create' | 'edit' | 'delete' | null>(null);
  const [current, setCurrent] = useState<Research | null>(null);
  const [form, setForm] = useState<any>({});
  const [uploading, setUploading] = useState<string | null>(null);

  const fetchItems = async () => {
    setLoading(true); setError(null);
    try {
      const [res, catalogTranslations] = await Promise.all([
        fetch(`${API_BASE}/api/nexus-game/admin/content/research`, { credentials: "same-origin" }),
        fetchCatalogTranslations(API_BASE, ["research"]),
      ]);
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      setTranslations(catalogTranslations);
      setItems(data.research || []);
    } catch (e: any) { setError(e.message); } finally { setLoading(false); }
  };
  useEffect(() => { fetchItems(); }, []);

  const openCreate = () => { setCurrent(null); setForm({ contentId: '', nameKey: '', descriptionKey: '', flavorTextKey: '', levelDescriptionKeys: '{}', branch: 'economy', tier: 1, rarity: 'common', costBaseCredits: 500, durationBaseSeconds: 3600, effectsJSON: '[]', prerequisitesJSON: '[]' }); setModal('create'); };
  const openEdit = (item: Research) => { setCurrent(item); setForm({ ...item, levelDescriptionKeys: stringifyRecord(item.levelDescriptionKeys) }); setModal('edit'); };
  const openDelete = (item: Research) => { setCurrent(item); setModal('delete'); };
  const closeModal = () => { setModal(null); setCurrent(null); setForm({}); setError(null); };

  const submitForm = async () => {
    const isEdit = !!current;
    const url = isEdit ? `${API_BASE}/api/nexus-game/admin/content/research/${current.contentId}` : `${API_BASE}/api/nexus-game/admin/content/research`;
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
        ? `${API_BASE}/api/nexus-game/admin/content/research/${encodeURIComponent(current.contentId)}`
        : `${API_BASE}/api/nexus-game/admin/content/research/by-id/${current.id}`;
      const res = await fetch(target, { method: 'DELETE', credentials: 'same-origin' });
      if (!res.ok) throw new Error(await res.text());
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

  const invalidItems = items.filter((item) => !item.contentId && hasMeaningfulResearchData(item));

  return (
    <AdminShell title="Recherches — Arbre Nexus v2.0" description="11 branches × 7 tiers. CRUD + upload assets + dépendances (prerequisitesJSON).">
      <button onClick={() => window.location.href = '/admin/nexus/mmo'} style={{ marginBottom: 16 }}>← Retour Nexus MMO</button>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <h2>Recherches ({items.length})</h2>
        <button onClick={openCreate} style={{ background: '#8b5cf6', color: 'white', padding: '8px 16px', borderRadius: 6, border: 'none' }}>+ Créer Nœud</button>
      </div>

      {invalidItems.length > 0 && <p style={{ marginBottom: 12, color: '#fca5a5', fontSize: 13 }}>{invalidItems.length} entrée(s) invalide(s) sans <code>contentId</code> détectée(s).</p>}

      {loading && <p>Chargement...</p>}
      {error && <p style={{color:'red'}}>{error}</p>}

      <CatalogTable>
        <thead><tr><th style={{ width: 280 }}>Fiche</th><th style={{ width: 320 }}>Description</th><th style={{ width: 190 }}>Coûts</th><th style={{ width: 300 }}>Effet principal</th><th style={{ width: 280 }}>Niveaux</th><th style={{ width: 210 }}>Médias</th><th style={{ width: 130 }}>Actions</th></tr></thead>
        <tbody>
          {items.map(r => {
            const assets = collectAssets(r);
            return (
              <tr key={r.contentId || `research-${r.id}`} style={{ borderTop: '1px solid #334155', verticalAlign: 'top' }}>
                <td>
                  <CatalogIdentity
                    translations={translations}
                    titleKey={r.nameKey}
                    contentId={r.contentId}
                    badges={[r.branch || '-', `tier ${r.tier || '-'}`, r.rarity || '-']}
                  />
                </td>
                <td><DescriptionSummary translations={translations} descriptionKey={r.descriptionKey} flavorTextKey={r.flavorTextKey} /></td>
                <td><CostSummary item={r as any} kind="research" /></td>
                <td><EffectsPreview effects={r.effects || r.effectsJSON} /></td>
                <td><LevelDescriptionsPreview keys={r.levelDescriptionKeys} translations={translations} /></td>
                <td><AssetPreview assets={assets} fallbackLabel={r.contentId || `research-${r.id}`} /></td>
                <td><button onClick={() => openEdit(r)} style={{ fontSize: 12, marginRight: 6 }}>Éditer</button><button onClick={() => openDelete(r)} style={{ fontSize: 12, color: '#f87171' }}>Suppr</button></td>
              </tr>
            );
          })}
        </tbody>
      </CatalogTable>

      {modal && (
        <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.7)', display: 'flex', alignItems: 'flex-start', justifyContent: 'center', zIndex: 100, overflowY: 'auto', padding: '24px 12px' }}>
          <div className="panel" style={{ width: 620, maxWidth: 'calc(100vw - 24px)', maxHeight: 'calc(100vh - 48px)', overflowY: 'auto', padding: 24, position: 'relative' }}>
            <button onClick={closeModal} style={{ position: 'absolute', top: 8, right: 12, fontSize: 24, background: 'none', border: 'none' }}>×</button>
            <h3>{modal === 'create' ? 'Créer Nœud Recherche' : modal === 'edit' ? 'Éditer Nœud' : 'Supprimer Nœud'}</h3>

            {modal === 'delete' && current ? (
              <><p>Supprimer <strong>{current.contentId || `#${current.id}`}</strong> ?</p><button onClick={doDelete} style={{ background: '#ef4444', color: 'white', padding: '8px 16px' }}>Confirmer</button> <button onClick={closeModal}>Annuler</button></>
            ) : (
              <>
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))', gap: 8 }}>
                  <input placeholder="contentId (ex: research_efficient_storage ou research_solar_stabilization)" value={form.contentId || ''} onChange={e=>setForm({...form, contentId:e.target.value})} />
                  <input placeholder="nameKey" value={form.nameKey || ''} onChange={e=>setForm({...form, nameKey:e.target.value})} />
                  <input placeholder="descriptionKey" value={form.descriptionKey || ''} onChange={e=>setForm({...form, descriptionKey:e.target.value})} />
                  <input placeholder="flavorTextKey (optionnel)" value={form.flavorTextKey || ''} onChange={e=>setForm({...form, flavorTextKey:e.target.value})} />
                  <input placeholder="branch (economy, military...)" value={form.branch || ''} onChange={e=>setForm({...form, branch:e.target.value})} />
                  <input type="number" placeholder="tier (1-7)" value={form.tier || 1} onChange={e=>setForm({...form, tier:parseInt(e.target.value)})} />
                  <select value={form.rarity || 'common'} onChange={e=>setForm({...form, rarity:e.target.value})}>
                    <option>common</option><option>uncommon</option><option>rare</option><option>epic</option><option>legendary</option><option>nexus</option>
                  </select>
                  <input type="number" placeholder="costBaseCredits" value={form.costBaseCredits || ''} onChange={e=>setForm({...form, costBaseCredits:parseInt(e.target.value)})} />
                </div>

                <div style={{ marginTop: 12 }}>
                  <label style={{ fontSize: 12, color: '#94a3b8' }}>Descriptions par niveau (clés i18n JSON, ex: niveau 1 → nexus_game.research.xxx.level_1.description)</label>
                  <textarea placeholder='{"1":"nexus_game.research.example.level_1.description","2":"nexus_game.research.example.level_2.description"}' value={stringifyRecord(form.levelDescriptionKeys)} onChange={e=>setForm({...form, levelDescriptionKeys:e.target.value})} style={{ width: '100%', height: 110, marginTop: 6, fontFamily: 'monospace', fontSize: 12 }} />
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
