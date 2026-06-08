export async function loadAdminData<T>(path: string): Promise<T> {
  const url = `/admin/api/${path}`;
  const response = await fetch(url, {
    credentials: "same-origin",
    headers: { Accept: "application/json" },
  });

  if (response.status === 401) {
    window.location.href = "/admin/login";
    throw new Error("admin authentication required");
  }

  const text = await response.text();

  if (!response.ok) {
    console.error(`[admin] HTTP ${response.status} for ${url}. Body prefix:`, text.substring(0, 300));
    throw new Error(`HTTP ${response.status}`);
  }

  try {
    return JSON.parse(text) as T;
  } catch (parseErr) {
    // This is the key diagnostic for "Unexpected non-whitespace character after JSON" type issues.
    // The actual body from Go (or proxy/ingress) is logged here.
    console.error(`[admin] Invalid JSON from ${url} (status ${response.status}). Parse error:`, parseErr);
    console.error(`[admin] Offending body (first 300 chars):`, text.substring(0, 300));
    throw new Error(`Invalid JSON from backend (status ${response.status}). See console for body prefix.`);
  }
}

export function formatNumber(value: number | null | undefined): string {
  return new Intl.NumberFormat("fr-FR").format(value ?? 0);
}

export function formatDate(value: string | null | undefined): string {
  if (!value) {
    return "-";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return new Intl.DateTimeFormat("fr-FR", {
    dateStyle: "short",
    timeStyle: "short",
  }).format(date);
}

export function usdMicros(value: number): string {
  return `$${(value / 1_000_000).toFixed(6)}`;
}

export function bytes(value: number): string {
  if (value < 1024) {
    return `${value} B`;
  }
  if (value < 1024 * 1024) {
    return `${(value / 1024).toFixed(1)} KB`;
  }
  if (value < 1024 * 1024 * 1024) {
    return `${(value / 1024 / 1024).toFixed(1)} MB`;
  }
  return `${(value / 1024 / 1024 / 1024).toFixed(1)} GB`;
}
