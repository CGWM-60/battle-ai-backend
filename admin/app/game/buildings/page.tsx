"use client";

import { GameAdminPage } from "../GameAdminPage";

export default function BuildingsPage() {
  return (
    <GameAdminPage
      config={{
        title: "Catalogue de batiments",
        description: "Definitions dynamiques envoyees a Flutter avec categories, couts, effets et conditions.",
        endpoint: "game/buildings",
        columns: ["id", "key", "name", "category", "maxLevel", "isActive", "sortOrder", "updatedAt"],
        filters: ["type"],
        actions: [{ label: "Desactiver", method: "DELETE", path: (item) => `game/buildings/${item.id ?? item.Id}`, confirm: "Supprimer/desactiver ce batiment ?" }],
      }}
    />
  );
}
