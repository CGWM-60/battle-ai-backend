import type { ReactNode } from "react";

export function MetricGrid({
  items,
}: {
  items: Array<{ label: string; value: ReactNode; tone?: "good" | "bad" | "neutral" }>;
}) {
  return (
    <section className="metric-grid">
      {items.map((item) => (
        <article className="metric-tile" key={item.label}>
          <span>{item.label}</span>
          <strong className={item.tone ?? "neutral"}>{item.value}</strong>
        </article>
      ))}
    </section>
  );
}
