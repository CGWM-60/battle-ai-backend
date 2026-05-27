"use client";

import { GameAdminPage } from "../../GameAdminPage";

export default function AIFactionsPage() {
  return (
    <GameAdminPage
      config={{
        title: "Factions IA",
        description: "Factions autonomes utilisees par NEXUS pour garder les mondes actifs.",
        endpoint: "game/ai/factions",
        columns: ["id", "worldId", "continentId", "name", "type", "aggressiveness", "diplomacy", "economy", "militaryPower", "climateResistance", "status"],
        filters: ["worldId", "continentId", "status"],
      }}
    />
  );
}
