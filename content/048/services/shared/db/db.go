// Package db wires a pgx pool and runs the tenant-saas schema migrations.
// Migrations are idempotent CREATE IF NOT EXISTS — enough for a POC.
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const Schema = `
CREATE TABLE IF NOT EXISTS users (
  id          TEXT PRIMARY KEY,
  email       TEXT NOT NULL UNIQUE,
  pw_hash     TEXT NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS tenants (
  id               TEXT PRIMARY KEY,
  user_id          TEXT NOT NULL REFERENCES users(id),
  status           TEXT NOT NULL,   -- pending | provisioning | ready | failed | deleting
  chart_version    TEXT,
  grafana_password TEXT,            -- shown once; stored for dashboard display (POC — encrypt in prod)
  ingest_token     TEXT NOT NULL,
  collector_url    TEXT,
  grafana_url      TEXT,
  error            TEXT,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS tenants_status_idx ON tenants(status);

-- outbox pattern: provisioner polls for pending rows
CREATE TABLE IF NOT EXISTS provision_jobs (
  id          BIGSERIAL PRIMARY KEY,
  tenant_id   TEXT NOT NULL REFERENCES tenants(id),
  kind        TEXT NOT NULL,  -- create | upgrade | delete
  claimed_at  TIMESTAMPTZ,
  finished_at TIMESTAMPTZ,
  attempts    INT NOT NULL DEFAULT 0,
  last_error  TEXT,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS provision_jobs_unclaimed_idx
  ON provision_jobs(created_at) WHERE claimed_at IS NULL;
`

func Connect(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = 20
	cfg.MinConns = 2
	cfg.MaxConnLifetime = 30 * time.Minute

	// Retry initial connect — compose may race the app vs postgres.
	deadline := time.Now().Add(30 * time.Second)
	var pool *pgxpool.Pool
	for {
		pool, err = pgxpool.NewWithConfig(ctx, cfg)
		if err == nil {
			if err = pool.Ping(ctx); err == nil {
				break
			}
			pool.Close()
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("db: connect: %w", err)
		}
		time.Sleep(time.Second)
	}
	if _, err := pool.Exec(ctx, Schema); err != nil {
		return nil, fmt.Errorf("db: schema: %w", err)
	}
	return pool, nil
}
