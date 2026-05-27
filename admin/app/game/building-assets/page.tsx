"use client";

import { GameAdminPage } from "../GameAdminPage";

export default function BuildingAssetsPage() {
  return (
    <GameAdminPage
      config={{
        title: "Assets de batiments",
        description: "Images niveau par niveau, variantes, hash et versions d'assets.",
        endpoint: "game/building-assets",
        columns: ["id", "buildingDefinitionId", "level", "variant", "imageUrl", "imageHash", "imageSize", "version", "isActive"],
        filters: ["buildingDefinitionId"],
      }}
    />
  );
}
