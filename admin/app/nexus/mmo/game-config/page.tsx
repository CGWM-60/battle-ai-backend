"use client";

import { useEffect, useMemo, useState } from "react";
import { AdminShell } from "../../../components/AdminShell";

const API_BASE = (process.env.NEXT_PUBLIC_NEXUS_API_BASE || "").replace(/\/$/, "");

type GameConfig = {
  id?: number;
  key?: string;
  populationHabitatBase: number;
  populationHabitatPerLevel: number;
  foodPerPopulationPerHour: number;
  energyBaseConsumptionPerHour: number;
  populationPerEnergy: number;
  buildingEnergySurchargeExponent: number;
  buildingEnergyHighLevelThreshold: number;
  buildingEnergyHighLevelMultiplier: number;
  industryEnergyPerLevel: number;
  dataEnergyPerLevel: number;
  solarEnergyReliefPerLevel: number;
  habitatFarmCoverageDivisor: number;
  habitatFarmEnergyPenaltyDivisor: number;
  energyRatioCriticalThreshold: number;
  energyRatioTensionThreshold: number;
  energyRatioSurplusThreshold: number;
  energyRatioDominanceThreshold: number;
  resourceMultiplierCritical: number;
  resourceMultiplierTension: number;
  resourceMultiplierBalanced: number;
  resourceMultiplierSurplus: number;
  resourceMultiplierDominance: number;
  defaultBuildingDurationMultiplier: number;
  defaultBuildingMilestoneReduction: number;
  defaultBuildingHighLevelMilestoneReduction: number;
  updatedAt?: string;
  updatedBy?: string;
};

const sections: { title: string; fields: { key: keyof GameConfig; label: string; step?: string; help: string }[] }[] = [
  {
    title: "Population",
    fields: [
      { key: "populationHabitatBase", label: "Habitants habitation niv. 1", help: "Capacite de base d'une habitation modulaire terminee." },
      { key: "populationHabitatPerLevel", label: "Habitants par niveau", help: "Ajout par niveau apres le niveau 1." },
      { key: "foodPerPopulationPerHour", label: "Nourriture par habitant / h", step: "0.01", help: "Consommation alimentaire horaire par habitant." },
    ],
  },
  {
    title: "Energie",
    fields: [
      { key: "energyBaseConsumptionPerHour", label: "Consommation ville / h", help: "Base fixe avant population et batiments." },
      { key: "populationPerEnergy", label: "Population par 1 energie/h", help: "Plus la valeur est basse, plus la population coute cher en energie." },
      { key: "buildingEnergySurchargeExponent", label: "Exponent surcharge batiments", step: "0.01", help: "Calcule floor(level^exponent) par batiment termine." },
      { key: "buildingEnergyHighLevelThreshold", label: "Seuil haut niveau", help: "Au-dessus de ce niveau, le multiplicateur crise s'applique." },
      { key: "buildingEnergyHighLevelMultiplier", label: "Multiplicateur haut niveau", step: "0.01", help: "Surcharge additionnelle des batiments tres hauts niveaux." },
      { key: "industryEnergyPerLevel", label: "Industrie energie / niveau", help: "Pression des mines, usines, raffineries, logistique." },
      { key: "dataEnergyPerLevel", label: "Data/IA energie / niveau", help: "Pression des labs, data banks, IA, relais monde." },
      { key: "solarEnergyReliefPerLevel", label: "Reduction solaire / niveau", help: "Reduction de pression par niveau de centrale solaire." },
      { key: "habitatFarmCoverageDivisor", label: "Couverture ferme/habitat", help: "Condition: fermes * valeur < habitats declenche une surcharge." },
      { key: "habitatFarmEnergyPenaltyDivisor", label: "Penalite habitat sans ferme", help: "Surcharge: habitatTotal / valeur." },
    ],
  },
  {
    title: "Ratio energie",
    fields: [
      { key: "energyRatioCriticalThreshold", label: "Seuil critique", step: "0.01", help: "Sous ce ratio, malus critique." },
      { key: "energyRatioTensionThreshold", label: "Seuil tension", step: "0.01", help: "Sous ce ratio, malus leger." },
      { key: "energyRatioSurplusThreshold", label: "Seuil surplus", step: "0.01", help: "Au-dessus, bonus production." },
      { key: "energyRatioDominanceThreshold", label: "Seuil dominance", step: "0.01", help: "Au-dessus, meilleur bonus production." },
      { key: "resourceMultiplierCritical", label: "Multiplicateur critique", step: "0.01", help: "Production non-energie quand le ratio est critique." },
      { key: "resourceMultiplierTension", label: "Multiplicateur tension", step: "0.01", help: "Production non-energie en tension." },
      { key: "resourceMultiplierBalanced", label: "Multiplicateur equilibre", step: "0.01", help: "Production nominale." },
      { key: "resourceMultiplierSurplus", label: "Multiplicateur surplus", step: "0.01", help: "Bonus quand l'energie est confortable." },
      { key: "resourceMultiplierDominance", label: "Multiplicateur dominance", step: "0.01", help: "Bonus maximum quand la ville surproduit." },
    ],
  },
  {
    title: "Construction",
    fields: [
      { key: "defaultBuildingDurationMultiplier", label: "Multiplicateur duree par niveau", step: "0.01", help: "Fallback global si un batiment n'a pas sa propre valeur." },
      { key: "defaultBuildingMilestoneReduction", label: "Reduction paliers", step: "0.01", help: "Reduction aux niveaux 5, 10, 15, 20, 25, 30." },
      { key: "defaultBuildingHighLevelMilestoneReduction", label: "Reduction haut niveau", step: "0.01", help: "Reserve pour equilibrage haut niveau." },
    ],
  },
];

