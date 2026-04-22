// provisioner — long-running worker that reconciles tenants to kubernetes.
//
// Loop:
//   1. Claim one row from provision_jobs (FOR UPDATE SKIP LOCKED) → at-most-one
//      worker touches a given tenant at a time; horizontally safe.
//   2. Look up the tenant row to know desired state + credentials.
//   3. Run `helm upgrade --install tenant-stack` into namespace tenant-<id>
//      via the Helm Go SDK.
//   4. Mark tenant status=ready or failed.
//
// Why poll + SKIP LOCKED instead of LISTEN/NOTIFY? Poll is simpler to reason
// about under restarts, and at our rates (tenants-per-day) a 2s loop is fine.
// Later: Kubernetes leader election on the pod, not the job.
package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/obs-saas/shared/config"
	"github.com/obs-saas/shared/db"
	"github.com/obs-saas/shared/log"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type app struct {
	pool         *pgxpool.Pool
	chartPath    string
	ingestDomain string
	kubeconfig   string
	log          logger
}

type logger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	Debug(msg string, args ...any)
}

func main() {
	logger := log.New("provisioner")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := db.Connect(ctx, config.Must("OBS_DB_DSN"))
	if err != nil {
		logger.Error("db", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	a := &app{
		pool:         pool,
		chartPath:    config.Must("OBS_CHART_PATH"),
		ingestDomain: config.Get("OBS_INGEST_DOMAIN", "localtest.me"),
		kubeconfig:   config.Get("OBS_KUBECONFIG", ""),
		log:          logger,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	logger.Info("provisioner started", "chart", a.chartPath)

	t := time.NewTicker(2 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-stop:
			logger.Info("shutting down")
			return
		case <-t.C:
			if err := a.tick(ctx); err != nil {
				logger.Error("tick", "err", err)
			}
		}
	}
}

// tick runs one iteration of the reconciliation loop.
func (a *app) tick(ctx context.Context) error {
	job, err := a.claim(ctx)
	if err != nil {
		return err
	}
	if job == nil {
		return nil
	}
	a.log.Info("claimed job", "id", job.ID, "tenant", job.TenantID, "kind", job.Kind)

	runErr := a.processJob(ctx, job)
	a.finish(ctx, job, runErr)
	return runErr
}

type jobRow struct {
	ID       int64
	TenantID string
	Kind     string
	Attempts int
}

func (a *app) claim(ctx context.Context) (*jobRow, error) {
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var job jobRow
	err = tx.QueryRow(ctx, `
		SELECT id, tenant_id, kind, attempts
		FROM provision_jobs
		WHERE claimed_at IS NULL AND finished_at IS NULL
		ORDER BY created_at
		FOR UPDATE SKIP LOCKED
		LIMIT 1`).Scan(&job.ID, &job.TenantID, &job.Kind, &job.Attempts)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	_, err = tx.Exec(ctx, `UPDATE provision_jobs SET claimed_at=now(), attempts=attempts+1 WHERE id=$1`, job.ID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &job, nil
}

func (a *app) finish(ctx context.Context, job *jobRow, runErr error) {
	if runErr != nil {
		a.log.Error("job failed", "tenant", job.TenantID, "err", runErr)
		_, _ = a.pool.Exec(ctx, `UPDATE provision_jobs SET last_error=$1, claimed_at=NULL WHERE id=$2`,
			runErr.Error(), job.ID)
		_, _ = a.pool.Exec(ctx, `UPDATE tenants SET status='failed', error=$1, updated_at=now() WHERE id=$2`,
			runErr.Error(), job.TenantID)
		return
	}
	_, _ = a.pool.Exec(ctx, `UPDATE provision_jobs SET finished_at=now() WHERE id=$1`, job.ID)
}

// processJob dispatches on job.Kind.
func (a *app) processJob(ctx context.Context, job *jobRow) error {
	switch job.Kind {
	case "create", "upgrade":
		return a.installOrUpgrade(ctx, job.TenantID)
	case "delete":
		return a.uninstall(ctx, job.TenantID)
	default:
		return fmt.Errorf("unknown job kind %q", job.Kind)
	}
}

// tenantValues loads current tenant row → helm values overrides.
func (a *app) tenantValues(ctx context.Context, tenantID string) (map[string]any, string, error) {
	var ingestToken, grafanaPwd string
	err := a.pool.QueryRow(ctx, `
		SELECT ingest_token, grafana_password FROM tenants WHERE id=$1`, tenantID).
		Scan(&ingestToken, &grafanaPwd)
	if err != nil {
		return nil, "", err
	}
	jwtSecret := config.Must("OBS_JWT_SECRET")
	chPassword := randHex(16)

	vals := map[string]any{
		"tenant": map[string]any{
			"id":                   tenantID,
			"ingestDomain":         a.ingestDomain,
			"grafanaAdminPassword": grafanaPwd,
			"ingestJWT": map[string]any{
				"secret":      jwtSecret,
				"expectedTid": tenantID,
			},
		},
		"clickhouse": map[string]any{
			"password": chPassword,
		},
	}
	_ = ingestToken // not forwarded into the chart (it's a derivative)
	return vals, "ns-" + tenantID, nil
}

func (a *app) installOrUpgrade(ctx context.Context, tenantID string) error {
	ns := "tenant-" + tenantID
	if _, err := a.pool.Exec(ctx, `UPDATE tenants SET status='provisioning', updated_at=now() WHERE id=$1`, tenantID); err != nil {
		return err
	}

	vals, _, err := a.tenantValues(ctx, tenantID)
	if err != nil {
		return err
	}

	cfg, settings, err := a.helmConfig(ns)
	if err != nil {
		return err
	}

	ch, err := loader.Load(a.chartPath)
	if err != nil {
		return fmt.Errorf("load chart: %w", err)
	}
	_ = settings

	// Check if release exists → upgrade; otherwise install.
	listAct := action.NewList(cfg)
	listAct.AllNamespaces = false
	releases, err := listAct.Run()
	if err != nil {
		return fmt.Errorf("list releases: %w", err)
	}
	exists := false
	for _, r := range releases {
		if r.Name == "tenant" {
			exists = true
			break
		}
	}

	if exists {
		up := action.NewUpgrade(cfg)
		up.Namespace = ns
		up.Wait = false
		up.Timeout = 5 * time.Minute
		if _, err := up.Run("tenant", ch, vals); err != nil {
			return fmt.Errorf("helm upgrade: %w", err)
		}
	} else {
		in := action.NewInstall(cfg)
		in.ReleaseName = "tenant"
		in.Namespace = ns
		in.CreateNamespace = true
		in.Wait = false
		in.Timeout = 5 * time.Minute
		if _, err := in.Run(ch, vals); err != nil {
			return fmt.Errorf("helm install: %w", err)
		}
	}

	_, err = a.pool.Exec(ctx, `
		UPDATE tenants
		SET status='ready', chart_version=$1, updated_at=now(), error=NULL
		WHERE id=$2`, ch.Metadata.Version, tenantID)
	return err
}

func (a *app) uninstall(ctx context.Context, tenantID string) error {
	ns := "tenant-" + tenantID
	cfg, _, err := a.helmConfig(ns)
	if err != nil {
		return err
	}
	un := action.NewUninstall(cfg)
	un.Wait = false
	if _, err := un.Run("tenant"); err != nil && !strings.Contains(err.Error(), "not found") {
		return err
	}
	_, _ = a.pool.Exec(ctx, `UPDATE tenants SET status='deleted', updated_at=now() WHERE id=$1`, tenantID)
	return nil
}

func (a *app) helmConfig(namespace string) (*action.Configuration, *cli.EnvSettings, error) {
	settings := cli.New()
	if a.kubeconfig != "" {
		settings.KubeConfig = a.kubeconfig
	}
	settings.SetNamespace(namespace)

	cfg := new(action.Configuration)
	getter := genericclioptions.NewConfigFlags(false)
	ns := namespace
	getter.Namespace = &ns
	if a.kubeconfig != "" {
		kc := a.kubeconfig
		getter.KubeConfig = &kc
	}
	if err := cfg.Init(getter, namespace, "secret", func(fmt string, v ...interface{}) {
		a.log.Debug("helm", "msg", fmt)
	}); err != nil {
		return nil, nil, err
	}
	return cfg, settings, nil
}

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)[:n]
}
