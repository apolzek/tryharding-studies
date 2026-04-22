package control

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/apolzek/trynet/internal/crypto"
	"github.com/apolzek/trynet/internal/protocol"
)

// Server wires up all HTTP routes that the control plane exposes.
type Server struct {
	store *Store
	mux   *http.ServeMux
	log   *log.Logger
}

// New returns a Server whose http.Handler is ready to serve.
func New(s *Store, lg *log.Logger) *Server {
	if lg == nil {
		lg = log.Default()
	}
	srv := &Server{store: s, mux: http.NewServeMux(), log: lg}
	srv.routes()
	return srv
}

// Handler returns the root HTTP handler.
func (s *Server) Handler() http.Handler { return s.mux }

func (s *Server) routes() {
	s.mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	// Machine-facing endpoints.
	s.mux.HandleFunc("/machine/register", s.handleRegister)
	s.mux.HandleFunc("/machine/map", s.handleMap)
	s.mux.HandleFunc("/machine/endpoints", s.handleEndpoints)
	s.mux.HandleFunc("/machine/logout", s.handleLogout)

	// Admin-facing endpoints (UI talks to these).
	s.mux.HandleFunc("/admin/nodes", s.adminOnly(s.handleAdminNodes))
	s.mux.HandleFunc("/admin/nodes/", s.adminOnly(s.handleAdminNodeItem))
	s.mux.HandleFunc("/admin/keys", s.adminOnly(s.handleAdminKeys))
	s.mux.HandleFunc("/admin/keys/", s.adminOnly(s.handleAdminKeyItem))
	s.mux.HandleFunc("/admin/acl", s.adminOnly(s.handleAdminACL))
	s.mux.HandleFunc("/admin/settings", s.adminOnly(s.handleAdminSettings))
}

// -----------------------------------------------------------------------------
// machine endpoints
// -----------------------------------------------------------------------------

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	var req protocol.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ak, ok := s.store.LookupKey(req.AuthKey)
	if !ok {
		http.Error(w, "invalid auth key", http.StatusUnauthorized)
		return
	}

	// Idempotent re-registration: if this node-key is already present, refresh it.
	if existing := s.store.NodeByKey(req.NodeKey); existing != nil {
		existing.LastSeen = time.Now()
		existing.Endpoints = req.Endpoints
		existing.OS = req.OS
		if err := s.store.UpsertNode(existing); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.writeJSON(w, protocol.RegisterResponse{
			NodeID:    existing.ID,
			TailnetIP: existing.TailnetIP,
			NetMask:   32,
			Expiry:    existing.Expiry,
		})
		return
	}

	ip, err := s.store.AllocateIP()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	settings := s.store.SettingsCopy()
	expiry := time.Now().Add(settings.DefaultKeyExpiry)

	routes, err := parsePrefixes(req.Routes)
	if err != nil {
		http.Error(w, "invalid routes: "+err.Error(), http.StatusBadRequest)
		return
	}

	n := &protocol.Node{
		ID:        "n-" + crypto.NewToken(6),
		Name:      sanitizeHostname(req.Hostname),
		User:      ak.User,
		Tags:      ak.Tags,
		NodeKey:   req.NodeKey,
		TailnetIP: ip,
		Endpoints: req.Endpoints,
		Routes:    routes,
		ExitNode:  req.ExitNode,
		OS:        req.OS,
		Created:   time.Now(),
		LastSeen:  time.Now(),
		Expiry:    expiry,
		Approved:  !settings.RequireApproval,
	}
	if err := s.store.UpsertNode(n); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.store.MarkKeyUsed(req.AuthKey, n.ID)

	s.log.Printf("register: %s (%s) -> %s", n.Name, n.NodeKey, n.TailnetIP)
	s.writeJSON(w, protocol.RegisterResponse{
		NodeID:    n.ID,
		TailnetIP: n.TailnetIP,
		NetMask:   32,
		Expiry:    n.Expiry,
	})
}

// handleMap is a long-poll. Clients pass their known netmap version; the
// handler returns immediately if the server's version is newer, otherwise it
// waits up to 30s for a wake-up.
func (s *Server) handleMap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	var req protocol.PollRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.Endpoints) > 0 {
		s.store.UpdateNodeEndpoints(req.NodeKey, req.Endpoints)
	}
	s.store.TouchNode(req.NodeKey)

	ch, cancel := s.store.Subscribe(req.NodeKey)
	defer cancel()

	// Fast path: state already newer than what client knows.
	if s.store.Version() > req.KnownVersion {
		nm, err := s.store.BuildNetMap(req.NodeKey)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		s.writeJSON(w, nm)
		return
	}

	select {
	case <-ch:
		nm, err := s.store.BuildNetMap(req.NodeKey)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		s.writeJSON(w, nm)
	case <-time.After(30 * time.Second):
		// keep-alive: return current map with same version so client can reuse
		nm, err := s.store.BuildNetMap(req.NodeKey)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		s.writeJSON(w, nm)
	case <-r.Context().Done():
		return
	}
}

