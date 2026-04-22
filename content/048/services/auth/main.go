// auth-service — registration, login, tenant provisioning enqueue.
//
// POST /register     {email, password}  → creates user + tenant + provision_job,
//                                         returns {tenant_id, ingest_token, grafana_password}
// POST /login        {email, password}  → 200 on ok (used to refresh UI state)
// GET  /tenants/:id                     → tenant status + URLs (internal, used by api-gateway)
//
// JWT secret is shared with the per-tenant auth-proxy. Grafana admin password
// is returned **once** at registration and also stored so the UI dashboard can
// show it — for the POC we keep it plaintext; prod should encrypt or show-once.
package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oklog/ulid/v2"

	"github.com/obs-saas/shared/config"
	"github.com/obs-saas/shared/db"
	jwtpkg "github.com/obs-saas/shared/jwt"
	"github.com/obs-saas/shared/log"
	"golang.org/x/crypto/bcrypt"
)

type server struct {
	pool   *pgxpool.Pool
	secret []byte
	log    logger
}

type logger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

func main() {
	logger := log.New("auth")
	ctx := context.Background()

	pool, err := db.Connect(ctx, config.Must("OBS_DB_DSN"))
	if err != nil {
		logger.Error("db", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	s := &server{
		pool:   pool,
		secret: []byte(config.Must("OBS_JWT_SECRET")),
		log:    logger,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	mux.HandleFunc("/register", s.handleRegister)
	mux.HandleFunc("/login", s.handleLogin)
	mux.HandleFunc("/tenants/", s.handleGetTenant)

	srv := &http.Server{
		Addr:              config.Get("OBS_LISTEN", ":8081"),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		logger.Info("listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutCtx)
}

// ---- handlers ----

type registerReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
type registerResp struct {
	TenantID        string `json:"tenant_id"`
	IngestToken     string `json:"ingest_token"`
	GrafanaPassword string `json:"grafana_password"`
	CollectorURL    string `json:"collector_url"`
	GrafanaURL      string `json:"grafana_url"`
}

func (s *server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	var req registerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", 400)
		return
	}
	if !validEmail(req.Email) || len(req.Password) < 12 {
		http.Error(w, "email invalid or password shorter than 12 chars", 400)
		return
	}

	resp, err := s.register(r.Context(), req.Email, req.Password)
	if err != nil {
		s.log.Warn("register", "email", req.Email, "err", err)
		if errors.Is(err, errEmailTaken) {
			http.Error(w, "email already registered", 409)
			return
		}
		http.Error(w, "internal error", 500)
		return
	}
	writeJSON(w, 201, resp)
}

var errEmailTaken = errors.New("email already registered")

// register is extracted from the handler so tests can drive it with a pgx pool.
func (s *server) register(ctx context.Context, email, password string) (*registerResp, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	pwHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	userID := newULID()
	tenantID := "t-" + newULID()

	ingestTok, err := jwtpkg.Issue(s.secret, tenantID, 0)
	if err != nil {
		return nil, err
	}
	grafanaPwd := randPassword(24)
	ingestDomain := config.Get("OBS_INGEST_DOMAIN", "localtest.me")
	collectorURL := "http://" + tenantID + "-ingest." + ingestDomain
	grafanaURL := "http://" + tenantID + "-grafana." + ingestDomain

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `INSERT INTO users(id, email, pw_hash) VALUES ($1,$2,$3)`, userID, email, string(pwHash))
	if err != nil {
		if isUniqueViolation(err) {
			return nil, errEmailTaken
		}
		return nil, err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO tenants(id, user_id, status, grafana_password, ingest_token, collector_url, grafana_url)
		VALUES ($1,$2,'pending',$3,$4,$5,$6)`,
		tenantID, userID, grafanaPwd, ingestTok, collectorURL, grafanaURL)
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO provision_jobs(tenant_id, kind) VALUES ($1,'create')`, tenantID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &registerResp{
		TenantID:        tenantID,
		IngestToken:     ingestTok,
		GrafanaPassword: grafanaPwd,
		CollectorURL:    collectorURL,
		GrafanaURL:      grafanaURL,
	}, nil
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", 400)
		return
	}
	var hash, userID string
	err := s.pool.QueryRow(r.Context(), `SELECT id, pw_hash FROM users WHERE email=$1`,
		strings.ToLower(strings.TrimSpace(req.Email))).Scan(&userID, &hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "invalid credentials", 401)
			return
		}
		s.log.Error("login query", "err", err)
		http.Error(w, "internal error", 500)
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)) != nil {
		http.Error(w, "invalid credentials", 401)
		return
	}
	// Return the user's tenant details for the dashboard.
	var tenantID, ingestTok, grafPwd, collURL, grafURL, status string
	err = s.pool.QueryRow(r.Context(), `
		SELECT id, ingest_token, grafana_password, collector_url, grafana_url, status
		FROM tenants WHERE user_id=$1 LIMIT 1`, userID).Scan(&tenantID, &ingestTok, &grafPwd, &collURL, &grafURL, &status)
	if err != nil {
		http.Error(w, "tenant not found", 500)
		return
	}
	writeJSON(w, 200, map[string]any{
		"tenant_id":        tenantID,
		"ingest_token":     ingestTok,
		"grafana_password": grafPwd,
		"collector_url":    collURL,
		"grafana_url":      grafURL,
		"status":           status,
	})
}

func (s *server) handleGetTenant(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/tenants/")
	if id == "" {
		http.Error(w, "id required", 400)
		return
	}
	var status, collURL, grafURL, errStr string
	err := s.pool.QueryRow(r.Context(), `
		SELECT status, collector_url, grafana_url, COALESCE(error,'') FROM tenants WHERE id=$1`, id).
		Scan(&status, &collURL, &grafURL, &errStr)
	if err != nil {
		http.Error(w, "not found", 404)
		return
	}
	writeJSON(w, 200, map[string]any{
		"status":        status,
		"collector_url": collURL,
		"grafana_url":   grafURL,
		"error":         errStr,
	})
}

// ---- helpers ----

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func newULID() string {
	return ulid.MustNew(ulid.Now(), rand.Reader).String()
}

func randPassword(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)[:n]
}

func validEmail(s string) bool {
	i := strings.IndexByte(s, '@')
	return i > 0 && i < len(s)-1 && strings.Contains(s[i:], ".")
}

func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "23505")
}
