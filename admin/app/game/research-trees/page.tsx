"use client";

import { GameAdminPage } from "../GameAdminPage";

export default function ResearchTreesPage() {
  return (
    <GameAdminPage
      config={{
        title: "Arbres de competence",
        description: "CRUD des arbres de recherche rattaches aux batiments du monde IA.",
        endpoint: "game/research-trees",
        columns: ["id", "key", "name", "domain", "buildingKey", "isActive", "sortOrder", "updatedAt"],
        filters: ["buildingKey", "domain", "isActive"],
        editable: true,
        createSample: {
          key: "technologie_science",
          name: "TECHNOLOGIE & SCIENCE",
          description: "Arbre de recherche technologie et science.",
          domain: "TECHNOLOGIE & SCIENCE",
          buildingKey: "research_center",
          isActive: true,
          sortOrder: 10,
        },
        actions: [{ label: "Desactiver", method: "DELETE", path: (item) => `game/research-trees/${item.id ?? item.Id}`, confirm: "Supprimer/desactiver cet arbre ?" }],
      }}
    />
  );
}
