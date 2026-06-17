"use client";

import { BookOpen, Save, Trash2 } from "lucide-react";
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
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [busyId, setBusyId] = useState<number | "all" | "backfill" | "unpublish-all" | null>(null);
  const [backfillResult, setBackfillResult] = useState<string | null>(null);

  const reload = () => {
    loadAdminData<AdminRolePlayQuestsResponse>("roleplay-quests").then((payload) => {
      setData(payload);
      setSelectedId((current) => current ?? payload.quests[0]?.id ?? null);
    }).catch((err: Error) => setError(err.message));
  };

  useEffect(() => {
    reload();
  }, []);

  const quests = useMemo(() => {
    const needle = query.trim().toLowerCase();
    const list = data?.quests ?? [];
    if (!needle) {
      return list;
    }
    return list.filter((quest) =>
      [quest.title, quest.slug, quest.theme, quest.level, quest.status, String(quest.id)].some((value) =>
        value.toLowerCase().includes(needle),
      ),
    );
  }, [data, query]);

  const selectedQuest = quests.find((quest) => quest.id === selectedId) ?? quests[0] ?? null;

  function patchQuest(id: number, patch: Partial<AdminRolePlayQuest>) {
    if (!data) {
      return;
    }
    setData({
      ...data,
      quests: data.quests.map((quest) => (quest.id === id ? { ...quest, ...patch } : quest)),
    });
  }

  async function saveQuest(quest: AdminRolePlayQuest) {
    setBusyId(quest.id);
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
      setBusyId(null);
    }
  }

  async function deleteQuest(quest: AdminRolePlayQuest) {
    if (!window.confirm(`Supprimer la quete RP "${quest.title}" ?`)) {
      return;
    }
    setBusyId(quest.id);
    setError(null);
    try {
      const response = await fetch(`/admin/api/roleplay-quests/${quest.id}`, {
        method: "DELETE",
        credentials: "same-origin",
      });
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }
      setSelectedId(null);
      reload();
    } catch (err) {
      setError(err instanceof Error ? err.message : "delete failed");
    } finally {
      setBusyId(null);
    }
  }

  async function unpublishAll() {
    if (!window.confirm("Cette action va retirer toutes les quetes RP du catalogue public. Continuer ?")) {
      return;
    }
    setBusyId("unpublish-all");
    setError(null);
    try {
      const response = await fetch("/admin/api/roleplay/quests/unpublish-all", {
        method: "POST",
        credentials: "same-origin",
        headers: { Accept: "application/json" },
      });
      if (!response.ok) {
        const payload = await response.json().catch(() => ({}));
        throw new Error(payload.error ?? `HTTP ${response.status}`);
      }
      reload();
    } catch (err) {
      setError(err instanceof Error ? err.message : "unpublish-all failed");
    } finally {
      setBusyId(null);
    }
  }

  async function backfillImagePrompts() {
    setBusyId("backfill");
    setError(null);
    setBackfillResult(null);
    try {
      const response = await fetch("/admin/api/roleplay/quests/backfill-image-prompts", {
        method: "POST",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json", Accept: "application/json" },
        body: JSON.stringify({ onlyMissing: true, sceneCount: 3 }),
      });
      const payload = await response.json().catch(() => ({}));
      if (!response.ok) {
        throw new Error(payload.error ?? `HTTP ${response.status}`);
      }
      setBackfillResult(
        `Mises a jour: ${payload.updatedQuests ?? 0} quetes, ${payload.createdScenes ?? 0} scenes, ${payload.updatedPrompts ?? 0} prompts`,
      );
      reload();
    } catch (err) {
      setError(err instanceof Error ? err.message : "backfill failed");
    } finally {
      setBusyId(null);
    }
  }

  async function clearAll() {
    if (!window.confirm("Effacer toutes les quetes RP du systeme ? Les sessions existantes gardent leur historique mais ne seront plus liees a un template.")) {
      return;
    }
    setBusyId("all");
    setError(null);
    try {
      const response = await fetch("/admin/api/roleplay-quests", {
        method: "DELETE",
        credentials: "same-origin",
      });
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }
      setSelectedId(null);
      reload();
    } catch (err) {
      setError(err instanceof Error ? err.message : "clear failed");
    } finally {
      setBusyId(null);
    }
  }

  return (
    <AdminShell title="Quetes RP" description="Catalogue roleplay, recompenses, arcs et chapitres.">
      {error ? <ErrorState message={error} /> : null}
      {!data && !error ? <LoadingState /> : null}
      {data ? (
        <>
          <MetricGrid
            items={[
              { label: "Quetes RP", value: formatNumber(data.stats.totalQuests) },
              { label: "Publiees", value: formatNumber(data.stats.published), tone: "good" },
              { label: "Brouillons", value: formatNumber(data.stats.draft) },
              { label: "Archivees", value: formatNumber(data.stats.archived) },
              { label: "Arcs", value: formatNumber(data.stats.totalArcs) },
              { label: "Chapitres", value: formatNumber(data.stats.totalChapters) },
            ]}
          />

          <section className="rp-toolbar">
            <div>
              <BookOpen size={18} aria-hidden />
              <strong>{formatNumber(quests.length)} quetes affichees</strong>
            </div>
            <input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Filtrer par id, titre, theme, niveau ou statut" />
            <button type="button" onClick={backfillImagePrompts} disabled={busyId === "backfill"}>
              {busyId === "backfill" ? "Backfill..." : "Mettre a jour prompts images"}
            </button>
            <button className="danger" type="button" onClick={unpublishAll} disabled={busyId === "unpublish-all"}>
              Depublier toutes les quetes RP
            </button>
            <button className="danger" type="button" onClick={clearAll} disabled={busyId === "all" || data.stats.totalQuests === 0}>
              <Trash2 size={16} aria-hidden />
              Clear all
            </button>
          </section>
          {backfillResult ? <p className="rp-backfill-result">{backfillResult}</p> : null}

          <section className="rp-admin-layout">
            <div className="panel rp-table-panel">
              <div className="table-wrap">
                <table className="rp-table">
                  <thead>
                    <tr>
                      <th>ID</th>
                      <th>Titre</th>
                      <th>Statut</th>
                      <th>Arc</th>
                      <th>Chap.</th>
                      <th>XP</th>
                      <th>Coin</th>
                    </tr>
                  </thead>
                  <tbody>
                    {quests.map((quest) => (
                      <tr
                        key={quest.id}
                        className={selectedQuest?.id === quest.id ? "selected" : ""}
                        onClick={() => setSelectedId(quest.id)}
                      >
                        <td>#{quest.id}</td>
                        <td>
                          <strong>{quest.title}</strong>
                          <small>{quest.theme || "-"} / {quest.level || "-"}</small>
                        </td>
                        <td><span className={`status ${quest.status}`}>{quest.status}</span></td>
                        <td>{formatNumber(quest.arcCount)}</td>
                        <td>{formatNumber(quest.chapterCount)}</td>
                        <td>{formatNumber(quest.xp)}</td>
                        <td>{formatNumber(quest.coin)}</td>
                      </tr>
                    ))}
                    {!quests.length ? (
                      <tr>
                        <td colSpan={7}>Aucune quete RP.</td>
                      </tr>
                    ) : null}
                  </tbody>
                </table>
              </div>
            </div>

            <QuestDetail
              quest={selectedQuest}
              busy={busyId}
              onPatch={patchQuest}
              onSave={saveQuest}
              onDelete={deleteQuest}
              onReload={reload}
            />
          </section>
        </>
      ) : null}
    </AdminShell>
  );
}

