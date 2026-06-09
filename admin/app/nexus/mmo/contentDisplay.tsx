"use client";

import type React from "react";

export type TranslationMap = Record<string, string>;
export type CatalogAsset = {
  key: string;
  fileName?: string;
  url: string | null;
};

const mutedText = "#94a3b8";
const subtleText = "#64748b";
const border = "#334155";
const panelBg = "#0f172a";

const clampStyle = (lines: number): React.CSSProperties => ({
  display: "-webkit-box",
  WebkitLineClamp: lines,
  WebkitBoxOrient: "vertical",
  overflow: "hidden",
});

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
    <div style={{ minWidth: 0 }}>
      <div style={{ color: value ? "#e2e8f0" : "#fca5a5", fontSize: 13, lineHeight: 1.35, ...clampStyle(3) }}>
        {value || "Traduction manquante"}
      </div>
      <code style={{ display: "block", color: subtleText, fontSize: 10, marginTop: 4, whiteSpace: "normal", wordBreak: "break-word" }}>
        {labelKey || "-"}
      </code>
    </div>
  );
}

export function CatalogTable({ children }: { children: React.ReactNode }) {
  return (
    <div style={{ overflowX: "auto", border: `1px solid ${border}`, borderRadius: 8, background: "#020617" }}>
      <table
        className="data-table"
        style={{
          width: "100%",
          minWidth: 1080,
          borderCollapse: "collapse",
          tableLayout: "fixed",
        }}
      >
        {children}
      </table>
    </div>
  );
}

export function CatalogIdentity({
  titleKey,
  contentId,
  translations,
  badges,
}: {
  titleKey?: string;
  contentId?: string;
  translations: TranslationMap;
  badges: string[];
}) {
  return (
    <div style={{ display: "grid", gap: 6, minWidth: 0 }}>
      <div style={{ color: "#f8fafc", fontWeight: 700, fontSize: 14, lineHeight: 1.25, ...clampStyle(2) }}>
        {translated(translations, titleKey) || titleKey || contentId || "-"}
      </div>
      <code style={{ color: subtleText, fontSize: 11, wordBreak: "break-word" }}>{contentId || "contentId manquant"}</code>
      <div style={{ display: "flex", flexWrap: "wrap", gap: 4 }}>
        {badges.filter(Boolean).map((badge) => <Chip key={badge}>{badge}</Chip>)}
      </div>
    </div>
  );
}

