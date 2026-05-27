"use client";

import { GameAdminPage } from "../GameAdminPage";

export default function AuditPage() {
  return (
    <GameAdminPage
      config={{
        title: "Audit admin",
        description: "Journal des actions sensibles realisees depuis l'administration.",
        endpoint: "game/audit",
        columns: ["id", "adminId", "action", "targetType", "targetId", "ipAddress", "createdAt"],
        filters: ["type"],
      }}
    />
  );
}
