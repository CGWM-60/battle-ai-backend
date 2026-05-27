"use client";

import { GameAdminPage } from "../../GameAdminPage";

export default function AIDecisionsPage() {
  return (
    <GameAdminPage
      config={{
        title: "Decisions IA",
        description: "Audit des entrees, sorties, providers, erreurs et changements appliques par NEXUS.",
        endpoint: "game/ai/decisions",
        columns: ["id", "worldId", "continentId", "type", "provider", "model", "status", "error", "createdAt"],
        filters: ["worldId", "continentId", "status", "type"],
        actions: [{ label: "Dry-run", method: "POST", path: (item) => `game/ai/decisions/${item.id ?? item.Id}/replay-dry-run` }],
      }}
    />
  );
}
