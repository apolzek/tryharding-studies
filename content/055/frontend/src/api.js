const base = "/api";

async function req(path, options = {}) {
  const res = await fetch(base + path, {
    headers: { "Content-Type": "application/json" },
    ...options,
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`${res.status} ${res.statusText}: ${text}`);
  }
  return res.json();
}

export const api = {
  networks: () => req("/networks"),
  listTargets: () => req("/targets"),
  createTarget: (handle, networks) =>
    req("/targets", { method: "POST", body: JSON.stringify({ handle, networks }) }),
  deleteTarget: (id) => req(`/targets/${id}`, { method: "DELETE" }),
  scrapeNow: (id) => req(`/targets/${id}/scrape`, { method: "POST" }),
  events: (id, { limit = 100, sinceHours, network } = {}) => {
    const params = new URLSearchParams();
    params.set("limit", limit);
    if (sinceHours != null) params.set("since_hours", sinceHours);
    if (network) params.set("network", network);
    return req(`/targets/${id}/events?${params}`);
  },
  digest: (id, hours = 24) => req(`/targets/${id}/digest?hours=${hours}`),
  runs: (id) => req(`/targets/${id}/runs`),
};
