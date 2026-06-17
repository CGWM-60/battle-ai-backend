"use client";

import { BookOpen, Copy, RefreshCw, Save, Trash2, Upload } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { AdminShell } from "../components/AdminShell";
import { ErrorState, LoadingState } from "../components/LoadState";
import { MetricGrid } from "../components/MetricGrid";
import { formatDate, formatNumber, loadAdminData } from "../components/api";
import type {
  AdminRolePlayQuest,
  AdminRolePlayScene,
  AdminRolePlayQuestsResponse,
  RolePlayImagePromptJob,
} from "../types";

export default function RolePlayQuestsPage() {
  const [data, setData] = useState<AdminRolePlayQuestsResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [query, setQuery] = useState("");
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [busyId, setBusyId] = useState<number | "all" | "unpublish-all" | null>(null);
  const [activeJob, setActiveJob] = useState<RolePlayImagePromptJob | null>(null);
  const [batchOnlyMissing, setBatchOnlyMissing] = useState(true);
  const [batchForce, setBatchForce] = useState(false);
  const [batchSceneCount, setBatchSceneCount] = useState(0);
  const [batchSize, setBatchSize] = useState(5);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const reload = () => {
    loadAdminData<AdminRolePlayQuestsResponse>("roleplay-quests").then((payload) => {
      setData(payload);
      setSelectedId((current) => current ?? payload.quests[0]?.id ?? null);
    }).catch((err: Error) => setError(err.message));
  };

  useEffect(() => {
    reload();
    return () => {
      if (pollRef.current) clearInterval(pollRef.current);
    };
  }, []);

  const quests = useMemo(() => {
    const needle = query.trim().toLowerCase();
    const list = data?.quests ?? [];
    if (!needle) return list;
    return list.filter((quest) =>
      [quest.title, quest.slug, quest.theme, quest.level, quest.status, String(quest.id)].some((value) =>
        value.toLowerCase().includes(needle),
      ),
    );
  }, [data, query]);

  const selectedQuest = quests.find((quest) => quest.id === selectedId) ?? quests[0] ?? null;

  function patchQuest(id: number, patch: Partial<AdminRolePlayQuest>) {
    if (!data) return;
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
    if (!window.confirm(`Supprimer la quete RP "${quest.title}" ?`)) return;
    setBusyId(quest.id);
    try {
      const response = await fetch(`/admin/api/roleplay-quests/${quest.id}`, {
        method: "DELETE",
        credentials: "same-origin",
      });
      if (!response.ok) throw new Error(`HTTP ${response.status}`);
      setSelectedId(null);
      reload();
    } catch (err) {
      setError(err instanceof Error ? err.message : "delete failed");
    } finally {
      setBusyId(null);
    }
  }

  async function unpublishAll() {
    if (!window.confirm("Retirer toutes les quetes RP du catalogue public ?")) return;
    setBusyId("unpublish-all");
    try {
      const response = await fetch("/admin/api/roleplay/quests/unpublish-all", {
        method: "POST",
        credentials: "same-origin",
        headers: { Accept: "application/json" },
      });
      if (!response.ok) throw new Error(`HTTP ${response.status}`);
      reload();
    } catch (err) {
      setError(err instanceof Error ? err.message : "unpublish-all failed");
    } finally {
      setBusyId(null);
    }
  }

  async function clearAll() {
    if (!window.confirm("Effacer toutes les quetes RP ?")) return;
    setBusyId("all");
    try {
      const response = await fetch("/admin/api/roleplay-quests", {
        method: "DELETE",
        credentials: "same-origin",
      });
      if (!response.ok) throw new Error(`HTTP ${response.status}`);
      setSelectedId(null);
      reload();
    } catch (err) {
      setError(err instanceof Error ? err.message : "clear failed");
    } finally {
      setBusyId(null);
    }
  }

  function stopJobPolling() {
    if (pollRef.current) {
      clearInterval(pollRef.current);
      pollRef.current = null;
    }
  }

  async function pollJob(jobId: number) {
    const response = await fetch(`/admin/api/roleplay/quests/image-prompts/jobs/${jobId}`, {
      credentials: "same-origin",
      headers: { Accept: "application/json" },
    });
    if (!response.ok) return;
    const payload = (await response.json()) as RolePlayImagePromptJob;
    setActiveJob(payload);
    if (["completed", "failed", "cancelled", "interrupted"].includes(payload.status)) {
      stopJobPolling();
      reload();
    }
  }

  async function startBatchJob() {
    setError(null);
    try {
      const response = await fetch("/admin/api/roleplay/quests/image-prompts/jobs", {
        method: "POST",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json", Accept: "application/json" },
        body: JSON.stringify({
          scope: "all",
          onlyMissing: batchOnlyMissing,
          forceRegenerate: batchForce,
          sceneMode: "per_chapter",
          sceneCount: batchSceneCount,
          batchSize,
        }),
      });
      const payload = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(payload.error ?? `HTTP ${response.status}`);
      setActiveJob(payload as RolePlayImagePromptJob);
      stopJobPolling();
      pollRef.current = setInterval(() => {
        void pollJob((payload as RolePlayImagePromptJob).jobId);
      }, 1500);
    } catch (err) {
      setError(err instanceof Error ? err.message : "batch job failed");
    }
  }

  async function cancelBatchJob() {
    if (!activeJob) return;
    await fetch(`/admin/api/roleplay/quests/image-prompts/jobs/${activeJob.jobId}/cancel`, {
      method: "POST",
      credentials: "same-origin",
    });
    await pollJob(activeJob.jobId);
    stopJobPolling();
  }

  return (
    <AdminShell title="Quetes RP" description="Catalogue roleplay, visuels, prompts image et upload scenes.">
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

          <section className="rp-batch-panel">
            <strong>Generation prompts images (batch)</strong>
            <div className="rp-batch-controls">
              <label><input type="checkbox" checked={batchOnlyMissing} onChange={(e) => setBatchOnlyMissing(e.target.checked)} /> Seulement manquants</label>
              <label><input type="checkbox" checked={batchForce} onChange={(e) => setBatchForce(e.target.checked)} /> Forcer regeneration</label>
              <label>Limite scenes (0 = tous chapitres) <input type="number" min={0} max={48} value={batchSceneCount} onChange={(e) => setBatchSceneCount(Number(e.target.value))} /></label>
              <label>Batch <input type="number" min={1} max={20} value={batchSize} onChange={(e) => setBatchSize(Number(e.target.value))} /></label>
              <button type="button" onClick={startBatchJob}>Mettre a jour toutes les quetes</button>
              {activeJob && ["running", "pending"].includes(activeJob.status) ? (
                <button type="button" className="danger" onClick={cancelBatchJob}>Annuler</button>
              ) : null}
            </div>
            {activeJob ? (
              <>
                <div className="rp-progress"><span style={{ width: `${activeJob.percent}%` }} /></div>
                <small>
                  {activeJob.status} — {activeJob.processedQuests}/{activeJob.totalQuests} —
                  maj {activeJob.updatedQuests}, scenes {activeJob.createdScenes}, prompts {activeJob.updatedPrompts}, erreurs {activeJob.failedQuests}
                  {activeJob.currentQuestTitle ? ` — en cours: ${activeJob.currentQuestTitle}` : ""}
                </small>
                {activeJob.errors?.length ? (
                  <ul>{activeJob.errors.map((item) => <li key={item.questId}>#{item.questId} {item.title}: {item.error}</li>)}</ul>
                ) : null}
              </>
            ) : null}
          </section>

          <section className="rp-toolbar">
            <div><BookOpen size={18} aria-hidden /><strong>{formatNumber(quests.length)} quetes</strong></div>
            <input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Filtrer..." />
            <button className="danger" type="button" onClick={unpublishAll} disabled={busyId === "unpublish-all"}>Depublier tout</button>
            <button className="danger" type="button" onClick={clearAll} disabled={busyId === "all"}><Trash2 size={16} /> Clear all</button>
          </section>

          <section className="rp-admin-layout">
            <div className="panel rp-table-panel">
              <div className="table-wrap">
                <table className="rp-table">
                  <thead>
                    <tr><th>ID</th><th>Titre</th><th>Statut</th><th>Arc</th><th>Chap.</th><th>XP</th><th>Coin</th></tr>
                  </thead>
                  <tbody>
                    {quests.map((quest) => (
                      <tr key={quest.id} className={selectedQuest?.id === quest.id ? "selected" : ""} onClick={() => setSelectedId(quest.id)}>
                        <td>#{quest.id}</td>
                        <td><strong>{quest.title}</strong><small>{quest.theme || "-"} / {quest.level || "-"}</small></td>
                        <td><span className={`status ${quest.status}`}>{quest.status}</span></td>
                        <td>{formatNumber(quest.arcCount)}</td>
                        <td>{formatNumber(quest.chapterCount)}</td>
                        <td>{formatNumber(quest.xp)}</td>
                        <td>{formatNumber(quest.coin)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>

            <QuestDetail quest={selectedQuest} busy={busyId} onPatch={patchQuest} onSave={saveQuest} onDelete={deleteQuest} onReload={reload} />
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
  busy: number | "all" | "unpublish-all" | null;
  onPatch: (id: number, patch: Partial<AdminRolePlayQuest>) => void;
  onSave: (quest: AdminRolePlayQuest) => void;
  onDelete: (quest: AdminRolePlayQuest) => void;
  onReload: () => void;
}) {
  const [sceneBusy, setSceneBusy] = useState(false);

  if (!quest) {
    return <aside className="panel rp-detail empty">Selectionne une quete RP.</aside>;
  }

  const questId = quest.id;

  async function generateQuestPrompts(force: boolean) {
    if (force && !window.confirm("Regenerer tous les prompts image de cette quete ?")) return;
    setSceneBusy(true);
    try {
      await fetch(`/admin/api/roleplay/quests/${questId}/generate-image-prompts`, {
        method: "POST",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json", Accept: "application/json" },
        body: JSON.stringify({ onlyMissing: !force, forceRegenerate: force, sceneMode: "per_chapter", sceneCount: 0 }),
      });
      onReload();
    } finally {
      setSceneBusy(false);
    }
  }

  const allScenePrompts = (quest.scenes ?? [])
    .map((scene) => `# ${scene.sceneKey}\n${scene.imagePrompt}\n${scene.imageNegativePrompt}`)
    .join("\n\n");

  return (
    <aside className="panel rp-detail">
      <header>
        <span className={`status ${quest.status}`}>{quest.status}</span>
        <h2>#{quest.id} {quest.title}</h2>
        <p>{quest.summary || "Sans resume."}</p>
      </header>

      <div className="rp-edit-strip">
        <label>XP <input type="number" value={quest.xp} onChange={(e) => onPatch(quest.id, { xp: Number(e.target.value) })} /></label>
        <label>Coins <input type="number" value={quest.coin} onChange={(e) => onPatch(quest.id, { coin: Number(e.target.value) })} /></label>
        <label>Statut
          <select value={quest.status} onChange={(e) => onPatch(quest.id, { status: e.target.value })}>
            <option value="published">published</option>
            <option value="draft">draft</option>
            <option value="archived">archived</option>
          </select>
        </label>
      </div>

      <div className="rp-detail-actions">
        <button className="primary" type="button" onClick={() => onSave(quest)} disabled={busy === quest.id}><Save size={16} /> Enregistrer</button>
        <button type="button" onClick={() => publishQuest(quest.id, onReload)}>Publier</button>
        <button type="button" onClick={() => unpublishQuest(quest.id, onReload)}>Depublier</button>
        <button className="danger" type="button" onClick={() => onDelete(quest)}><Trash2 size={16} /> Supprimer</button>
      </div>

      <section className="rp-read-block">
        <h3>Prompts image quete</h3>
        <div className="rp-copy-row">
          <button type="button" onClick={() => generateQuestPrompts(false)} disabled={sceneBusy}>Generer prompts image</button>
          <button type="button" onClick={() => generateQuestPrompts(true)} disabled={sceneBusy}>Regenerer prompts image</button>
          <CopyButton label="Copier prompt global" text={quest.imagePrompt} />
          <CopyButton label="Copier negative prompt" text={quest.imageNegativePrompt} />
          <CopyButton label="Copier tous prompts scenes" text={allScenePrompts} />
        </div>
        <dl className="rp-facts">
          <dt>Image prompt</dt><dd className="prewrap">{quest.imagePrompt || "-"}</dd>
          <dt>Negative prompt</dt><dd className="prewrap">{quest.imageNegativePrompt || "-"}</dd>
          <dt>Style</dt><dd>{quest.visualStyle || "-"}</dd>
          <dt>Tags</dt><dd>{quest.visualTags?.join(", ") || "-"}</dd>
        </dl>
      </section>

      <section className="rp-read-block">
        <h3>Structure narrative</h3>
        <div className="rp-arc-list">
          {quest.arcs.map((arc) => (
            <article key={arc.id} className="rp-arc">
              <h4>Arc {arc.position}: {arc.title}</h4>
              <p>{arc.objective || arc.summary || "Sans objectif."}</p>
              <ul>
                {arc.chapters.map((chapter) => (
                  <li key={chapter.id}>
                    <strong>Ch. {chapter.position}: {chapter.title}</strong>
                    {chapter.isBoss ? " [BOSS]" : ""} — {chapter.objective || chapter.summary || "Sans objectif."}
                    {" "}(+{chapter.xp} XP / {chapter.coin} coins)
                  </li>
                ))}
              </ul>
            </article>
          ))}
        </div>
      </section>

      <section className="rp-read-block">
        <h3>Visuels par chapitre</h3>
        <small>{formatNumber(quest.scenes?.length ?? 0)} scenes pour {formatNumber(quest.chapterCount)} chapitres</small>
        <div className="rp-arc-list">
          {quest.arcs.map((arc) => (
            <article key={`visual-${arc.id}`} className="rp-arc">
              <h4>Arc {arc.position}: {arc.title}</h4>
              {arc.chapters.map((chapter) => {
                const scene =
                  quest.scenes?.find((item) => item.chapterId === chapter.id) ??
                  quest.scenes?.find((item) => item.arcIndex === arc.position && item.chapterIndex === chapter.position) ??
                  null;
                return (
                  <div key={`chapter-visual-${chapter.id}`} className="rp-chapter-visual">
                    <h5>Chapitre {chapter.position}: {chapter.title}</h5>
                    <p>{chapter.objective || chapter.summary || "Sans objectif."}</p>
                    {scene ? (
                      <SceneCard
                        questId={quest.id}
                        scene={scene}
                        onReload={onReload}
                        busy={sceneBusy}
                        setBusy={setSceneBusy}
                        chapterId={chapter.id}
                      />
                    ) : (
                      <ChapterScenePending
                        questId={quest.id}
                        chapterId={chapter.id}
                        onReload={onReload}
                        busy={sceneBusy}
                        setBusy={setSceneBusy}
                      />
                    )}
                  </div>
                );
              })}
            </article>
          ))}
        </div>
      </section>
    </aside>
  );
}

function ChapterScenePending({
  questId,
  chapterId,
  onReload,
  busy,
  setBusy,
}: {
  questId: number;
  chapterId: number;
  onReload: () => void;
  busy: boolean;
  setBusy: (value: boolean) => void;
}) {
  async function createScenePrompt() {
    setBusy(true);
    try {
      await fetch(`/admin/api/roleplay/quests/${questId}/chapters/${chapterId}/generate-image-prompt`, {
        method: "POST",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json", Accept: "application/json" },
        body: JSON.stringify({ forceRegenerate: true }),
      });
      onReload();
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="rp-chapter-pending">
      <p>Aucune scene visuelle liee a ce chapitre.</p>
      <button type="button" onClick={createScenePrompt} disabled={busy}>
        <RefreshCw size={14} /> Creer scene + prompt
      </button>
    </div>
  );
}

function SceneCard({
  questId,
  scene,
  onReload,
  busy,
  setBusy,
  chapterId,
}: {
  questId: number;
  scene: AdminRolePlayScene;
  onReload: () => void;
  busy: boolean;
  setBusy: (value: boolean) => void;
  chapterId?: number;
}) {
  const [dragging, setDragging] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  async function uploadFiles(files: FileList | File[]) {
    const list = Array.from(files);
    if (!list.length) return;
    setBusy(true);
    try {
      const form = new FormData();
      if (list.length === 1) {
        form.append("image", list[0]);
      } else {
        list.forEach((file) => form.append("images[]", file));
      }
      const uploadPath = chapterId
        ? `/admin/api/roleplay/quests/${questId}/chapters/${chapterId}/images`
        : `/admin/api/roleplay/quests/${questId}/scenes/${scene.id}/images`;
      const response = await fetch(uploadPath, {
        method: "POST",
        credentials: "same-origin",
        body: form,
      });
      if (!response.ok) {
        const payload = await response.json().catch(() => ({}));
        throw new Error(payload.error ?? `HTTP ${response.status}`);
      }
      onReload();
    } finally {
      setBusy(false);
    }
  }

  async function deleteImage(imageId: number) {
    if (!window.confirm("Supprimer cette image ?")) return;
    setBusy(true);
    try {
      const deletePath = chapterId
        ? `/admin/api/roleplay/quests/${questId}/chapters/${chapterId}/images/${imageId}`
        : `/admin/api/roleplay/quests/${questId}/scenes/${scene.id}/images/${imageId}`;
      await fetch(deletePath, {
        method: "DELETE",
        credentials: "same-origin",
      });
      onReload();
    } finally {
      setBusy(false);
    }
  }

  async function setMain(imageId: number) {
    setBusy(true);
    try {
      const mainPath = chapterId
        ? `/admin/api/roleplay/quests/${questId}/chapters/${chapterId}/images/${imageId}/main`
        : `/admin/api/roleplay/quests/${questId}/scenes/${scene.id}/images/${imageId}/main`;
      await fetch(mainPath, {
        method: "PATCH",
        credentials: "same-origin",
      });
      onReload();
    } finally {
      setBusy(false);
    }
  }

  async function regenerateScenePrompt() {
    setBusy(true);
    try {
      const promptPath = chapterId
        ? `/admin/api/roleplay/quests/${questId}/chapters/${chapterId}/generate-image-prompt`
        : `/admin/api/roleplay/quests/${questId}/scenes/${scene.id}/generate-image-prompt`;
      await fetch(promptPath, {
        method: "POST",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json", Accept: "application/json" },
        body: JSON.stringify({ forceRegenerate: true }),
      });
      onReload();
    } finally {
      setBusy(false);
    }
  }

  return (
    <article className="rp-arc">
      <h4>{scene.sceneKey}: {scene.title}</h4>
      <p>{scene.summary || "Sans resume."}</p>
      <small>{scene.sceneType} / {scene.roomType} / {scene.atmosphere} / danger {scene.dangerLevel}</small>
      <div className="rp-copy-row">
        <CopyButton label="Copier prompt" text={scene.imagePrompt} />
        <CopyButton label="Copier negative" text={scene.imageNegativePrompt} />
        <button type="button" onClick={regenerateScenePrompt} disabled={busy}><RefreshCw size={14} /> Regenerer prompt scene</button>
      </div>
      <details>
        <summary>Prompts image complets</summary>
        <p className="prewrap">{scene.imagePrompt || "-"}</p>
        <p className="prewrap">{scene.imageNegativePrompt || "-"}</p>
      </details>

      {scene.imageUrl ? <img src={scene.imageUrl} alt={scene.title} className="rp-scene-preview" /> : null}

      <div
        className={`rp-dropzone ${dragging ? "dragging" : ""}`}
        onDragOver={(e) => { e.preventDefault(); setDragging(true); }}
        onDragLeave={() => setDragging(false)}
        onDrop={(e) => { e.preventDefault(); setDragging(false); void uploadFiles(e.dataTransfer.files); }}
      >
        <Upload size={16} /> Glisser-deposer ou
        <button type="button" onClick={() => inputRef.current?.click()} disabled={busy}>Uploader image(s)</button>
        <input ref={inputRef} type="file" accept="image/png,image/jpeg,image/webp" multiple hidden onChange={(e) => { if (e.target.files) void uploadFiles(e.target.files); e.target.value = ""; }} />
      </div>

      <div className="rp-scene-gallery">
        {(scene.images ?? []).map((image) => (
          <div key={image.id} className={`rp-scene-gallery-item ${image.isMain ? "main" : ""}`}>
            <img src={image.url} alt={image.alt || scene.title} />
            {image.isMain ? <small>Principale</small> : null}
            <div className="rp-scene-gallery-actions">
              {!image.isMain ? <button type="button" onClick={() => setMain(image.id)} disabled={busy}>Principale</button> : null}
              <CopyButton label="URL" text={image.url} />
              <button type="button" className="danger" onClick={() => deleteImage(image.id)} disabled={busy}><Trash2 size={12} /></button>
            </div>
          </div>
        ))}
      </div>
    </article>
  );
}

function CopyButton({ label, text }: { label: string; text: string }) {
  return (
    <button
      type="button"
      onClick={() => navigator.clipboard.writeText(text || "")}
      disabled={!text}
    >
      <Copy size={14} /> {label}
    </button>
  );
}

async function publishQuest(id: number, onReload: () => void) {
  const response = await fetch(`/admin/api/roleplay/quests/${id}/publish`, { method: "POST", credentials: "same-origin" });
  if (response.ok) onReload();
}

async function unpublishQuest(id: number, onReload: () => void) {
  const response = await fetch(`/admin/api/roleplay/quests/${id}/unpublish`, { method: "POST", credentials: "same-origin" });
  if (response.ok) onReload();
}