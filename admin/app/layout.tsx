import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Admin go-battle-ia",
  description: "Console de pilotage go-battle-ia",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="fr">
      <body>{children}</body>
    </html>
  );
}
