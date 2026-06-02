"use client";

import { GameAdminPage } from "../GameAdminPage";

export default function ContinentsPage() {
  return (
    <GameAdminPage
      config={{
        title: "Continents",
        description: "Pilotage des profils IA, tension, meteo et economie continentale.",
        endpoint: "game/continents",
        columns: ["id", "worldId", "name", "index", "status", "currentPlayers", "aiBehaviorProfile", "tensionLevel", "climateState", "politicalState", "economicState"],
        filters: ["worldId", "status"],
        editable: true,
      }}
    />
  );
}