function NumberInput({ cfg, field, onChange }: { cfg: GameConfig; field: { key: keyof GameConfig; label: string; step?: string; help: string }; onChange: (key: keyof GameConfig, value: number) => void }) {
  return (
    <label style={{ display: "grid", gap: 7 }}>
      <span style={{ color: "#c4b5fd", fontSize: 12, fontWeight: 700 }}>{field.label}</span>
      <input
        type="number"
        step={field.step || "1"}
        value={Number(cfg[field.key] ?? 0)}
        onChange={(event) => onChange(field.key, Number(event.target.value))}
        style={{ background: "#070b18", color: "#e0f2fe", border: "1px solid rgba(34,211,238,.35)", borderRadius: 6, padding: "10px 11px" }}
      />
      <span style={{ color: "#7dd3fc", fontSize: 11, lineHeight: 1.35 }}>{field.help}</span>
    </label>
  );
}

export default function GameConfigPage() {
  const [config, setConfig] = useState<GameConfig | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");

  async function loadConfig() {
    setLoading(true);
    setError("");
    try {
      const res = await fetch(`${API_BASE}/api/nexus-game/admin/game-config`, { credentials: "same-origin" });
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      setConfig(data.config);
    } catch (err: any) {
      setError(err.message || "Erreur chargement reglages");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { loadConfig(); }, []);

  const preview = useMemo(() => {
    if (!config) return null;
    const capacity = (level: number) => config.populationHabitatBase + config.populationHabitatPerLevel * (level - 1);
    const population = 500;
    return {
      levels: [1, 2, 3, 10, 20, 30].map((level) => ({ level, capacity: capacity(level) })),
      food: population * config.foodPerPopulationPerHour,
      energy: config.energyBaseConsumptionPerHour + Math.floor(population / Math.max(1, config.populationPerEnergy)),
    };
  }, [config]);

  function updateField(key: keyof GameConfig, value: number) {
    if (!config) return;
    setConfig({ ...config, [key]: value });
  }

  async function save() {
    if (!config) return;
    setSaving(true);
    setError("");
    setMessage("");
    try {
      const res = await fetch(`${API_BASE}/api/nexus-game/admin/game-config`, {
        method: "PUT",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(config),
      });
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      setConfig(data.config);
      setMessage("Reglages sauvegardes. Les prochains recalculs ressources utiliseront ces valeurs.");
    } catch (err: any) {
      setError(err.message || "Erreur sauvegarde");
    } finally {
      setSaving(false);
    }
  }

  return (
    <AdminShell title="Reglages systeme Nexus" description="Parametres live des calculs serveur: population, nourriture, energie, production et durees.">
      <button onClick={() => window.location.href = "/admin/nexus/mmo"} style={{ marginBottom: 16 }}>Retour Nexus MMO</button>
      {loading && <p>Chargement...</p>}
      {error && <div style={{ color: "#fecaca", marginBottom: 12 }}>{error}</div>}
      {message && <div style={{ color: "#86efac", marginBottom: 12 }}>{message}</div>}
      {config && (
        <div style={{ display: "grid", gridTemplateColumns: "minmax(0, 1fr) 320px", gap: 18, alignItems: "start" }}>
          <div style={{ display: "grid", gap: 16 }}>
            {sections.map((section) => (
              <section key={section.title} style={{ border: "1px solid rgba(125,211,252,.22)", background: "linear-gradient(135deg, rgba(15,23,42,.96), rgba(30,41,59,.82))", borderRadius: 8, padding: 16 }}>
                <h2 style={{ margin: "0 0 14px", color: "#22d3ee", fontSize: 18 }}>{section.title}</h2>
                <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(220px, 1fr))", gap: 14 }}>
                  {section.fields.map((field) => <NumberInput key={String(field.key)} cfg={config} field={field} onChange={updateField} />)}
                </div>
              </section>
            ))}
            <button onClick={save} disabled={saving} style={{ justifySelf: "start", padding: "11px 18px", background: "#0891b2", color: "white", border: 0, borderRadius: 6, fontWeight: 800 }}>
              {saving ? "Sauvegarde..." : "Sauvegarder les reglages"}
            </button>
          </div>
          <aside style={{ position: "sticky", top: 18, border: "1px solid rgba(34,211,238,.3)", background: "#020617", borderRadius: 8, padding: 16 }}>
            <h2 style={{ margin: "0 0 12px", color: "#67e8f9", fontSize: 17 }}>Apercu live</h2>
            <div style={{ display: "grid", gap: 8, fontSize: 13, color: "#cbd5e1" }}>
              {preview?.levels.map((row) => (
                <div key={row.level} style={{ display: "flex", justifyContent: "space-between", borderBottom: "1px solid rgba(148,163,184,.18)", paddingBottom: 6 }}>
                  <span>Habitation niv. {row.level}</span>
                  <strong style={{ color: "#f0abfc" }}>{row.capacity.toLocaleString("fr-FR")} habitants</strong>
                </div>
              ))}
              <div style={{ marginTop: 10, color: "#93c5fd" }}>Exemple 500 habitants</div>
              <div>Nourriture: <strong>{preview?.food.toFixed(1)}/h</strong></div>
              <div>Energie ville: <strong>{preview?.energy.toFixed(1)}/h</strong></div>
              <div style={{ color: "#64748b", fontSize: 11, lineHeight: 1.45, marginTop: 10 }}>
                Les valeurs sont appliquees au prochain appel de synchronisation ressources ou tick monde. Flutter ne recalcule pas ces formules localement.
              </div>
            </div>
          </aside>
        </div>
      )}
    </AdminShell>
  );
}

