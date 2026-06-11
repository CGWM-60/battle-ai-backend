"use client";

import Link from "next/link";
import {
  Activity,
  BookOpen,
  BrainCircuit,
  Building2,
  Coins,
  RadioTower,
  Gauge,
  ListChecks,
  LogOut,
  Radio,
  Settings,
  Shield,
  Users,
  Zap,
} from "lucide-react";
import { useEffect, useState, type ReactNode } from "react";

type NavItem = {
  href: string;
  path: string;
  label: string;
  icon: typeof Gauge;
  exact?: boolean;
};

const navSections: { title: string; items: NavItem[] }[] = [
  {
    title: "Commandement",
    items: [
      { href: "/", path: "/admin/", label: "Vue generale", icon: Gauge, exact: true },
      { href: "/accounts/", path: "/admin/accounts/", label: "Comptes", icon: Users },
      { href: "/system/", path: "/admin/system/", label: "Systeme", icon: Activity },
      { href: "/usage/", path: "/admin/usage/", label: "IA & couts", icon: BrainCircuit },
    ],
  },
  {
    title: "Nexus World",
    items: [
      { href: "/nexus/world/", path: "/admin/nexus/world/", label: "Tour de controle", icon: RadioTower },
      { href: "/nexus/mmo/", path: "/admin/nexus/mmo/", label: "Console MMO", icon: Users, exact: true },
      { href: "/nexus/mmo/buildings", path: "/admin/nexus/mmo/buildings", label: "Batiments", icon: Building2 },
      { href: "/nexus/mmo/units", path: "/admin/nexus/mmo/units", label: "Unites", icon: Zap },
      { href: "/nexus/mmo/research", path: "/admin/nexus/mmo/research", label: "Recherches", icon: BookOpen },
      { href: "/nexus/mmo/game-config", path: "/admin/nexus/mmo/game-config", label: "Reglages systeme", icon: Settings },
      { href: "/nexus/seasonal-events", path: "/admin/nexus/seasonal-events", label: "Evenements", icon: Activity },
      { href: "/nexus-coin/", path: "/admin/nexus-coin/", label: "Nexus Coin", icon: Coins },
      { href: "/live/", path: "/admin/live/", label: "Live", icon: Radio },
      { href: "/quests/", path: "/admin/quests/", label: "Quetes", icon: ListChecks },
      { href: "/roleplay-quests/", path: "/admin/roleplay-quests/", label: "Quetes RP", icon: ListChecks },
      { href: "/tribunal-ai/", path: "/admin/tribunal-ai/", label: "Tribunal IA", icon: Shield },
      { href: "/nexus/ai-server/", path: "/admin/nexus/ai-server/", label: "IA serveur", icon: RadioTower, exact: true },
      { href: "/nexus/ai-server/cities", path: "/admin/nexus/ai-server/cities", label: "Villes IA", icon: Building2 },
      { href: "/nexus/ai-server/attacks", path: "/admin/nexus/ai-server/attacks", label: "Attaques", icon: Zap },
      { href: "/nexus/ai-server/broadcasts", path: "/admin/nexus/ai-server/broadcasts", label: "Broadcasts", icon: Radio },
      { href: "/nexus/ai-server/memory", path: "/admin/nexus/ai-server/memory", label: "Memoire", icon: BrainCircuit },
      { href: "/nexus/ai-server/prompts", path: "/admin/nexus/ai-server/prompts", label: "Prompts IA", icon: BrainCircuit },
      { href: "/nexus/ai-server/logs", path: "/admin/nexus/ai-server/logs", label: "Logs & couts", icon: Activity },
      { href: "/nexus/translations/", path: "/admin/nexus/translations/", label: "Traductions", icon: ListChecks },
    ],
  },
];

export function AdminShell({
  title,
  description,
  children,
}: {
  title: string;
  description: string;
  children: ReactNode;
}) {
  const [locationState, setLocationState] = useState({ pathname: "/admin/", flash: "", error: "" });

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    setLocationState({
      pathname: window.location.pathname,
      flash: params.get("flash") ?? "",
      error: params.get("error") ?? "",
    });
  }, []);

  return (
    <div className="admin-frame">
      <aside className="sidebar">
        <div className="brand">
          <Shield size={22} aria-hidden />
          <div>
            <strong>go-battle-ia</strong>
            <span>admin</span>
          </div>
        </div>
        <nav className="nav-list" aria-label="Navigation admin">
          {navSections.map((section) => (
            <div className="nav-section" key={section.title}>
              <span className="nav-section-title">{section.title}</span>
              {section.items.map((item) => {
                const Icon = item.icon;
                const active = item.exact
                  ? locationState.pathname === item.path || locationState.pathname === item.path.replace(/\/$/, "")
                  : locationState.pathname.startsWith(item.path);
                return (
                  <Link className={active ? "nav-link active" : "nav-link"} href={item.href} key={item.href}>
                    <Icon size={18} aria-hidden />
                    <span>{item.label}</span>
                  </Link>
                );
              })}
            </div>
          ))}
        </nav>
        <form action="/admin/logout" method="post" className="logout-form">
          <button className="icon-button" type="submit" title="Se deconnecter">
            <LogOut size={17} aria-hidden />
            <span>Deconnexion</span>
          </button>
        </form>
      </aside>

      <main className="content">
        <header className="page-header">
          <div>
            <p className="eyebrow">Pilotage backend</p>
            <h1>{title}</h1>
            <p>{description}</p>
          </div>
        </header>
        {locationState.flash ? <div className="alert ok">{locationState.flash}</div> : null}
        {locationState.error ? <div className="alert error">{locationState.error}</div> : null}
        {children}
      </main>
    </div>
  );
}