function QuestDetail({
  quest,
  busy,
  onPatch,
  onSave,
  onDelete,
  onReload,
}: {
  quest: AdminRolePlayQuest | null;
  busy: number | "all" | "backfill" | "unpublish-all" | null;
  onPatch: (id: number, patch: Partial<AdminRolePlayQuest>) => void;
  onSave: (quest: AdminRolePlayQuest) => void;
  onDelete: (quest: AdminRolePlayQuest) => void;
  onReload: () => void;
}) {
  if (!quest) {
    return <aside className="panel rp-detail empty">Selectionne une quete RP.</aside>;
  }

  return (
    <aside className="panel rp-detail">
      <header>
        <span className={`status ${quest.status}`}>{quest.status}</span>
        <h2>#{quest.id} {quest.title}</h2>
        <p>{quest.summary || "Sans resume."}</p>
      </header>

      <div className="rp-edit-strip">
        <label>
          XP
          <input type="number" value={quest.xp} onChange={(event) => onPatch(quest.id, { xp: Number(event.target.value) })} />
        </label>
        <label>
          Coins
          <input type="number" value={quest.coin} onChange={(event) => onPatch(quest.id, { coin: Number(event.target.value) })} />
        </label>
        <label>
          Statut
          <select value={quest.status} onChange={(event) => onPatch(quest.id, { status: event.target.value })}>
            <option value="published">published</option>
            <option value="draft">draft</option>
            <option value="archived">archived</option>
          </select>
        </label>
      </div>

      <div className="rp-detail-actions">
        <button className="primary" type="button" onClick={() => onSave(quest)} disabled={busy === quest.id}>
          <Save size={16} aria-hidden />
          Enregistrer
        </button>
        <button type="button" onClick={() => publishQuest(quest.id, onReload)} disabled={busy === quest.id}>
          Publier
        </button>
        <button type="button" onClick={() => unpublishQuest(quest.id, onReload)} disabled={busy === quest.id}>
          Depublier
        </button>
        <button className="danger" type="button" onClick={() => onDelete(quest)} disabled={busy === quest.id}>
          <Trash2 size={16} aria-hidden />
          Supprimer
        </button>
      </div>

      <dl className="rp-facts">
        <dt>Slug</dt>
        <dd>{quest.slug}</dd>
        <dt>Theme</dt>
        <dd>{quest.theme || "-"}</dd>
        <dt>Niveau</dt>
        <dd>{quest.level || "-"}</dd>
        <dt>Arcs / chapitres</dt>
        <dd>{quest.arcCount} / {quest.chapterCount}</dd>
        <dt>Creation</dt>
        <dd>{formatDate(quest.createdAt)}</dd>
      </dl>

      <section className="rp-read-block">
        <h3>Prompt</h3>
        <p className="prewrap">{quest.prompt}</p>
      </section>

      <section className="rp-read-block">
        <h3>Visuels de quete</h3>
        <dl className="rp-facts">
          <dt>Image prompt</dt>
          <dd className="prewrap">{quest.imagePrompt || "-"}</dd>
          <dt>Negative prompt</dt>
          <dd className="prewrap">{quest.imageNegativePrompt || "-"}</dd>
          <dt>Style</dt>
          <dd>{quest.visualStyle || "-"}</dd>
          <dt>Tags</dt>
          <dd>{quest.visualTags?.join(", ") || "-"}</dd>
        </dl>
        <div className="rp-arc-list">
          {(quest.scenes ?? []).map((scene) => (
            <article key={scene.id} className="rp-arc">
              <h4>{scene.sceneKey}: {scene.title}</h4>
              <p>{scene.summary || "Sans resume."}</p>
              <small>{scene.sceneType} / {scene.roomType} / {scene.atmosphere} / danger {scene.dangerLevel}</small>
              <details>
                <summary>Prompts image</summary>
                <p className="prewrap">{scene.imagePrompt || "-"}</p>
                <p className="prewrap">{scene.imageNegativePrompt || "-"}</p>
              </details>
              {scene.imageUrl ? <img src={scene.imageUrl} alt={scene.title} className="rp-scene-preview" /> : null}
              <SceneImageUpload questId={quest.id} sceneId={scene.id} onDone={onReload} />
            </article>
          ))}
        </div>
      </section>

      <section className="rp-read-block">
        <h3>Structure</h3>
        <div className="rp-arc-list">
          {quest.arcs.map((arc) => (
            <article key={arc.id} className="rp-arc">
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
            </article>
          ))}
        </div>
      </section>
    </aside>
  );
}

async function publishQuest(id: number, onReload: () => void) {
  const response = await fetch(`/admin/api/roleplay/quests/${id}/publish`, {
    method: "POST",
    credentials: "same-origin",
    headers: { Accept: "application/json" },
  });
  if (response.ok) onReload();
}

async function unpublishQuest(id: number, onReload: () => void) {
  const response = await fetch(`/admin/api/roleplay/quests/${id}/unpublish`, {
    method: "POST",
    credentials: "same-origin",
    headers: { Accept: "application/json" },
  });
  if (response.ok) onReload();
}

function SceneImageUpload({
  questId,
  sceneId,
  onDone,
}: {
  questId: number;
  sceneId: number;
  onDone: () => void;
}) {
  async function onFileChange(event: React.ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    if (!file) return;
    const form = new FormData();
    form.append("image", file);
    const response = await fetch(
      `/admin/api/roleplay/quests/${questId}/scenes/${sceneId}/images`,
      { method: "POST", credentials: "same-origin", body: form },
    );
    if (response.ok) onDone();
    event.target.value = "";
  }

  return (
    <label className="rp-upload-btn">
      Uploader image
      <input type="file" accept="image/png,image/jpeg,image/webp" onChange={onFileChange} hidden />
    </label>
  );
}
