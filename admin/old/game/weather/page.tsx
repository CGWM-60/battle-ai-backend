"use client";

import { GameAdminPage } from "../GameAdminPage";

export default function WeatherPage() {
  return (
    <GameAdminPage
      config={{
        title: "Meteo",
        description: "Alertes climatiques et effets actifs par continent.",
        endpoint: "game/weather",
        columns: ["id", "worldId", "continentId", "type", "severity", "title", "startsAt", "endsAt", "createdByAi"],
        filters: ["worldId", "continentId", "type"],
        editable: true,
        createSample: {
          worldId: 1,
          continentId: 1,
          type: "tempete",
          severity: 25,
          title: "Tempete admin",
          description: "Meteo creee manuellement.",
          startsAt: new Date().toISOString(),
          endsAt: new Date(Date.now() + 3 * 60 * 60 * 1000).toISOString(),
          effectsJson: { energy: -10 },
          createdByAi: false,
        },
        actions: [{ label: "Terminer", method: "POST", path: (item) => `game/weather/${item.id ?? item.Id}/end` }],
      }}
    />
  );
}
