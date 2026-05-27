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
        actions: [{ label: "Terminer", method: "POST", path: (item) => `game/weather/${item.id ?? item.Id}/end` }],
      }}
    />
  );
}