export function DescriptionSummary({
  translations,
  descriptionKey,
  flavorTextKey,
}: {
  translations: TranslationMap;
  descriptionKey?: string;
  flavorTextKey?: string;
}) {
  const description = translated(translations, descriptionKey);
  const flavor = translated(translations, flavorTextKey);
  return (
    <div style={{ minWidth: 0 }}>
      <div style={{ color: description ? "#cbd5e1" : "#fca5a5", fontSize: 13, lineHeight: 1.35, ...clampStyle(4) }}>
        {description || "Description globale manquante"}
      </div>
      {flavor && <div style={{ color: mutedText, fontSize: 11, lineHeight: 1.35, marginTop: 6, ...clampStyle(2) }}>{flavor}</div>}
      <details style={{ marginTop: 6 }}>
        <summary style={{ color: subtleText, cursor: "pointer", fontSize: 11 }}>Clés traduction</summary>
        <code style={{ display: "block", color: subtleText, fontSize: 10, marginTop: 4, wordBreak: "break-word" }}>{descriptionKey || "-"}</code>
        {flavorTextKey && <code style={{ display: "block", color: subtleText, fontSize: 10, marginTop: 2, wordBreak: "break-word" }}>{flavorTextKey}</code>}
      </details>
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
    <div style={{ display: "grid", gap: 4, minWidth: 0 }}>
      {parsed.slice(0, 3).map((effect, index) => (
        <div key={index} style={{ border: `1px solid ${border}`, borderRadius: 6, padding: "4px 6px", background: panelBg }}>
          <div style={{ color: "#e2e8f0", fontSize: 12 }}>{effectTitle(effect)}</div>
          <div style={{ color: mutedText, fontSize: 11, ...clampStyle(2) }}>{effectDetail(effect)}</div>
        </div>
      ))}
      {parsed.length > 3 && (
        <details>
          <summary style={{ color: subtleText, cursor: "pointer", fontSize: 11 }}>+{parsed.length - 3} apport(s)</summary>
          <pre style={{ whiteSpace: "pre-wrap", color: mutedText, fontSize: 10, marginTop: 6, maxHeight: 180, overflow: "auto" }}>
            {JSON.stringify(parsed.slice(3), null, 2)}
          </pre>
        </details>
      )}
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
  const preferred = ["1", "10", "20", "30"];
  const shown = preferred
    .filter((level) => keys?.[level])
    .map((level) => [level, keys![level]] as [string, string]);
  return (
    <div style={{ minWidth: 0 }}>
      <div style={{ color: mutedText, fontSize: 11, marginBottom: 4 }}>{entries.length}/30 niveaux renseignes</div>
      <div style={{ display: "grid", gap: 4 }}>
        {shown.map(([level, key]) => (
          <div key={level} style={{ fontSize: 11, lineHeight: 1.35 }}>
            <strong style={{ color: "#fbbf24" }}>L{level}</strong>{" "}
            <span style={{ color: translations[key] ? "#cbd5e1" : "#fca5a5", ...clampStyle(2) }}>{translations[key] || "Traduction manquante"}</span>
          </div>
        ))}
      </div>
      <details style={{ marginTop: 6 }}>
        <summary style={{ color: subtleText, cursor: "pointer", fontSize: 11 }}>Voir toutes les clés</summary>
        <div style={{ maxHeight: 160, overflow: "auto", marginTop: 6 }}>
          {entries.map(([level, key]) => (
            <code key={level} style={{ display: "block", color: subtleText, fontSize: 10, wordBreak: "break-word" }}>
              L{level}: {key}
            </code>
          ))}
        </div>
      </details>
    </div>
  );
}

export function AssetPreview({ assets, fallbackLabel }: { assets: CatalogAsset[]; fallbackLabel: string }) {
  const shown = assets.slice(0, 5);
  if (shown.length === 0) {
    return (
      <div style={{ width: 76, height: 58, background: "#1e2937", borderRadius: 6, display: "flex", alignItems: "center", justifyContent: "center", fontSize: 10, color: subtleText }}>
        no img
      </div>
    );
  }
  return (
    <div style={{ minWidth: 0 }}>
      <div style={{ display: "flex", flexWrap: "wrap", gap: 4 }}>
        {shown.map((asset) => (
          <div key={asset.key} style={{ position: "relative" }}>
            <img
              src={asset.url!}
              alt={`${fallbackLabel}-${asset.key}`}
              style={{ width: 42, height: 42, objectFit: "cover", borderRadius: 5, border: `1px solid ${border}`, background: panelBg }}
              onError={(e) => (e.currentTarget.style.opacity = "0.18")}
            />
            <span style={{ position: "absolute", left: 2, bottom: 2, fontSize: 8, padding: "1px 3px", borderRadius: 3, background: "rgba(15,23,42,0.82)", color: "#cbd5e1" }}>{asset.key}</span>
          </div>
        ))}
      </div>
      <details style={{ marginTop: 6 }}>
        <summary style={{ color: subtleText, cursor: "pointer", fontSize: 11 }}>Références</summary>
        <div style={{ maxHeight: 140, overflow: "auto", marginTop: 4 }}>
          {assets.map((asset) => (
            <code key={asset.key} style={{ display: "block", color: subtleText, fontSize: 10, wordBreak: "break-word" }}>
              {asset.key}: {asset.fileName}
            </code>
          ))}
        </div>
      </details>
    </div>
  );
}

function CompactList({ rows }: { rows: string[] }) {
  return (
    <div style={{ display: "flex", flexWrap: "wrap", gap: 4, minWidth: 150 }}>
      {rows.map((row) => (
        <span key={row} style={{ border: `1px solid ${border}`, borderRadius: 999, padding: "2px 7px", color: "#cbd5e1", fontSize: 11 }}>
          {row}
        </span>
      ))}
    </div>
  );
}

function Chip({ children }: { children: React.ReactNode }) {
  return (
    <span style={{ border: `1px solid ${border}`, borderRadius: 999, padding: "2px 7px", color: "#cbd5e1", fontSize: 11, background: panelBg }}>
      {children}
    </span>
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
