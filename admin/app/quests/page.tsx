"use client";

import { Save, Sparkles } from "lucide-react";
import { useEffect, useState } from "react";
import { AdminShell } from "../components/AdminShell";
import { ErrorState, LoadingState } from "../components/LoadState";
import { MetricGrid } from "../components/MetricGrid";
import { formatDate, formatNumber, loadAdminData } from "../components/api";
import type { BattleQuest, RolePlayQuest, StatsData } from "../types";

type QuestsResponse = {
  stats: StatsData;
  recent: {
    battleQuests: BattleQuest[];
    rolePlayQuests: RolePlayQuest[];
  };
};

const providers = ["mistral", "openai", "openrouter", "xia", "claude", "gemini"];

export default function QuestsPage() {
  const [data, setData] = useState<QuestsResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadAdminData<QuestsResponse>("quests").then(setData).catch((err: Error) => setError(err.message));
  }, []);

  return (
    <AdminShell title="Quetes" description="Creation manuelle, generation IA et controle des dernieres quetes publiees.">
      {error ? <ErrorState message={error} /> : null}
      {!data && !error ? <LoadingState /> : null}
      {data ? (
        <>
          <MetricGrid
            items={[
              { label: "Quetes Battle", value: formatNumber(data.stats.BattleQuests) },
              { label: "Quetes RP", value: formatNumber(data.stats.RolePlayQuests) },
            ]}
          />
          <section className="form-grid">
            <BattleQuestForm />
            <RolePlayQuestForm />
          </section>
          <section className="form-grid">
            <GenerationForm title="Generation IA Battle" action="/admin/generate/battle" />
            <GenerationForm title="Generation IA RP" action="/admin/generate/rp" />
          </section>
          <section className="split">
            <QuestTable title="Dernieres quetes Battle" rows={data.recent.battleQuests} />
            <QuestTable title="Dernieres quetes RP" rows={data.recent.rolePlayQuests} />
          </section>
        </>
      ) : null}
    </AdminShell>
  );
}

function BattleQuestForm() {
  return (
    <article className="panel">
      <h2>Creer Quete Battle</h2>
      <form method="post" action="/admin/quests/battle">
        <input name="title" placeholder="Titre" required />
        <textarea name="content" placeholder="Question complete" required />
        <div className="row">
          <input name="theme" placeholder="Theme" />
          <input name="level" placeholder="Niveau" />
          <input name="slug" placeholder="Slug optionnel" />
        </div>
        <div className="row">
          <input name="point" type="number" placeholder="Points" />
          <input name="xp" type="number" placeholder="XP" />
          <input name="coin" type="number" placeholder="Coins" />
        </div>
        <select name="status" defaultValue="published">
          <option value="published">published</option>
          <option value="draft">draft</option>
          <option value="archived">archived</option>
        </select>
        <textarea name="metadata" placeholder='Metadata JSON optionnel: {"tag":"fun"}' />
        <button className="primary" type="submit">
          <Save size={16} aria-hidden />
          Sauvegarder
        </button>
      </form>
    </article>
  );
}

function RolePlayQuestForm() {
  return (
    <article className="panel">
      <h2>Creer Quete RP</h2>
      <form method="post" action="/admin/quests/rp">
        <input name="title" placeholder="Titre" required />
        <textarea name="summary" placeholder="Resume court" />
        <textarea name="prompt" placeholder="Prompt complet jouable" required />
        <div className="row">
          <input name="theme" placeholder="Theme" />
          <input name="level" placeholder="Niveau" />
          <input name="slug" placeholder="Slug optionnel" />
        </div>
        <div className="row">
          <input name="xp" type="number" placeholder="XP" />
          <input name="coin" type="number" placeholder="Coins" />
          <select name="status" defaultValue="published">
            <option value="published">published</option>
            <option value="draft">draft</option>
            <option value="archived">archived</option>
          </select>
        </div>
        <textarea name="metadata" placeholder='Metadata JSON optionnel: {"ton":"noir"}' />
        <button className="primary" type="submit">
          <Save size={16} aria-hidden />
          Sauvegarder
        </button>
      </form>
    </article>
  );
}

function GenerationForm({ title, action }: { title: string; action: string }) {
  return (
    <article className="panel">
      <h2>{title}</h2>
      <form method="post" action={action}>
        <div className="row">
          <select name="provider" defaultValue="mistral">
            {providers.map((provider) => (
              <option value={provider} key={provider}>
                {provider}
              </option>
            ))}
          </select>
          <input name="model" placeholder="Modele" required />
          <input name="count" type="number" defaultValue="10" min="1" max="20" />
        </div>
        <input name="api_key" type="password" placeholder="Cle API non stockee" required />
        <button className="secondary" type="submit">
          <Sparkles size={16} aria-hidden />
          Generer et sauvegarder
        </button>
      </form>
    </article>
  );
}

function QuestTable({ title, rows }: { title: string; rows: Array<BattleQuest | RolePlayQuest> }) {
  return (
    <article className="panel">
      <h2>{title}</h2>
      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th>ID</th>
              <th>Titre</th>
              <th>Theme</th>
              <th>Niveau</th>
              <th>Status</th>
              <th>Creation</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((row) => (
              <tr key={row.Id}>
                <td>#{row.Id}</td>
                <td>{row.Title}</td>
                <td>{row.Theme || "-"}</td>
                <td>{row.Level || "-"}</td>
                <td><span className={`status ${row.Status}`}>{row.Status || "-"}</span></td>
                <td>{formatDate(row.CreatedAt)}</td>
              </tr>
            ))}
            {!rows.length ? (
              <tr>
                <td colSpan={6}>Aucune quete.</td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </div>
    </article>
  );
}
