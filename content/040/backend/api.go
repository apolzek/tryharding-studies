package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type API struct {
	store *Store
	token string
}

func (a *API) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/ingest", a.requireAuth(a.ingest))
	mux.HandleFunc("/api/v1/users", a.listUsers)
	mux.HandleFunc("/api/v1/sessions", a.listSessions)
	mux.HandleFunc("/api/v1/session/", a.sessionDetail) // /api/v1/session/{id}
	mux.HandleFunc("/api/v1/samples", a.samples)
	mux.HandleFunc("/api/v1/report", a.report)
	mux.HandleFunc("/api/v1/health", a.health)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func (a *API) requireAuth(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h2 := r.Header.Get("Authorization")
		want := "Bearer " + a.token
		if h2 != want {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		h(w, r)
	}
}

// POST /api/v1/ingest
func (a *API) ingest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var s Sample
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if s.MachineID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "machine_id required"})
		return
	}
	id, err := a.store.Ingest(&s)
	if err != nil {
		log.Printf("ingest: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "ingest failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"session_id": id, "ok": true})
}

// GET /api/v1/users
func (a *API) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := a.store.ListUsers()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": users})
}

// GET /api/v1/sessions?user=foo&since=unix
func (a *API) listSessions(w http.ResponseWriter, r *http.Request) {
	user := r.URL.Query().Get("user")
	sinceStr := r.URL.Query().Get("since")
	since := time.Now().Add(-30 * 24 * time.Hour).Unix()
	if sinceStr != "" {
		if v, err := strconv.ParseInt(sinceStr, 10, 64); err == nil {
			since = v
		}
	}
	sessions, err := a.store.ListSessions(user, since)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if sessions == nil {
		sessions = []Session{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"sessions": sessions})
}

// GET /api/v1/session/{id}
func (a *API) sessionDetail(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/session/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	sess, err := a.store.GetSession(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, sess)
}

// GET /api/v1/samples?user=foo&session=123&from=...&to=...
func (a *API) samples(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	user := q.Get("user")
	sessID, _ := strconv.ParseInt(q.Get("session"), 10, 64)
	now := time.Now().Unix()
	from := now - 3600
	to := now
	if v, err := strconv.ParseInt(q.Get("from"), 10, 64); err == nil {
		from = v
	}
	if v, err := strconv.ParseInt(q.Get("to"), 10, 64); err == nil {
		to = v
	}
	samples, err := a.store.Samples(user, sessID, from, to)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if samples == nil {
		samples = []Sample{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"samples": samples})
}

// GET /api/v1/report?user=foo&from=...&to=...
// If from/to are not provided, defaults to the last 24h.
func (a *API) report(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	user := q.Get("user")
	if user == "" {
		http.Error(w, "user is required", http.StatusBadRequest)
		return
	}
	now := time.Now().Unix()
	from := now - ReportWindowSec
	to := now
	if v, err := strconv.ParseInt(q.Get("from"), 10, 64); err == nil {
		from = v
	}
	if v, err := strconv.ParseInt(q.Get("to"), 10, 64); err == nil {
		to = v
	}
	samples, err := a.store.Samples(user, 0, from, to)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	rep := BuildReport(user, samples, from, to)
	writeJSON(w, http.StatusOK, rep)
}

func (a *API) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "ts": time.Now().Unix()})
}
