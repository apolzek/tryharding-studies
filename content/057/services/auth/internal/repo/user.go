package repo

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID       string
	Email    string
	Password string
}

type UserRepo interface {
	Create(ctx context.Context, u *User) error
	FindByEmail(ctx context.Context, email string) (*User, error)
}

type userRepo struct{ pool *pgxpool.Pool }

func NewUserRepo(p *pgxpool.Pool) UserRepo { return &userRepo{p} }

func (r *userRepo) Create(ctx context.Context, u *User) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO users (id, email, password) VALUES ($1,$2,$3)`,
		u.ID, u.Email, u.Password)
	return err
}

func (r *userRepo) FindByEmail(ctx context.Context, email string) (*User, error) {
	row := r.pool.QueryRow(ctx, `SELECT id, email, password FROM users WHERE email=$1`, email)
	u := &User{}
	if err := row.Scan(&u.ID, &u.Email, &u.Password); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return u, nil
}

func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		email TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL,
		created_at TIMESTAMPTZ DEFAULT now()
	);`)
	return err
}
