// Package ui is the admin web console. Server-rendered HTML via html/template;
// the only JS is "none". The UI calls the control server's /admin/* API with
// a shared secret pulled from config.
package ui

import (
	"bytes"
	"context"
	"crypto/tls"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/apolzek/trynet/internal/protocol"
)

//go:embed templates/*.html
var tpls embed.FS

// Config drives the UI server.
type Config struct {
	Addr       string
	ControlURL string
	AdminToken string
	Insecure   bool
	UIUser     string // basic-auth user for browser access
	UIPass     string // basic-auth password
}

// Server is the UI HTTP server.
type Server struct {
	cfg   Config
	log   *log.Logger
	t     *template.Template
	http  *http.Client

	mu           sync.Mutex
	flashMessage string
	lastKey      string
}

// New returns a ready-to-serve UI server.
func New(cfg Config, lg *log.Logger) (*Server, error) {
	if lg == nil {
		lg = log.Default()
	}
	t, err := template.New("").Funcs(template.FuncMap{}).ParseFS(tpls, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.Insecure}}
	return &Server{
		cfg:  cfg,
		log:  lg,
		t:    t,
		http: &http.Client{Transport: tr, Timeout: 15 * time.Second},
	}, nil
}

// Run starts the HTTP server.
func (s *Server) Run(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/",              s.basicAuth(s.handleHome))
	mux.HandleFunc("/nodes",         s.basicAuth(s.handleNodes))
	mux.HandleFunc("/nodes/",        s.basicAuth(s.handleNodeAction))
	mux.HandleFunc("/keys",          s.basicAuth(s.handleKeys))
	mux.HandleFunc("/keys/create",   s.basicAuth(s.handleKeyCreate))
	mux.HandleFunc("/keys/",         s.basicAuth(s.handleKeyAction))
	mux.HandleFunc("/acls",          s.basicAuth(s.handleACLs))
	mux.HandleFunc("/settings",      s.basicAuth(s.handleSettings))

	srv := &http.Server{Addr: s.cfg.Addr, Handler: mux}
	errCh := make(chan error, 1)
	go func() {
		s.log.Printf("ui: listening on %s", s.cfg.Addr)
		errCh <- srv.ListenAndServe()
	}()
	select {
	case <-ctx.Done():
		sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(sctx)
		return nil
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func (s *Server) basicAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.cfg.UIUser == "" {
			next(w, r)
			return
		}
		u, p, ok := r.BasicAuth()
		if !ok || u != s.cfg.UIUser || p != s.cfg.UIPass {
			w.Header().Set("WWW-Authenticate", `Basic realm="trynet"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// -----------------------------------------------------------------------------
// page data + render
// -----------------------------------------------------------------------------

type pageData struct {
	Page           string
	Flash          string
	Error          string
	Nodes          []*protocol.Node
	Online         int
	Keys           []*protocol.PreAuthKey
	ACLJSON        string
	Settings       map[string]any
	JustCreatedKey string
}

func (s *Server) render(w http.ResponseWriter, name string, data *pageData) {
	data.Page = name
	s.mu.Lock()
	if s.flashMessage != "" {
		data.Flash = s.flashMessage
		s.flashMessage = ""
	}
	if s.lastKey != "" {
		data.JustCreatedKey = s.lastKey
		s.lastKey = ""
	}
	s.mu.Unlock()

	var buf bytes.Buffer
	tpl := template.Must(s.t.Clone())
	if _, err := tpl.ParseFS(tpls, "templates/"+name+".html"); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if err := tpl.ExecuteTemplate(&buf, "layout", data); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.Copy(w, &buf)
}

// -----------------------------------------------------------------------------
// handlers
// -----------------------------------------------------------------------------

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	nodes, _ := s.fetchNodes()
	keys, _ := s.fetchKeys()
	online := 0
	for _, n := range nodes {
		if n.Online() {
			online++
		}
	}
	s.render(w, "home", &pageData{Nodes: nodes, Keys: keys, Online: online})
}

func (s *Server) handleNodes(w http.ResponseWriter, r *http.Request) {
	nodes, err := s.fetchNodes()
	s.render(w, "nodes", &pageData{Nodes: nodes, Error: errStr(err)})
}

func (s *Server) handleNodeAction(w http.ResponseWriter, r *http.Request) {
	// /nodes/<id>/approve or /nodes/<id>/delete
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/nodes/"), "/")
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	id, action := parts[0], parts[1]
	switch action {
	case "approve":
		payload := map[string]any{"approved": true}
		_ = s.adminCall("PATCH", "/admin/nodes/"+id, payload, nil)
		s.flash("node approved")
	case "delete":
		_ = s.adminCall("DELETE", "/admin/nodes/"+id, nil, nil)
		s.flash("node removed")
	}
	http.Redirect(w, r, "/nodes", http.StatusSeeOther)
}

func (s *Server) handleKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := s.fetchKeys()
	s.render(w, "keys", &pageData{Keys: keys, Error: errStr(err)})
}

func (s *Server) handleKeyCreate(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	tags := splitCSV(r.FormValue("tags"))
	days, _ := strconv.Atoi(r.FormValue("ttl_days"))
	if days <= 0 {
		days = 90
	}
	req := map[string]any{
		"user":     r.FormValue("user"),
		"tags":     tags,
		"reusable": r.FormValue("reusable") != "",
		"ttl":      int64(days) * int64(24*time.Hour),
	}
	var key protocol.PreAuthKey
	if err := s.adminCall("POST", "/admin/keys", req, &key); err != nil {
		s.errFlash(err.Error())
	} else {
		s.mu.Lock()
		s.lastKey = key.Secret
		s.mu.Unlock()
	}
	http.Redirect(w, r, "/keys", http.StatusSeeOther)
}

func (s *Server) handleKeyAction(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/keys/"), "/")
	if len(parts) != 2 || parts[1] != "delete" {
		http.NotFound(w, r)
		return
	}
	_ = s.adminCall("DELETE", "/admin/keys/"+parts[0], nil, nil)
	s.flash("key revoked")
	http.Redirect(w, r, "/keys", http.StatusSeeOther)
}

func (s *Server) handleACLs(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		_ = r.ParseForm()
		var p protocol.ACLPolicy
		if err := json.Unmarshal([]byte(r.FormValue("policy")), &p); err != nil {
			s.errFlash("invalid JSON: " + err.Error())
			http.Redirect(w, r, "/acls", http.StatusSeeOther)
			return
		}
		if err := s.adminCall("PUT", "/admin/acl", p, nil); err != nil {
			s.errFlash(err.Error())
		} else {
			s.flash("policy saved")
		}
		http.Redirect(w, r, "/acls", http.StatusSeeOther)
		return
	}
	var p protocol.ACLPolicy
	if err := s.adminCall("GET", "/admin/acl", nil, &p); err != nil {
		s.render(w, "acls", &pageData{Error: err.Error()})
		return
	}
	b, _ := json.MarshalIndent(p, "", "  ")
	s.render(w, "acls", &pageData{ACLJSON: string(b)})
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		_ = r.ParseForm()
		patch := map[string]any{
			"tailnet_name":     r.FormValue("tailnet_name"),
			"ipv4_cidr":        r.FormValue("ipv4_cidr"),
			"dns_suffix":       r.FormValue("dns_suffix"),
			"derp_url":         r.FormValue("derp_url"),
			"require_approval": r.FormValue("require_approval") != "",
		}
		if err := s.adminCall("PATCH", "/admin/settings", patch, nil); err != nil {
			s.errFlash(err.Error())
		} else {
			s.flash("settings saved")
		}
		http.Redirect(w, r, "/settings", http.StatusSeeOther)
		return
	}
	var settings map[string]any
	err := s.adminCall("GET", "/admin/settings", nil, &settings)
	s.render(w, "settings", &pageData{Settings: settings, Error: errStr(err)})
}

// -----------------------------------------------------------------------------
// helpers
// -----------------------------------------------------------------------------

func (s *Server) fetchNodes() ([]*protocol.Node, error) {
	var out []*protocol.Node
	err := s.adminCall("GET", "/admin/nodes", nil, &out)
	return out, err
}

func (s *Server) fetchKeys() ([]*protocol.PreAuthKey, error) {
	var out []*protocol.PreAuthKey
	err := s.adminCall("GET", "/admin/keys", nil, &out)
	return out, err
}

func (s *Server) adminCall(method, path string, in, out any) error {
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, s.cfg.ControlURL+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Token", s.cfg.AdminToken)
	resp, err := s.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s %s: %s: %s", method, path, resp.Status, bytes.TrimSpace(b))
	}
	if out == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (s *Server) flash(msg string) {
	s.mu.Lock()
	s.flashMessage = msg
	s.mu.Unlock()
}

func (s *Server) errFlash(msg string) { s.flash("error: " + msg) }

func errStr(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func splitCSV(s string) []string {
	if s = strings.TrimSpace(s); s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
