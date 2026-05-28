"use client";

import { GameAdminPage } from "../GameAdminPage";

export default function ResourcesPage() {
  return (
    <GameAdminPage
      config={{
        title: "Ressources systeme",
        description: "Ressources referencees par les arbres de recherche et autres systemes dynamiques.",
        endpoint: "game/resources",
        columns: ["id", "key", "name", "category", "isActive", "sortOrder", "updatedAt"],
        filters: ["isActive"],
        editable: true,
        createSample: {
          key: "science",
          name: "Science",
          description: "Ressource de recherche scientifique.",
          category: "research",
          isActive: true,
          sortOrder: 10,
        },
        actions: [{ label: "Desactiver", method: "DELETE", path: (item) => `game/resources/${item.id ?? item.Id}`, confirm: "Supprimer/desactiver cette ressource ?" }],
      }}
    />
  );
}
