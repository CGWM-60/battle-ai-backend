"use client";

import { GameAdminPage } from "../GameAdminPage";

export default function WorldsPage() {
  return (
    <GameAdminPage
      config={{
        title: "Mondes",
        description: "Capacite, tension globale, cycle et simulations des mondes NEXUS.",
        endpoint: "game/worlds",
        columns: ["id", "name", "status", "currentPlayers", "maxPlayers", "globalTensionLevel", "globalWeatherRisk", "currentCycle", "lastSimulationAt"],
        filters: ["status"],
        editable: true,
        createSample: {},
        actions: [{ label: "Simuler", method: "POST", path: (item) => `game/worlds/${item.id ?? item.Id}/simulate` }],
      }}
    />
  );
}
