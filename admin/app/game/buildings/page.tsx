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
        editable: true,
        createSample: {
          key: "solar_park",
          name: "Parc solaire",
          description: "Produit de l'energie propre.",
          category: "energy",
          maxLevel: 30,
          baseCostJson: { credits: 1000, energy: 0 },
          levelCostFormulaJson: { multiplier: 1.18 },
          effectsJson: { energyProduction: 100 },
          unlockRequirementsJson: { cityLevel: 1 },
          isActive: true,
          sortOrder: 10,
        },
        actions: [{ label: "Desactiver", method: "DELETE", path: (item) => `game/buildings/${item.id ?? item.Id}`, confirm: "Supprimer/desactiver ce batiment ?" }],
      }}
    />
  );
}
