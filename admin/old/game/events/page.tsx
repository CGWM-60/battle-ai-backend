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
        editable: true,
        createSample: {
          worldId: 1,
          title: "Operation controlee",
          description: "Evenement manuel admin.",
          type: "manual",
          difficulty: "medium",
          status: "active",
          startsAt: new Date().toISOString(),
          endsAt: new Date(Date.now() + 2 * 60 * 60 * 1000).toISOString(),
          durationMinutes: 120,
          rewardsJson: { credits: 500, xp: 50 },
          requirementsJson: {},
          consequencesJson: {},
          createdByAi: false,
        },
        actions: [
          { label: "Demarrer", method: "POST", path: (item) => `game/events/${item.id ?? item.Id}/force-start` },
          { label: "Finir", method: "POST", path: (item) => `game/events/${item.id ?? item.Id}/force-end` },
        ],
      }}
    />
  );
}
