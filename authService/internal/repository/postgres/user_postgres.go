package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/yourusername/user-service/internal/config"
	domain "github.com/yourusername/user-service/internal/domains"
	"go.opentelemetry.io/otel"
	"log"

	_ "github.com/lib/pq"
)

type UserRepositoryInt interface {
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByID(id int64) (*domain.User, error)
	Create(user *domain.User) error
	Close() error
}

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(cfg *config.Config) (UserRepositoryInt, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Database connected successfully")
	return &UserRepository{db: db}, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	tracer := otel.Tracer("user-service")
	ctx, span := tracer.Start(ctx, "FindByEmail")
	defer span.End()

	query := `SELECT id, email, name, password, created_at, updated_at 
              FROM users WHERE email = $1`

	var user domain.User
	err := r.db.QueryRow(query, email).Scan(
		&user.ID, &user.Email, &user.Name, &user.Password, &user.CreatedAt, &user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByID(id int64) (*domain.User, error) {
	query := `SELECT id, email, name, created_at, updated_at 
              FROM users WHERE id = $1`

	var user domain.User
	err := r.db.QueryRow(query, id).Scan(
		&user.ID, &user.Email, &user.Name, &user.CreatedAt, &user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) Create(user *domain.User) error {
	query := `INSERT INTO users (email, name, password, created_at, updated_at)
			  VALUES ($1, $2, $3, NOW(), NOW()) RETURNING id, created_at, updated_at`

	err := r.db.QueryRow(query, user.Email, user.Name, user.Password).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
	return err
}

func (r *UserRepository) Close() error {
	return r.db.Close()
}
