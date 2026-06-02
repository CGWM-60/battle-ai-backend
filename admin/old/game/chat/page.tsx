"use client";

import { GameAdminPage } from "../GameAdminPage";

export default function ChatPage() {
  return (
    <GameAdminPage
      config={{
        title: "Chat",
        description: "Moderation des canaux monde, continent et guilde.",
        endpoint: "game/chat/messages",
        columns: ["id", "channelType", "worldId", "continentId", "guildId", "playerId", "message", "createdAt", "moderatedAt"],
        filters: ["channelType", "worldId", "continentId", "guildId", "playerId"],
        actions: [
          { label: "Masquer", method: "DELETE", path: (item) => `game/chat/messages/${item.id ?? item.Id}`, confirm: "Masquer ce message ?" },
          { label: "Restaurer", method: "POST", path: (item) => `game/chat/messages/${item.id ?? item.Id}/restore` },
        ],
      }}
    />
  );
}
