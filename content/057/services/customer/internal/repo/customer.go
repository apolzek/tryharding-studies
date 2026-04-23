package repo

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Customer struct {
	ID       string
	Name     string
	Email    string
	Document string
}

type CustomerRepo interface {
	Create(ctx context.Context, c *Customer) error
	Get(ctx context.Context, id string) (*Customer, error)
	List(ctx context.Context) ([]Customer, error)
}

type customerRepo struct{ pool *pgxpool.Pool }

func NewCustomerRepo(p *pgxpool.Pool) CustomerRepo { return &customerRepo{p} }

func (r *customerRepo) Create(ctx context.Context, c *Customer) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO customers (id, name, email, document) VALUES ($1,$2,$3,$4)`,
		c.ID, c.Name, c.Email, c.Document)
	return err
}

func (r *customerRepo) Get(ctx context.Context, id string) (*Customer, error) {
	row := r.pool.QueryRow(ctx, `SELECT id,name,email,document FROM customers WHERE id=$1`, id)
	c := &Customer{}
	if err := row.Scan(&c.ID, &c.Name, &c.Email, &c.Document); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return c, nil
}

func (r *customerRepo) List(ctx context.Context) ([]Customer, error) {
	rows, err := r.pool.Query(ctx, `SELECT id,name,email,document FROM customers ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Customer
	for rows.Next() {
		var c Customer
		if err := rows.Scan(&c.ID, &c.Name, &c.Email, &c.Document); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
	CREATE TABLE IF NOT EXISTS customers (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		email TEXT UNIQUE NOT NULL,
		document TEXT,
		created_at TIMESTAMPTZ DEFAULT now()
	);`)
	return err
}