func (s *Server) handleEndpoints(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	var req protocol.EndpointsReport
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.store.UpdateNodeEndpoints(req.NodeKey, req.Endpoints)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		NodeKey protocol.Key `json:"node_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	n := s.store.NodeByKey(req.NodeKey)
	if n == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	_ = s.store.DeleteNode(n.ID)
	w.WriteHeader(http.StatusNoContent)
}

// -----------------------------------------------------------------------------
// admin endpoints
// -----------------------------------------------------------------------------

func (s *Server) adminOnly(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		want := s.store.SettingsCopy().AdminToken
		got := r.Header.Get("X-Admin-Token")
		if want == "" || got != want {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

func (s *Server) handleAdminNodes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.writeJSON(w, s.store.ListNodes())
	default:
		http.Error(w, "method", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAdminNodeItem(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/admin/nodes/")
	switch r.Method {
	case http.MethodGet:
		n := s.store.NodeByID(id)
		if n == nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		s.writeJSON(w, n)
	case http.MethodPatch:
		var patch struct {
			Approved *bool `json:"approved,omitempty"`
			Tags     *[]string `json:"tags,omitempty"`
			ExitNode *bool `json:"exit_node,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		n := s.store.NodeByID(id)
		if n == nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if patch.Approved != nil {
			n.Approved = *patch.Approved
		}
		if patch.Tags != nil {
			n.Tags = *patch.Tags
		}
		if patch.ExitNode != nil {
			n.ExitNode = *patch.ExitNode
		}
		if err := s.store.UpsertNode(n); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.writeJSON(w, n)
	case http.MethodDelete:
		if err := s.store.DeleteNode(id); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAdminKeys(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.writeJSON(w, s.store.ListKeys())
	case http.MethodPost:
		var req struct {
			User     string        `json:"user"`
			Tags     []string      `json:"tags"`
			Reusable bool          `json:"reusable"`
			TTL      time.Duration `json:"ttl"` // in nanoseconds for JSON, ok for UI
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.TTL == 0 {
			req.TTL = 90 * 24 * time.Hour
		}
		k := &protocol.PreAuthKey{
			ID:       "k-" + crypto.NewToken(4),
			Secret:   "tskey-" + crypto.NewToken(16),
			User:     req.User,
			Tags:     req.Tags,
			Reusable: req.Reusable,
			Created:  time.Now(),
			Expires:  time.Now().Add(req.TTL),
		}
		if err := s.store.CreateKey(k); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.writeJSON(w, k)
	default:
		http.Error(w, "method", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAdminKeyItem(w http.ResponseWriter, r *http.Request) {
	secret := strings.TrimPrefix(r.URL.Path, "/admin/keys/")
	if r.Method == http.MethodDelete {
		s.store.DeleteKey(secret)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Error(w, "method", http.StatusMethodNotAllowed)
}

func (s *Server) handleAdminACL(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		acl := s.store.ACL()
		s.writeJSON(w, &acl)
	case http.MethodPut:
		var p protocol.ACLPolicy
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := s.store.SetACL(&p); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAdminSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cp := s.store.SettingsCopy()
		s.writeJSON(w, cp)
	case http.MethodPatch:
		var patch Settings
		if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err := s.store.UpdateSettings(func(cur *Settings) {
			if patch.TailnetName != "" {
				cur.TailnetName = patch.TailnetName
			}
			if patch.IPv4CIDR != "" {
				cur.IPv4CIDR = patch.IPv4CIDR
			}
			if patch.DNSSuffix != "" {
				cur.DNSSuffix = patch.DNSSuffix
			}
			if patch.DerpURL != "" {
				cur.DerpURL = patch.DerpURL
			}
			var zero protocol.Key
			if patch.DerpKey != zero {
				cur.DerpKey = patch.DerpKey
			}
			cur.RequireApproval = patch.RequireApproval
			if patch.DefaultKeyExpiry != 0 {
				cur.DefaultKeyExpiry = patch.DefaultKeyExpiry
			}
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method", http.StatusMethodNotAllowed)
	}
}

// -----------------------------------------------------------------------------
// helpers
// -----------------------------------------------------------------------------

func (s *Server) writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		s.log.Printf("encode: %v", err)
	}
}

func parsePrefixes(ss []string) ([]netip.Prefix, error) {
	out := make([]netip.Prefix, 0, len(ss))
	for _, s := range ss {
		p, err := netip.ParsePrefix(s)
		if err != nil {
			return nil, fmt.Errorf("%q: %w", s, err)
		}
		out = append(out, p)
	}
	return out, nil
}

func sanitizeHostname(h string) string {
	h = strings.TrimSpace(h)
	h = strings.ToLower(h)
	h = strings.ReplaceAll(h, " ", "-")
	h = strings.ReplaceAll(h, "_", "-")
	if h == "" {
		h = "node-" + crypto.NewToken(3)
	}
	return h
}

// Run starts HTTP (or HTTPS) on addr. certFile/keyFile optional.
func (s *Server) Run(ctx context.Context, addr, certFile, keyFile string) error {
	srv := &http.Server{
		Addr:         addr,
		Handler:      s.Handler(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 60 * time.Second, // long-poll needs headroom
		IdleTimeout:  120 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		if certFile != "" && keyFile != "" {
			s.log.Printf("control: listening https on %s", addr)
			errCh <- srv.ListenAndServeTLS(certFile, keyFile)
		} else {
			s.log.Printf("control: listening http on %s", addr)
			errCh <- srv.ListenAndServe()
		}
	}()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		return nil
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
