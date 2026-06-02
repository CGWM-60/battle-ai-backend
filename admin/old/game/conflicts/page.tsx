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
        editable: true,
        createSample: {
          worldId: 1,
          attackerType: "ai_faction",
          attackerId: 0,
          defenderType: "continent",
          defenderId: 1,
          title: "Conflit manuel",
          description: "Conflit cree par admin.",
          intensity: 30,
          riskLevel: "medium",
          status: "active",
          startsAt: new Date().toISOString(),
          endsAt: new Date(Date.now() + 6 * 60 * 60 * 1000).toISOString(),
          rewardsJson: {},
          penaltiesJson: {},
          createdByAi: false,
        },
        actions: [{ label: "Resoudre", method: "POST", path: (item) => `game/conflicts/${item.id ?? item.Id}/resolve` }],
      }}
    />
  );
}
