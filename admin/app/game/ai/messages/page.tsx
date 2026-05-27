"use client";

import { GameAdminPage } from "../../GameAdminPage";

export default function AIMessagesPage() {
  return (
    <GameAdminPage
      config={{
        title: "Messages IA",
        description: "Messages quotidiens et transmissions manuelles de NEXUS.",
        endpoint: "game/ai/messages",
        columns: ["id", "worldId", "continentId", "playerId", "title", "tone", "isRead", "createdAt"],
        filters: ["worldId", "continentId", "playerId"],
      }}
    />
  );
}
