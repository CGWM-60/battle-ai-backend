"use client";

import { GameAdminPage } from "../GameAdminPage";

export default function ResearchNodesPage() {
  return (
    <GameAdminPage
      config={{
        title: "Noeuds de recherche",
        description: "CRUD des branches, niveaux, ressources et prerequis des arbres de competence.",
        endpoint: "game/research-nodes",
        columns: ["id", "researchTreeDefinitionId", "key", "name", "branch", "maxLevel", "isActive", "sortOrder"],
        filters: ["researchTreeDefinitionId", "domain", "isActive"],
        editable: true,
        createSample: {
          researchTreeDefinitionId: 1,
          key: "intelligence_artificielle",
          name: "Intelligence Artificielle",
          description: "IA, machine learning, robotique",
          domain: "TECHNOLOGIE & SCIENCE",
          branch: "Intelligence Artificielle",
          resourcesJson: ["Science", "Tech", "Math"],
          parentKeysJson: [],
          requirementsJson: {},
          effectsJson: {},
          levelProgressionJson: [
            { level: 1, durationMinutes: 180, cumulativeMinutes: 180 },
            { level: 2, durationMinutes: 300, cumulativeMinutes: 480 },
          ],
          maxLevel: 2,
          positionX: 0,
          positionY: 0,
          isActive: true,
          sortOrder: 10,
        },
        actions: [{ label: "Desactiver", method: "DELETE", path: (item) => `game/research-nodes/${item.id ?? item.Id}`, confirm: "Supprimer/desactiver ce noeud ?" }],
      }}
    />
  );
}
