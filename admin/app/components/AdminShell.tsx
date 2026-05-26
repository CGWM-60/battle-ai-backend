"use client";

import Link from "next/link";
import {
  Activity,
  BrainCircuit,
  Gauge,
  ListChecks,
  LogOut,
  Radio,
  Shield,
  Users,
} from "lucide-react";
import { useEffect, useState, type ReactNode } from "react";

const navItems = [
  { href: "/admin/", label: "Vue generale", icon: Gauge },
  { href: "/admin/accounts/", label: "Comptes", icon: Users },
  { href: "/admin/system/", label: "Systeme", icon: Activity },
  { href: "/admin/usage/", label: "IA & couts", icon: BrainCircuit },
  { href: "/admin/quests/", label: "Quetes", icon: ListChecks },
  { href: "/admin/live/", label: "Live", icon: Radio },
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
          {navItems.map((item) => {
            const Icon = item.icon;
            const active =
              item.href === "/admin/"
                ? locationState.pathname === "/admin" || locationState.pathname === "/admin/"
                : locationState.pathname.startsWith(item.href);
            return (
              <Link className={active ? "nav-link active" : "nav-link"} href={item.href} key={item.href}>
                <Icon size={18} aria-hidden />
                <span>{item.label}</span>
              </Link>
            );
          })}
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
