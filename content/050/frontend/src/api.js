const BASE = import.meta.env?.VITE_API_URL || '';

let authToken = localStorage.getItem('token') || null;

export function setToken(t) {
  authToken = t;
  if (t) localStorage.setItem('token', t);
  else localStorage.removeItem('token');
}

export function getToken() {
  return authToken;
}

async function request(path, { method = 'GET', body, headers = {} } = {}) {
  const opts = { method, headers: { ...headers } };
  if (body !== undefined) {
    opts.headers['Content-Type'] = 'application/json';
    opts.body = JSON.stringify(body);
  }
  if (authToken) opts.headers['Authorization'] = `Bearer ${authToken}`;
  const res = await fetch(`${BASE}/api${path}`, opts);
  const text = await res.text();
  const data = text ? JSON.parse(text) : null;
  if (!res.ok) {
    const err = new Error((data && data.error) || `HTTP ${res.status}`);
    err.status = res.status;
    err.body = data;
    throw err;
  }
  return data;
}

export const api = {
  register: (username, password, display_name) =>
    request('/auth/register', { method: 'POST', body: { username, password, display_name } }),
  login: (username, password) =>
    request('/auth/login', { method: 'POST', body: { username, password } }),

  me: () => request('/users/me'),
  updateMe: (patch) => request('/users/me', { method: 'PUT', body: patch }),
  getUser: (id) => request(`/users/${id}`),
  searchUsers: (q) => request(`/users/search?q=${encodeURIComponent(q)}`),

  listScraps: (userId) => request(`/scraps/${userId}`),
  postScrap: (userId, body) => request(`/scraps/${userId}`, { method: 'POST', body: { body } }),
  deleteScrap: (id) => request(`/scraps/${id}`, { method: 'DELETE' }),

  listTestimonials: (userId) => request(`/testimonials/${userId}`),
  postTestimonial: (userId, body) => request(`/testimonials/${userId}`, { method: 'POST', body: { body } }),
  deleteTestimonial: (id) => request(`/testimonials/${id}`, { method: 'DELETE' }),

  friendRequest: (addresseeId) => request(`/friends/request/${addresseeId}`, { method: 'POST' }),
  friendAccept: (requesterId) => request(`/friends/accept/${requesterId}`, { method: 'POST' }),
  friendRemove: (otherId) => request(`/friends/${otherId}`, { method: 'DELETE' }),
  friendList: (userId) => request(`/friends/list/${userId}`),
  friendPending: () => request('/friends/pending'),

  listCommunities: (q) => request(`/communities?q=${encodeURIComponent(q || '')}`),
  myCommunities: () => request('/communities/mine'),
  userCommunities: (userId) => request(`/communities/user/${userId}`),
  getCommunity: (id) => request(`/communities/${id}`),
  createCommunity: (payload) => request('/communities', { method: 'POST', body: payload }),
  joinCommunity: (id) => request(`/communities/${id}/join`, { method: 'POST' }),
  leaveCommunity: (id) => request(`/communities/${id}/leave`, { method: 'POST' }),

  rate: (userId, payload) => request(`/ratings/${userId}`, { method: 'PUT', body: payload }),
  ratingSummary: (userId) => request(`/ratings/${userId}/summary`),
  fansOf: (userId) => request(`/ratings/${userId}/fans`),

  listPhotos: (userId) => request(`/photos/${userId}`),
  addPhoto: (url, caption) => request('/photos', { method: 'POST', body: { url, caption } }),
  deletePhoto: (id) => request(`/photos/${id}`, { method: 'DELETE' }),

  trackVisit: (userId) => request(`/visits/${userId}`, { method: 'POST' }),
  listVisits: (userId) => request(`/visits/${userId}`)
};
