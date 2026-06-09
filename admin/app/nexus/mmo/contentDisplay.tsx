"use client";

export type TranslationMap = Record<string, string>;

export async function fetchCatalogTranslations(apiBase: string, domains: string[]) {
  const merged: TranslationMap = {};
  await Promise.all(domains.map(async (domain) => {
    try {
      const res = await fetch(`${apiBase}/api/translations/domain/${encodeURIComponent(domain)}?locale=fr`, {
        credentials: "same-origin",
        cache: "no-store",
      });
      if (!res.ok) return;
      const data = await res.json();
      Object.assign(merged, data.translations || {});
    } catch {
      // Admin display fallback: keep keys visible if translations are unavailable.
    }
  }));
  return merged;
}

export function translated(translations: TranslationMap, key?: string) {
  if (!key) return "";
  return translations[key] || "";
}

export function TranslationCell({ translations, labelKey }: { translations: TranslationMap; labelKey?: string }) {
  const value = translated(translations, labelKey);
  return (
    <div style={{ minWidth: 220 }}>
      <div style={{ color: value ? "#e2e8f0" : "#fca5a5", fontSize: 13, lineHeight: 1.35 }}>
        {value || "Traduction manquante"}
      </div>
      <code style={{ display: "block", color: "#64748b", fontSize: 10, marginTop: 4, whiteSpace: "normal", wordBreak: "break-word" }}>
        {labelKey || "-"}
      </code>
    </div>
  );
}

export function CostSummary({ item, kind }: { item: Record<string, any>; kind: "building" | "unit" | "research" }) {
  const rows: string[] = [];
  if (kind === "unit") {
    if (item.healthBase != null) rows.push(`PV ${item.healthBase}`);
    if (item.attackBase != null) rows.push(`ATQ ${item.attackBase}`);
    if (item.defenseBase != null) rows.push(`DEF ${item.defenseBase}`);
    if (item.speedBase != null) rows.push(`VIT ${item.speedBase}`);
    if (item.trainingTimeBaseSeconds != null) rows.push(`Formation ${formatDuration(item.trainingTimeBaseSeconds)}`);
    if (item.upkeepBase != null) rows.push(`Upkeep ${item.upkeepBase}`);
  } else {
    if (item.costBaseCredits != null) rows.push(`${item.costBaseCredits} credits`);
    if (item.costBaseMetal != null) rows.push(`${item.costBaseMetal} metal`);
    if (item.costBaseData != null) rows.push(`${item.costBaseData} data`);
    if (item.durationBaseSeconds != null) rows.push(formatDuration(item.durationBaseSeconds));
  }
  return <CompactList rows={rows.length ? rows : ["-"]} />;
}

export function EffectsPreview({ effects }: { effects?: unknown }) {
  const parsed = parseEffects(effects);
  if (parsed.length === 0) return <span style={{ color: "#fca5a5", fontSize: 12 }}>Aucun apport</span>;
  return (
    <div style={{ display: "grid", gap: 4, minWidth: 220 }}>
      {parsed.slice(0, 6).map((effect, index) => (
        <div key={index} style={{ border: "1px solid #334155", borderRadius: 6, padding: "4px 6px", background: "#0f172a" }}>
          <div style={{ color: "#e2e8f0", fontSize: 12 }}>{effectTitle(effect)}</div>
          <div style={{ color: "#94a3b8", fontSize: 11 }}>{effectDetail(effect)}</div>
        </div>
      ))}
      {parsed.length > 6 && <div style={{ color: "#64748b", fontSize: 11 }}>+{parsed.length - 6} apport(s)</div>}
    </div>
  );
}

export function LevelDescriptionsPreview({
  keys,
  translations,
}: {
  keys?: Record<string, string>;
  translations: TranslationMap;
}) {
  const entries = Object.entries(keys || {});
  if (entries.length === 0) return <span style={{ color: "#fca5a5", fontSize: 12 }}>Aucune description par niveau</span>;
  const preferred = ["1", "5", "10", "15", "20", "25", "30"];
  const shown = preferred
    .filter((level) => keys?.[level])
    .map((level) => [level, keys![level]] as [string, string]);
  return (
    <div style={{ minWidth: 260 }}>
      <div style={{ color: "#94a3b8", fontSize: 11, marginBottom: 4 }}>{entries.length}/30 niveaux renseignes</div>
      <div style={{ display: "grid", gap: 4 }}>
        {shown.map(([level, key]) => (
          <div key={level} style={{ fontSize: 11, lineHeight: 1.35 }}>
            <strong style={{ color: "#fbbf24" }}>L{level}</strong>{" "}
            <span style={{ color: translations[key] ? "#cbd5e1" : "#fca5a5" }}>{translations[key] || "Traduction manquante"}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

function CompactList({ rows }: { rows: string[] }) {
  return (
    <div style={{ display: "flex", flexWrap: "wrap", gap: 4, minWidth: 150 }}>
      {rows.map((row) => (
        <span key={row} style={{ border: "1px solid #334155", borderRadius: 999, padding: "2px 7px", color: "#cbd5e1", fontSize: 11 }}>
          {row}
        </span>
      ))}
    </div>
  );
}

function parseEffects(effects?: unknown): any[] {
  if (!effects) return [];
  if (Array.isArray(effects)) return effects;
  if (typeof effects === "string") {
    try {
      const parsed = JSON.parse(effects);
      return Array.isArray(parsed) ? parsed : [];
    } catch {
      return [];
    }
  }
  return [];
}

function effectTitle(effect: Record<string, any>) {
  return String(effect.target || effect.effectType || "apport");
}

function effectDetail(effect: Record<string, any>) {
  if (effect.effect) return String(effect.effect);
  if (effect.role) return String(effect.role);
  if (effect.costL1) return `Cout L1: ${effect.costL1}${effect.durationL1 ? ` · Duree: ${effect.durationL1}` : ""}`;
  const parts = [effect.valueBase, effect.valuePerLevel, effect.stat].filter(Boolean).map(String);
  return parts.length ? parts.join(" · ") : String(effect.effectType || "details");
}

function formatDuration(seconds: number) {
  if (!Number.isFinite(seconds)) return "-";
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${Math.round(seconds / 60)}min`;
  if (seconds < 86400) return `${Math.round(seconds / 3600)}h`;
  return `${Math.round(seconds / 86400)}j`;
}
