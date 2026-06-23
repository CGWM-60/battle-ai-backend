"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import type { ReactNode } from "react";

export default function DeprecatedNexusLayout({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  const isTranslationsRoute =
    pathname === "/nexus/translations" ||
    pathname.startsWith("/nexus/translations/") ||
    pathname === "/admin/nexus/translations" ||
    pathname.startsWith("/admin/nexus/translations/");

  if (isTranslationsRoute) {
    return <>{children}</>;
  }

  return (
    <main style={{
      minHeight: "100vh",
      display: "grid",
      placeItems: "center",
      padding: 32,
      background: "#0f172a",
      color: "#e2e8f0",
      fontFamily: "Inter, Arial, sans-serif",
    }}>
      <section style={{ maxWidth: 680 }}>
        <p style={{ color: "#94a3b8", textTransform: "uppercase", letterSpacing: 0, fontSize: 12 }}>
          Module deprecie
        </p>
        <h1 style={{ margin: "8px 0 12px", fontSize: 32 }}>Nexus Games / MMO est desactive</h1>
        <p style={{ lineHeight: 1.6 }}>
          Les consoles Nexus, MMO, IA serveur et evenements sont masquees. Les modules actifs dans l'administration
          sont Battle IA, Quetes RP, Coop, Live et Traductions.
        </p>
        <Link href="/" style={{ color: "#67e8f9", display: "inline-block", marginTop: 16 }}>
          Retour admin
        </Link>
      </section>
    </main>
  );
}
