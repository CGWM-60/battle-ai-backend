"use client";

import { Save } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { AdminShell } from "../components/AdminShell";
import { ErrorState, LoadingState } from "../components/LoadState";
import { MetricGrid } from "../components/MetricGrid";
import { formatDate, formatNumber, loadAdminData } from "../components/api";
import type { AdminRolePlayQuest, AdminRolePlayQuestsResponse } from "../types";

export default function RolePlayQuestsPage() {
  const [data, setData] = useState<AdminRolePlayQuestsResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [query, setQuery] = useState("");
  const [savingId, setSavingId] = useState<number | null>(null);

  const reload = () => {
    loadAdminData<AdminRolePlayQuestsResponse>("roleplay-quests").then(setData).catch((err: Error) => setError(err.message));
  };

  useEffect(() => {
    reload();
  }, []);

  const quests = useMemo(() => {
    const needle = query.trim().toLowerCase();
    if (!data || !needle) {
      return data?.quests ?? [];
    }
    return data.quests.filter((quest) =>
      [quest.title, quest.slug, quest.theme, quest.level, quest.status, String(quest.id)].some((value) =>
        value.toLowerCase().includes(needle),
      ),
    );
  }, [data, query]);

  function updateQuest(index: number, patch: Partial<AdminRolePlayQuest>) {
    if (!data) {
      return;
    }
    const next = data.quests.map((quest, current) => (current === index ? { ...quest, ...patch } : quest));
    setData({ ...data, quests: next });
  }

  async function saveQuest(quest: AdminRolePlayQuest) {
    setSavingId(quest.id);
    setError(null);
    try {
      const response = await fetch(`/admin/api/roleplay-quests/${quest.id}`, {
        method: "PATCH",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json", Accept: "application/json" },
        body: JSON.stringify({ xp: quest.xp, coin: quest.coin, status: quest.status }),
      });
      if (!response.ok) {
        const payload = await response.json().catch(() => ({}));
        throw new Error(payload.error ?? `HTTP ${response.status}`);
      }
      reload();
    } catch (err) {
      setError(err instanceof Error ? err.message : "save failed");
    } finally {
      setSavingId(null);
    }
  }

  return (
    <AdminShell title="Quetes RP" description="Lecture des quetes roleplay, structure arcs/chapitres et recompenses XP/coins.">
      {error ? <ErrorState message={error} /> : null}
      {!data && !error ? <LoadingState /> : null}
      {data ? (
        <>
          <MetricGrid
            items={[
              { label: "Quetes RP", value: formatNumber(data.stats.totalQuests) },
              { label: "Publiees", value: formatNumber(data.stats.published), tone: "good" },
              { label: "Draft", value: formatNumber(data.stats.draft) },
              { label: "Archivees", value: formatNumber(data.stats.archived) },
              { label: "Arcs", value: formatNumber(data.stats.totalArcs) },
              { label: "Chapitres", value: formatNumber(data.stats.totalChapters) },
            ]}
          />

          <section className="panel">
            <h2>Catalogue RP</h2>
            <input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Filtrer par id, titre, theme, niveau ou statut" />
            <div className="rp-quest-list">
              {quests.map((quest, index) => (
                <RolePlayQuestCard
                  key={quest.id}
                  quest={quest}
                  saving={savingId === quest.id}
                  onChange={(patch) => updateQuest(index, patch)}
                  onSave={() => saveQuest(quest)}
                />
              ))}
              {!quests.length ? <p className="hint">Aucune quete RP trouvee.</p> : null}
            </div>
          </section>
        </>
      ) : null}
    </AdminShell>
  );
}

function RolePlayQuestCard({
  quest,
  saving,
  onChange,
  onSave,
}: {
  quest: AdminRolePlayQuest;
  saving: boolean;
  onChange: (patch: Partial<AdminRolePlayQuest>) => void;
  onSave: () => void;
}) {
  return (
    <article className="rp-quest-card">
      <header>
        <div>
          <span className={`status ${quest.status}`}>{quest.status || "-"}</span>
          <h3>#{quest.id} {quest.title}</h3>
          <p>{quest.summary || "Sans resume."}</p>
        </div>
        <dl>
          <dt>Arcs</dt>
          <dd>{formatNumber(quest.arcCount)}</dd>
          <dt>Chapitres</dt>
          <dd>{formatNumber(quest.chapterCount)}</dd>
          <dt>Maj</dt>
          <dd>{formatDate(quest.updatedAt)}</dd>
        </dl>
      </header>

      <div className="rp-reward-row">
        <label>
          XP
          <input type="number" value={quest.xp} onChange={(event) => onChange({ xp: Number(event.target.value) })} />
        </label>
        <label>
          Coins
          <input type="number" value={quest.coin} onChange={(event) => onChange({ coin: Number(event.target.value) })} />
        </label>
        <label>
          Status
          <select value={quest.status} onChange={(event) => onChange({ status: event.target.value })}>
            <option value="published">published</option>
            <option value="draft">draft</option>
            <option value="archived">archived</option>
          </select>
        </label>
        <button className="primary" type="button" onClick={onSave} disabled={saving}>
          <Save size={16} aria-hidden />
          {saving ? "Enregistrement" : "Enregistrer"}
        </button>
      </div>

      <details>
        <summary>Lire le prompt</summary>
        <p className="prewrap">{quest.prompt}</p>
      </details>

      <details>
        <summary>Voir les arcs et chapitres</summary>
        <div className="rp-arc-list">
          {quest.arcs.map((arc) => (
            <section key={arc.id} className="rp-arc">
              <h4>Arc {arc.position}: {arc.title}</h4>
              <p>{arc.objective || arc.summary || "Sans objectif."}</p>
              <ul>
                {arc.chapters.map((chapter) => (
                  <li key={chapter.id}>
                    <strong>Chapitre {chapter.position}: {chapter.title}</strong>
                    <span>{chapter.objective || chapter.summary || "Sans objectif."}</span>
                    <small>{chapter.isBoss ? "Boss" : "Standard"} - XP {chapter.xp} - Coins {chapter.coin}</small>
                  </li>
                ))}
              </ul>
            </section>
          ))}
        </div>
      </details>
    </article>
  );
}
