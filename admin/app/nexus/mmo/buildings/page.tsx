"use client";

import { useState, useEffect } from "react";
import { AdminShell } from "../../../components/AdminShell";
import { AssetPreview, CatalogIdentity, CatalogTable, CostSummary, DescriptionSummary, EffectsPreview, LevelDescriptionsPreview, TranslationMap, fetchCatalogTranslations } from "../contentDisplay";

const API_BASE = (process.env.NEXT_PUBLIC_NEXUS_API_BASE || "").replace(/\/$/, "");

interface Building {
  id: number;
  contentId: string;
  domain?: string;
  type?: string;
  nameKey?: string;
  descriptionKey?: string;
  flavorTextKey?: string;
  levelDescriptionKeys?: Record<string, string>;
  rarity: string;
  maxLevel: number;
  slotsMax?: number;
  availableAtSpawn?: boolean;
  unlockEra?: string;
  unlockType?: string;
  unlockMessage?: string;
  nonConstructible?: boolean;
  nexusLevelRequired?: number;
  requiredBuildings?: string;
  requiredResearch?: string;
  costBaseCredits?: number;
  costBaseMetal?: number;
  costBaseData?: number;
  durationBaseSeconds?: number;
  durationMultiplier?: number;
  milestoneReduction?: number;
  storageResource?: string;
  storageCapBase?: number;
  storageGrowth?: number;
  overflowBehavior?: string;
  overflowDecayPercentPerHour?: number;
  productionBasePerHour?: number;
  productionGrowth?: number;
  harvestRecommendedIntervalSeconds?: number;
  workersMin?: number;
  workersMax?: number;
  assetId?: string;
  assetsByTier?: Record<string, string>;
  effects?: string;
  effectsJSON?: string;
  synergies?: string;
  synergiesJSON?: string;
  risks?: string;
  risksJSON?: string;
  aiActions?: string;
  aiActionsJSON?: string;
  aiAgentSlots?: number;
  aiAgentTypes?: string[];
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

function parseList(value: unknown): any[] {
  if (!value) return [];
  if (Array.isArray(value)) return value;
  if (typeof value === "string") {
    try {
      const parsed = JSON.parse(value);
      return Array.isArray(parsed) ? parsed : [];
    } catch {
      return [];
    }
  }
  return [];
}

function formatSeconds(seconds?: number) {
  if (!seconds || !Number.isFinite(seconds)) return "-";
  if (seconds < 3600) return `${Math.round(seconds / 60)}min`;
  if (seconds < 86400) return `${Math.round(seconds / 3600)}h`;
  return `${Math.round(seconds / 86400)}j`;
}

function TextInput({ label, value, onChange, type = "text" }: { label: string; value: any; onChange: (value: any) => void; type?: string }) {
  return (
    <label style={{ display: "grid", gap: 4, fontSize: 12, color: "#94a3b8" }}>
      {label}
      <input
        type={type}
        value={value ?? ""}
        onChange={(e) => onChange(type === "number" ? Number(e.target.value || 0) : e.target.value)}
      />
    </label>
  );
}

function JsonTextarea({ label, value, onChange, placeholder }: { label: string; value: string; onChange: (value: string) => void; placeholder?: string }) {
  return (
    <label style={{ display: "grid", gap: 4, fontSize: 12, color: "#94a3b8", marginTop: 12 }}>
      {label}
      <textarea
        placeholder={placeholder}
        value={value || ""}
        onChange={(e) => onChange(e.target.value)}
        style={{ width: "100%", height: 92, fontFamily: "monospace", fontSize: 12 }}
      />
    </label>
  );
}

function MMODesignSummary({ building }: { building: Building }) {
  const actions = parseList(building.aiActions || building.aiActionsJSON);
  const synergies = parseList(building.synergies || building.synergiesJSON);
  const risks = parseList(building.risks || building.risksJSON);
  const storageLabel = building.overflowBehavior === "realtime"
    ? "Flux temps réel"
    : building.storageCapBase
      ? `${building.storageCapBase.toLocaleString("fr-FR")} ${building.storageResource || ""}`
      : "Pas de stock";
  return (
    <div style={{ display: "grid", gap: 5, fontSize: 11, color: "#cbd5e1" }}>
      <div><strong style={{ color: "#67e8f9" }}>Slots</strong> {building.slotsMax || 1}/cité</div>
      <div><strong style={{ color: "#67e8f9" }}>Stock</strong> {storageLabel} · {building.overflowBehavior || "none"}</div>
      <div><strong style={{ color: "#67e8f9" }}>Prod</strong> {building.productionBasePerHour || 0}/h ×{building.productionGrowth || 1}</div>
      <div><strong style={{ color: "#67e8f9" }}>Durée</strong> {formatSeconds(building.durationBaseSeconds)} ×{building.durationMultiplier || 1.28}</div>
      <div style={{ color: "#94a3b8" }}>{actions.length} action(s) IA · {synergies.length} synergie(s) · {risks.length} risque(s)</div>
    </div>
  );
}

function UnlockSummary({ building }: { building: Building }) {
  const requirements = parseList(building.requiredBuildings);
  const research = parseList(building.requiredResearch);
  return (
    <div style={{ display: "grid", gap: 5, fontSize: 11, color: "#cbd5e1" }}>
      <div>
        <span style={{ color: "#f0abfc", fontWeight: 700 }}>Ère {building.unlockEra || "-"}</span>{" "}
        <span>{building.unlockType || "AND"}</span>
      </div>
      <div>{building.availableAtSpawn ? "Disponible spawn" : `${requirements.length} prérequis bâtiment(s)`}</div>
      {research.length > 0 && <div>{research.length} recherche(s) requise(s)</div>}
      {building.nonConstructible && <div style={{ color: "#fca5a5" }}>Non constructible</div>}
      {building.unlockMessage && <div style={{ color: "#94a3b8", lineHeight: 1.35 }}>{building.unlockMessage}</div>}
    </div>
  );
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
    setForm({
      contentId: '',
      domain: 'building',
      type: 'production',
      nameKey: '',
      descriptionKey: '',
      flavorTextKey: '',
      levelDescriptionKeys: '{}',
      rarity: 'common',
      maxLevel: 30,
      slotsMax: 1,
      availableAtSpawn: false,
      unlockEra: '1',
      unlockType: 'AND',
      unlockMessage: '',
      nonConstructible: false,
      nexusLevelRequired: 1,
      requiredBuildings: '[]',
      requiredResearch: '[]',
      costBaseCredits: 100,
      costBaseMetal: 200,
      costBaseData: 0,
      durationBaseSeconds: 600,
      durationMultiplier: 1.28,
      milestoneReduction: 0.15,
      storageResource: '',
      storageCapBase: 0,
      storageGrowth: 1.1,
      overflowBehavior: 'none',
      overflowDecayPercentPerHour: 0,
      productionBasePerHour: 0,
      productionGrowth: 1.13,
      harvestRecommendedIntervalSeconds: 0,
      workersMin: 1,
      workersMax: 4,
      aiAgentSlots: 0,
      isPublished: true,
      effectsJSON: '[]',
      synergiesJSON: '[]',
      risksJSON: '[]',
      aiActionsJSON: '[]',
    });
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
      const payload = {
        ...form,
        levelDescriptionKeys: parseRecord(form.levelDescriptionKeys),
        effects: form.effects || form.effectsJSON || "[]",
        synergies: form.synergies || form.synergiesJSON || "[]",
        risks: form.risks || form.risksJSON || "[]",
        aiActions: form.aiActions || form.aiActionsJSON || "[]",
      };
      delete payload.effectsJSON;
      delete payload.synergiesJSON;
      delete payload.risksJSON;
      delete payload.aiActionsJSON;
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
  const groupedItems = items.reduce<Record<string, Building[]>>((acc, item) => {
    const category = item.type || "sans_categorie";
    acc[category] = acc[category] || [];
    acc[category].push(item);
    return acc;
  }, {});
  const totalSlots = items.reduce((sum, item) => sum + (item.slotsMax || 1), 0);
  const producerCount = items.filter((item) => (item.productionBasePerHour || 0) > 0 || item.overflowBehavior === "realtime").length;

  return (
    <AdminShell title="Tour de contrôle — Bâtiments Nexus v2.x" description="Catalogue MMO temps réel : slots cité, stockage, production, déblocages, risques, synergies et actions IA. Les médias existants restent servis via /nexus-assets.">
      <button onClick={() => window.location.href = '/admin/nexus/mmo'} style={{ marginBottom: 16 }}>← Retour Nexus MMO</button>

      <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(180px, 1fr))", gap: 10, marginBottom: 16 }}>
        {[
          ["Bâtiments", items.length],
          ["Publiés", items.filter(i => i.isPublished).length],
          ["Slots cité", totalSlots],
          ["Producteurs", producerCount],
        ].map(([label, value]) => (
          <div key={label} style={{ border: "1px solid #155e75", borderRadius: 8, padding: 12, background: "linear-gradient(135deg, rgba(8,47,73,.92), rgba(15,23,42,.96))", boxShadow: "0 0 22px rgba(34,211,238,.10)" }}>
            <div style={{ color: "#67e8f9", fontSize: 11, textTransform: "uppercase" }}>{label}</div>
            <div style={{ color: "#f8fafc", fontSize: 24, fontWeight: 800 }}>{value}</div>
          </div>
        ))}
      </div>

      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <h2>Bâtiments par catégorie</h2>
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
            <th style={{ width: 260 }}>Déblocage</th>
            <th style={{ width: 250 }}>MMO</th>
            <th style={{ width: 190 }}>Coûts</th>
            <th style={{ width: 280 }}>Apports</th>
            <th style={{ width: 280 }}>Niveaux</th>
            <th style={{ width: 220 }}>Médias</th>
            <th style={{ width: 130 }}>Actions</th>
          </tr>
        </thead>
        <tbody>
          {Object.entries(groupedItems).flatMap(([category, categoryItems]) => [
            <tr key={`category-${category}`}>
              <td colSpan={9} style={{ background: "#082f49", color: "#67e8f9", fontWeight: 800, letterSpacing: 0, textTransform: "uppercase" }}>
                {category} · {categoryItems.length}
              </td>
            </tr>,
            ...categoryItems.map((b) => {
            const assets = collectAssets(b);
            return (
              <tr key={b.contentId || `building-${b.id}`} style={{ borderTop: '1px solid #334155', verticalAlign: 'top' }}>
                <td>
                  <CatalogIdentity
                    translations={translations}
                    titleKey={b.nameKey}
                    contentId={b.contentId}
                    badges={[b.rarity || '-', b.type || "-", `lvl ${b.maxLevel || '-'}`, `${b.aiAgentSlots ?? 0} agent(s)`]}
                  />
                </td>
                <td><DescriptionSummary translations={translations} descriptionKey={b.descriptionKey} flavorTextKey={b.flavorTextKey} /></td>
                <td><UnlockSummary building={b} /></td>
                <td><MMODesignSummary building={b} /></td>
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
            }),
          ])}
        </tbody>
      </CatalogTable>

      {modal && (
        <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.7)', display: 'flex', alignItems: 'flex-start', justifyContent: 'center', zIndex: 100, overflowY: 'auto', padding: '24px 12px' }}>
          <div className="panel" style={{ width: 980, maxWidth: 'calc(100vw - 24px)', maxHeight: 'calc(100vh - 48px)', overflowY: 'auto', padding: 24, position: 'relative' }}>
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
                  <input placeholder="Catégorie/type" value={form.type || ''} onChange={e => setForm({ ...form, type: e.target.value })} />
                  <input placeholder="nameKey" value={form.nameKey || ''} onChange={e => setForm({ ...form, nameKey: e.target.value })} />
                  <input placeholder="descriptionKey" value={form.descriptionKey || ''} onChange={e => setForm({ ...form, descriptionKey: e.target.value })} />
                  <input placeholder="flavorTextKey (optionnel)" value={form.flavorTextKey || ''} onChange={e => setForm({ ...form, flavorTextKey: e.target.value })} />
                  <select value={form.rarity || 'common'} onChange={e => setForm({ ...form, rarity: e.target.value })}>
                    <option value="common">common</option><option value="uncommon">uncommon</option><option value="rare">rare</option>
                    <option value="epic">epic</option><option value="legendary">legendary</option><option value="nexus">nexus</option>
                  </select>
                  <input type="number" placeholder="maxLevel" value={form.maxLevel || 30} onChange={e => setForm({ ...form, maxLevel: parseInt(e.target.value) })} />
                  <input type="number" placeholder="aiAgentSlots" value={form.aiAgentSlots || 0} onChange={e => setForm({ ...form, aiAgentSlots: parseInt(e.target.value) })} />
                  <TextInput label="Slots max / cité" type="number" value={form.slotsMax ?? 1} onChange={(v) => setForm({ ...form, slotsMax: v })} />
                  <TextInput label="Niveau Nexus requis" type="number" value={form.nexusLevelRequired ?? 1} onChange={(v) => setForm({ ...form, nexusLevelRequired: v })} />
                  <TextInput label="Durée L1 secondes" type="number" value={form.durationBaseSeconds ?? 600} onChange={(v) => setForm({ ...form, durationBaseSeconds: v })} />
                  <TextInput label="Multiplicateur durée" type="number" value={form.durationMultiplier ?? 1.28} onChange={(v) => setForm({ ...form, durationMultiplier: v })} />
                  <TextInput label="Réduction palier" type="number" value={form.milestoneReduction ?? 0.15} onChange={(v) => setForm({ ...form, milestoneReduction: v })} />
                </div>

                <div style={{ marginTop: 14, padding: 12, border: "1px solid #155e75", borderRadius: 8, background: "rgba(8,47,73,.32)" }}>
                  <h4 style={{ margin: "0 0 10px", color: "#67e8f9" }}>Déblocage MMO</h4>
                  <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(180px, 1fr))", gap: 10 }}>
                    <TextInput label="Ère" value={form.unlockEra || "0"} onChange={(v) => setForm({ ...form, unlockEra: v })} />
                    <label style={{ display: "grid", gap: 4, fontSize: 12, color: "#94a3b8" }}>
                      Type unlock
                      <select value={form.unlockType || "AND"} onChange={e => setForm({ ...form, unlockType: e.target.value })}>
                        <option value="AND">AND</option>
                        <option value="OR">OR</option>
                      </select>
                    </label>
                    <label style={{ display: "flex", alignItems: "center", gap: 8, color: "#cbd5e1", fontSize: 12 }}>
                      <input type="checkbox" checked={Boolean(form.availableAtSpawn)} onChange={e => setForm({ ...form, availableAtSpawn: e.target.checked })} />
                      Disponible au spawn
                    </label>
                    <label style={{ display: "flex", alignItems: "center", gap: 8, color: "#cbd5e1", fontSize: 12 }}>
                      <input type="checkbox" checked={Boolean(form.nonConstructible)} onChange={e => setForm({ ...form, nonConstructible: e.target.checked })} />
                      Non constructible
                    </label>
                  </div>
                  <JsonTextarea label="Required buildings JSON" value={form.requiredBuildings || "[]"} onChange={(v) => setForm({ ...form, requiredBuildings: v })} placeholder='[{"contentId":"building_data_bank","level":3}]' />
                  <JsonTextarea label="Required research JSON" value={form.requiredResearch || "[]"} onChange={(v) => setForm({ ...form, requiredResearch: v })} placeholder='["research_drone_assembly"]' />
                  <textarea placeholder="Message de déblocage" value={form.unlockMessage || ""} onChange={e => setForm({ ...form, unlockMessage: e.target.value })} style={{ width: "100%", height: 64, marginTop: 12 }} />
                </div>

                <div style={{ marginTop: 14, padding: 12, border: "1px solid #164e63", borderRadius: 8, background: "rgba(6,78,59,.24)" }}>
                  <h4 style={{ margin: "0 0 10px", color: "#5eead4" }}>Stockage & production</h4>
                  <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(180px, 1fr))", gap: 10 }}>
                    <TextInput label="Ressource stockée" value={form.storageResource || ""} onChange={(v) => setForm({ ...form, storageResource: v })} />
                    <TextInput label="Cap stockage L1" type="number" value={form.storageCapBase ?? 0} onChange={(v) => setForm({ ...form, storageCapBase: v })} />
                    <TextInput label="Croissance stockage" type="number" value={form.storageGrowth ?? 1.1} onChange={(v) => setForm({ ...form, storageGrowth: v })} />
                    <TextInput label="Overflow" value={form.overflowBehavior || "none"} onChange={(v) => setForm({ ...form, overflowBehavior: v })} />
                    <TextInput label="Decay %/h" type="number" value={form.overflowDecayPercentPerHour ?? 0} onChange={(v) => setForm({ ...form, overflowDecayPercentPerHour: v })} />
                    <TextInput label="Production L1 /h" type="number" value={form.productionBasePerHour ?? 0} onChange={(v) => setForm({ ...form, productionBasePerHour: v })} />
                    <TextInput label="Croissance production" type="number" value={form.productionGrowth ?? 1.13} onChange={(v) => setForm({ ...form, productionGrowth: v })} />
                    <TextInput label="Récolte recommandée sec" type="number" value={form.harvestRecommendedIntervalSeconds ?? 0} onChange={(v) => setForm({ ...form, harvestRecommendedIntervalSeconds: v })} />
                  </div>
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
                <JsonTextarea label="Synergies JSON" value={form.synergies || form.synergiesJSON || "[]"} onChange={(v) => setForm({ ...form, synergies: v, synergiesJSON: v })} />
                <JsonTextarea label="Risques & malus JSON" value={form.risks || form.risksJSON || "[]"} onChange={(v) => setForm({ ...form, risks: v, risksJSON: v })} />
                <JsonTextarea label="Actions IA JSON" value={form.aiActions || form.aiActionsJSON || "[]"} onChange={(v) => setForm({ ...form, aiActions: v, aiActionsJSON: v })} />

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
