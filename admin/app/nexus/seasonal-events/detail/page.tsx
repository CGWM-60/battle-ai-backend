"use client";

import { useEffect, useState } from "react";
import { AdminShell } from "../../../components/AdminShell";

const API_BASE = (process.env.NEXT_PUBLIC_NEXUS_API_BASE || "").replace(/\/$/, "");

export default function SeasonalEventDetailPage() {
  const [id, setId] = useState<string>("");
  const [event, setEvent] = useState<any>(null);

  const load = async (eventId = id) => {
    if (!eventId) return;
    const res = await fetch(`${API_BASE}/api/nexus-game/admin/seasonal-events/${eventId}`, { credentials: "same-origin" });
    if (res.ok) setEvent((await res.json()).event);
  };

  useEffect(() => {
    const eventId = new URLSearchParams(window.location.search).get("id") || "";
    setId(eventId);
    load(eventId);
  }, []);

  const transition = async (action: string) => {
    if (!id) return;
    await fetch(`${API_BASE}/api/nexus-game/admin/seasonal-events/${id}/${action}`, { method: "POST", credentials: "same-origin" });
    await load(id);
  };

  return (
    <AdminShell title="Détail événement saisonnier" description="Validation, dates, règles, récompenses et risques.">
      <div style={{ display: "flex", gap: 8, flexWrap: "wrap", marginBottom: 12 }}>
        {["approve", "reject", "schedule", "start", "end", "archive"].map((a) => <button key={a} onClick={() => transition(a)}>{a}</button>)}
      </div>
      <section className="panel"><pre>{JSON.stringify(event, null, 2)}</pre></section>
    </AdminShell>
  );
}
