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
        actions: [{ label: "Dissoudre", method: "DELETE", path: (item) => `game/guilds/${item.id ?? item.Id}`, confirm: "Dissoudre cette guilde ?" }],
      }}
    />
  );
}
