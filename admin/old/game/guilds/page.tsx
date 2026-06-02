"use client";

import { GameAdminPage } from "../GameAdminPage";

export default function GuildsPage() {
  return (
    <GameAdminPage
      config={{
        title: "Guildes",
        description: "Alliances, proprietaires, niveau, membres et visibilite.",
        endpoint: "game/guilds",
        columns: ["id", "worldId", "name", "tag", "ownerPlayerId", "level", "xp", "maxMembers", "visibility", "requiredLevel"],
        filters: ["worldId"],
        editable: true,
        createSample: {
          worldId: 1,
          name: "Alliance admin",
          tag: "ADM",
          description: "Guilde creee depuis l'administration.",
          ownerPlayerId: 1,
          level: 1,
          xp: 0,
          maxMembers: 30,
          visibility: "open",
          requiredLevel: 1,
        },
        actions: [{ label: "Dissoudre", method: "DELETE", path: (item) => `game/guilds/${item.id ?? item.Id}`, confirm: "Dissoudre cette guilde ?" }],
      }}
    />
  );
}
