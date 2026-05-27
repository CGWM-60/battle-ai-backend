"use client";

import { GameAdminPage } from "../GameAdminPage";

export default function PlayersPage() {
  return (
    <GameAdminPage
      config={{
        title: "Joueurs et sauvegardes",
        description: "Sauvegardes serveur, monde, continent, ressources et derniere synchronisation.",
        endpoint: "game/players",
        columns: ["id", "playerId", "worldId", "continentId", "cityName", "cityLevel", "population", "satisfaction", "credits", "gems", "version", "lastSyncedAt"],
        filters: ["worldId", "continentId", "playerId"],
      }}
    />
  );
}
