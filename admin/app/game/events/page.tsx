"use client";

import { GameAdminPage } from "../GameAdminPage";

export default function EventsPage() {
  return (
    <GameAdminPage
      config={{
        title: "Evenements",
        description: "Evenements mondiaux, continentaux, guildes et joueurs.",
        endpoint: "game/events",
        columns: ["id", "worldId", "continentId", "title", "type", "difficulty", "status", "startsAt", "endsAt", "createdByAi"],
        filters: ["worldId", "continentId", "status", "type"],
        actions: [
          { label: "Demarrer", method: "POST", path: (item) => `game/events/${item.id ?? item.Id}/force-start` },
          { label: "Finir", method: "POST", path: (item) => `game/events/${item.id ?? item.Id}/force-end` },
        ],
      }}
    />
  );
}
