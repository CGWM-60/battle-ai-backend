"use client";

import { GameAdminPage } from "../GameAdminPage";

export default function ConflictsPage() {
  return (
    <GameAdminPage
      config={{
        title: "Conflits",
        description: "Conflits PvP, guildes, factions IA et crises internes.",
        endpoint: "game/conflicts",
        columns: ["id", "worldId", "continentId", "attackerType", "defenderType", "title", "intensity", "riskLevel", "status", "startsAt", "endsAt"],
        filters: ["worldId", "continentId", "status"],
        actions: [{ label: "Resoudre", method: "POST", path: (item) => `game/conflicts/${item.id ?? item.Id}/resolve` }],
      }}
    />
  );
}
