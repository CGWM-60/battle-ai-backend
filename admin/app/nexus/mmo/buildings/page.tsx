"use client";

import { useState, useEffect } from "react";
import { AdminShell } from "../../../components/AdminShell";
import { AssetPreview, CatalogIdentity, CatalogTable, CostSummary, DescriptionSummary, EffectsPreview, LevelDescriptionsPreview, TranslationMap, fetchCatalogTranslations } from "../contentDisplay";

const API_BASE = (process.env.NEXT_PUBLIC_NEXUS_API_BASE || "").replace(/\/$/, "");

interface Building {
  id: number;
  contentId: string;
  nameKey?: string;
  descriptionKey?: string;
  flavorTextKey?: string;
  levelDescriptionKeys?: Record<string, string>;
  rarity: string;
  maxLevel: number;
  costBaseCredits?: number;
  costBaseMetal?: number;
  costBaseData?: number;
  durationBaseSeconds?: number;
  workersMin?: number;
  workersMax?: number;
  assetId?: string;
  assetsByTier?: Record<string, string>;
  effects?: string;
  effectsJSON?: string;
  aiAgentSlots?: number;
  isPublished?: boolean;
}

const ASSET_KEYS = ["main", "tier1", "tier2", "tier3", "tier4"] as const;

function buildAssetUrl(folder: string, fileName?: string) {
  if (!fileName) return null;
  if (/^https?:\/\//.test(fileName)) return fileName;
  if (fileName.startsWith("/")) return `${API_BASE}${fileName}`;
  return `${API_BASE}/nexus-assets/content/${folder}/${encodeURIComponent(fileName)}`;
}

function collectAssets(item: Pick<Building, "assetId" | "assetsByTier">) {
  const assetsByTier = item.assetsByTier && typeof item.assetsByTier === "object"
    ? item.assetsByTier
    : {};

  return ASSET_KEYS.map((key) => {
    const fileName = key === "main" ? (item.assetId || assetsByTier.main) : assetsByTier[key];
    return {
      key,
      fileName,
      url: buildAssetUrl("buildings", fileName),
    };
  }).filter((asset) => Boolean(asset.fileName));
}

function hasMeaningfulBuildingData(item: Building) {
  return Boolean(
    item.contentId ||
      item.nameKey ||
      item.descriptionKey ||
      item.assetId ||
      (item.assetsByTier && Object.keys(item.assetsByTier).length > 0),
  );
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

export default function BuildingsAdminPage() {
  const [items, setItems] = useState<Building[]>([]);
  const [translations, setTranslations] = useState<TranslationMap>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [modal, setModal] = useState<'create' | 'edit' | 'delete' | null>(null);
  const [current, setCurrent] = useState<Building | null>(null);
  const [form, setForm] = useState<any>({});
  const [uploading, setUploading] = useState<string | null>(null); // which tier is uploading

  const fetchItems = async () => {
    setLoading(true);
    setError(null);
    try {
      const [res, catalogTranslations] = await Promise.all([
        fetch(`${API_BASE}/api/nexus-game/admin/content/buildings`, { credentials: "same-origin" }),
        fetchCatalogTranslations(API_BASE, ["building"]),
      ]);
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      setTranslations(catalogTranslations);
      setItems(Array.isArray(data.buildings) ? data.buildings : []);
    } catch (e: any) {
      setError(e.message || "Erreur chargement bâtiments");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { fetchItems(); }, []);

  const openCreate = () => {
    setCurrent(null);
    setForm({ contentId: '', nameKey: '', descriptionKey: '', flavorTextKey: '', levelDescriptionKeys: '{}', rarity: 'common', maxLevel: 30, costBaseCredits: 100, costBaseMetal: 200, durationBaseSeconds: 60, aiAgentSlots: 0, isPublished: true, effectsJSON: '[]' });
    setModal('create');
  };

  const openEdit = (item: Building) => {
    setCurrent(item);
    setForm({ ...item, levelDescriptionKeys: stringifyRecord(item.levelDescriptionKeys) });
    setModal('edit');
  };

  const openDelete = (item: Building) => {
    setCurrent(item);
    setModal('delete');
  };

  const closeModal = () => {
    setModal(null);
    setCurrent(null);
    setForm({});
    setError(null);
  };

  const submitForm = async () => {
    const isEdit = !!current;
    const url = isEdit 
      ? `${API_BASE}/api/nexus-game/admin/content/buildings/${current.contentId}`
      : `${API_BASE}/api/nexus-game/admin/content/buildings`;
    const method = isEdit ? 'PUT' : 'POST';

    setLoading(true);
    try {
      const payload = { ...form, levelDescriptionKeys: parseRecord(form.levelDescriptionKeys) };
      const res = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        credentials: 'same-origin',
        body: JSON.stringify(payload),
      });
      if (!res.ok) throw new Error(await res.text());
      closeModal();
      await fetchItems();
    } catch (e: any) {
      setError(e.message || 'Erreur sauvegarde');
    } finally {
      setLoading(false);
    }
  };

  const doDelete = async () => {
    if (!current) return;
    setLoading(true);
    try {
      const target = current.contentId
        ? `${API_BASE}/api/nexus-game/admin/content/buildings/${encodeURIComponent(current.contentId)}`
        : `${API_BASE}/api/nexus-game/admin/content/buildings/by-id/${current.id}`;
      const res = await fetch(target, {
        method: 'DELETE',
        credentials: 'same-origin',
      });
      if (!res.ok) throw new Error(await res.text());
      closeModal();
      await fetchItems();
    } catch (e: any) {
      setError(e.message || 'Erreur suppression');
    } finally {
      setLoading(false);
    }
  };

  // Upload asset for a specific tier (or main)
  const uploadAsset = async (tier: string, file: File) => {
    if (!form.contentId && !current?.contentId) {
      setError("contentId requis avant d'uploader une image");
      return;
    }
    const cid = form.contentId || current!.contentId;
    const formData = new FormData();
    formData.append("file", file);
    formData.append("domain", "building");
    formData.append("contentId", cid);
    if (tier) formData.append("tier", tier);

    setUploading(tier || 'main');
    try {
      const res = await fetch(`${API_BASE}/api/nexus-game/admin/content/upload-asset`, {
        method: "POST",
        body: formData,
        credentials: "same-origin",
      });
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      const saved = data.url || data.urlHint || data.publicPath || data.savedAs || '';

      // Update form or current with the asset
      const key = tier ? `tier${tier}` : 'main';
      if (modal === 'create' || modal === 'edit') {
        const newAssets = { ...(form.assetsByTier || {}) };
        if (tier) {
          newAssets[key] = saved;
        } else {
          // main
          setForm((f: any) => ({ ...f, assetId: saved }));
        }
        setForm((f: any) => ({ ...f, assetsByTier: newAssets }));
      } else if (current) {
        // direct edit on list item
        const updated = { ...current };
        if (tier) {
          updated.assetsByTier = { ...(updated.assetsByTier || {}), [key]: saved };
        } else {
          updated.assetId = saved;
        }
        // persist the asset reference
        await fetch(`${API_BASE}/api/nexus-game/admin/content/buildings/${cid}`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(updated),
          credentials: 'same-origin',
        });
        await fetchItems();
      }
    } catch (e: any) {
      setError(e.message || 'Erreur upload image');
    } finally {
      setUploading(null);
    }
  };

  const invalidItems = items.filter((item) => !item.contentId && hasMeaningfulBuildingData(item));

  return (
    <AdminShell title="Bâtiments — Catalogue Nexus v2.0" description="CRUD complet des 20 bâtiments (niveaux 1-30). Upload images par tier (tier1-4). Données master pour construction / IA / équilibrage.">
      <button onClick={() => window.location.href = '/admin/nexus/mmo'} style={{ marginBottom: 16 }}>← Retour Nexus MMO</button>

      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <h2>Bâtiments ({items.length}) — {items.filter(i => i.isPublished).length} publiés</h2>
        <button onClick={openCreate} style={{ background: '#f59e0b', color: 'white', padding: '8px 16px', borderRadius: 6, border: 'none' }}>
          + Créer Bâtiment
        </button>
      </div>

      {invalidItems.length > 0 && (
        <p style={{ marginBottom: 12, color: '#fca5a5', fontSize: 13 }}>
          {invalidItems.length} entrée(s) invalide(s) sans <code>contentId</code> détectée(s). Elles restent supprimables via leur ID interne.
        </p>
      )}

      {loading && <p>Chargement...</p>}
      {error && <p style={{ color: 'red' }}>{error}</p>}

      <CatalogTable>
        <thead>
          <tr>
            <th style={{ width: 260 }}>Fiche</th>
            <th style={{ width: 310 }}>Description</th>
            <th style={{ width: 190 }}>Coûts</th>
            <th style={{ width: 280 }}>Apports</th>
            <th style={{ width: 280 }}>Niveaux</th>
            <th style={{ width: 220 }}>Médias</th>
            <th style={{ width: 130 }}>Actions</th>
          </tr>
        </thead>
        <tbody>
          {items.map((b) => {
            const assets = collectAssets(b);
            return (
              <tr key={b.contentId || `building-${b.id}`} style={{ borderTop: '1px solid #334155', verticalAlign: 'top' }}>
                <td>
                  <CatalogIdentity
                    translations={translations}
                    titleKey={b.nameKey}
                    contentId={b.contentId}
                    badges={[b.rarity || '-', `lvl ${b.maxLevel || '-'}`, `${b.aiAgentSlots ?? 0} agent(s)`]}
                  />
                </td>
                <td><DescriptionSummary translations={translations} descriptionKey={b.descriptionKey} flavorTextKey={b.flavorTextKey} /></td>
                <td><CostSummary item={b as any} kind="building" /></td>
                <td><EffectsPreview effects={b.effects || b.effectsJSON} /></td>
                <td><LevelDescriptionsPreview keys={b.levelDescriptionKeys} translations={translations} /></td>
                <td><AssetPreview assets={assets} fallbackLabel={b.contentId || `building-${b.id}`} /></td>
                <td>
                  <button onClick={() => openEdit(b)} style={{ marginRight: 6, fontSize: 12 }}>Éditer</button>
                  <button onClick={() => openDelete(b)} style={{ color: '#f87171', fontSize: 12 }}>Suppr</button>
                </td>
              </tr>
            );
          })}
        </tbody>
      </CatalogTable>

      {modal && (
        <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.7)', display: 'flex', alignItems: 'flex-start', justifyContent: 'center', zIndex: 100, overflowY: 'auto', padding: '24px 12px' }}>
          <div className="panel" style={{ width: 620, maxWidth: 'calc(100vw - 24px)', maxHeight: 'calc(100vh - 48px)', overflowY: 'auto', padding: 24, position: 'relative' }}>
            <button onClick={closeModal} style={{ position: 'absolute', top: 8, right: 12, fontSize: 24, background: 'none', border: 'none', cursor: 'pointer' }}>×</button>

            <h3>{modal === 'create' ? 'Créer Bâtiment' : modal === 'edit' ? 'Éditer Bâtiment' : 'Supprimer Bâtiment'}</h3>

            {modal === 'delete' && current ? (
              <>
                <p>Supprimer <strong>{current.contentId || `#${current.id}`}</strong> ? Cette action est définitive pour le catalogue master.</p>
                <div style={{ display: 'flex', gap: 8, marginTop: 16 }}>
                  <button onClick={doDelete} style={{ background: '#ef4444', color: 'white', padding: '8px 16px' }}>Confirmer Suppression</button>
                  <button onClick={closeModal}>Annuler</button>
                </div>
              </>
            ) : (
              <>
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))', gap: 12 }}>
                  <input placeholder="contentId (ex: building_modular_habitat)" value={form.contentId || ''} onChange={e => setForm({ ...form, contentId: e.target.value })} />
                  <input placeholder="nameKey" value={form.nameKey || ''} onChange={e => setForm({ ...form, nameKey: e.target.value })} />
                  <input placeholder="descriptionKey" value={form.descriptionKey || ''} onChange={e => setForm({ ...form, descriptionKey: e.target.value })} />
                  <input placeholder="flavorTextKey (optionnel)" value={form.flavorTextKey || ''} onChange={e => setForm({ ...form, flavorTextKey: e.target.value })} />
                  <select value={form.rarity || 'common'} onChange={e => setForm({ ...form, rarity: e.target.value })}>
                    <option value="common">common</option><option value="uncommon">uncommon</option><option value="rare">rare</option>
                    <option value="epic">epic</option><option value="legendary">legendary</option><option value="nexus">nexus</option>
                  </select>
                  <input type="number" placeholder="maxLevel" value={form.maxLevel || 30} onChange={e => setForm({ ...form, maxLevel: parseInt(e.target.value) })} />
                  <input type="number" placeholder="aiAgentSlots" value={form.aiAgentSlots || 0} onChange={e => setForm({ ...form, aiAgentSlots: parseInt(e.target.value) })} />
                </div>

                <div style={{ marginTop: 12 }}>
                  <label style={{ fontSize: 12, color: '#94a3b8' }}>Assets (upload par tier)</label>
                  <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginTop: 6 }}>
                    {['', '1', '2', '3', '4'].map(t => {
                      const key = t ? `tier${t}` : 'main';
                      const currentAsset = (form.assetsByTier && form.assetsByTier[key]) || (t==='' ? form.assetId : '');
                      return (
                        <div key={key} style={{ border: '1px solid #334155', padding: 6, borderRadius: 4, minWidth: 140, maxWidth: '100%' }}>
                          <div style={{ fontSize: 11 }}>{key}</div>
                          <input type="file" style={{ maxWidth: '100%' }} onChange={e => { if (e.target.files?.[0]) uploadAsset(t, e.target.files[0]); }} disabled={uploading === key} />
                          {currentAsset && <div style={{ fontSize: 10, color: '#64748b', marginTop: 2, wordBreak: 'break-all' }}>{currentAsset}</div>}
                          {uploading === key && <div style={{ fontSize: 11 }}>Upload...</div>}
                        </div>
                      );
                    })}
                  </div>
                </div>

                <div style={{ marginTop: 12 }}>
                  <label style={{ fontSize: 12, color: '#94a3b8' }}>Descriptions par niveau (clés i18n JSON, ex: niveau 1 → nexus_game.building.xxx.level_1.description)</label>
                  <textarea placeholder='{"1":"nexus_game.building.example.level_1.description","2":"nexus_game.building.example.level_2.description"}' value={stringifyRecord(form.levelDescriptionKeys)} onChange={e => setForm({ ...form, levelDescriptionKeys: e.target.value })} style={{ width: '100%', height: 110, marginTop: 6, fontFamily: 'monospace', fontSize: 12 }} />
                </div>

                <div style={{ marginTop: 12 }}>
                  <textarea placeholder="effectsJSON (array)" value={form.effectsJSON || '[]'} onChange={e => setForm({ ...form, effectsJSON: e.target.value })} style={{ width: '100%', height: 80, fontFamily: 'monospace', fontSize: 12 }} />
                </div>

                <div style={{ marginTop: 16, display: 'flex', gap: 8 }}>
                  <button onClick={submitForm} disabled={loading || !form.contentId} style={{ background: '#f59e0b', color: 'white', padding: '10px 20px' }}>
                    {modal === 'create' ? 'Créer' : 'Sauvegarder'}
                  </button>
                  <button onClick={closeModal}>Annuler</button>
                </div>
                <p style={{ fontSize: 11, color: '#64748b', marginTop: 8 }}>Les images sont uploadées vers le serveur Go et serviront via /nexus-assets/... dans le jeu.</p>
              </>
            )}
          </div>
        </div>
      )}
    </AdminShell>
  );
}
