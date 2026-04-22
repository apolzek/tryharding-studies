const API = import.meta.env.VITE_API_URL || "http://localhost:8054";

function authHeaders() {
  const token = localStorage.getItem("sre_token");
  return token ? { Authorization: `Bearer ${token}` } : {};
}

async function request(path, opts = {}) {
  const res = await fetch(`${API}${path}`, {
    ...opts,
    headers: {
      "Content-Type": "application/json",
      ...authHeaders(),
      ...(opts.headers || {}),
    },
  });
  const text = await res.text();
  const data = text ? JSON.parse(text) : null;
  if (!res.ok) {
    const msg = (data && data.error) || `HTTP ${res.status}`;
    throw new Error(msg);
  }
  return data;
}

export const api = {
  login(username, password) {
    return request("/api/auth/login", {
      method: "POST",
      body: JSON.stringify({ username, password }),
    });
  },
  listChallenges() {
    return request("/api/challenges");
  },
  getChallenge(slug) {
    return request(`/api/challenges/${slug}`);
  },
  startSession(slug) {
    return request("/api/sessions", {
      method: "POST",
      body: JSON.stringify({ slug }),
    });
  },
  verifySession(id) {
    return request(`/api/sessions/${id}/verify`, { method: "POST" });
  },
  stopSession(id) {
    return request(`/api/sessions/${id}/stop`, { method: "POST" });
  },
  admin: {
    listChallenges: () => request("/api/admin/challenges"),
    getChallenge: (id) => request(`/api/admin/challenges/${id}`),
    createChallenge: (body) =>
      request("/api/admin/challenges", {
        method: "POST",
        body: JSON.stringify(body),
      }),
    updateChallenge: (id, body) =>
      request(`/api/admin/challenges/${id}`, {
        method: "PUT",
        body: JSON.stringify(body),
      }),
    deleteChallenge: (id) =>
      request(`/api/admin/challenges/${id}`, { method: "DELETE" }),
    listSessions: () => request("/api/admin/sessions"),
  },
};
