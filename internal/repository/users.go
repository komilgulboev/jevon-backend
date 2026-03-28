package repository

import (
	"context"
	"database/sql"
	"fmt"

	"jevon/internal/models"

	"golang.org/x/crypto/bcrypt"
)

type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var u models.User
	err := r.db.QueryRowContext(ctx, `
		SELECT u.id, u.role_id, r.name, u.full_name, u.email,
		       u.password_hash, COALESCE(u.phone,''), u.is_active,
		       COALESCE(u.avatar_url,''), u.created_at, u.updated_at
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE u.email = $1 AND u.is_active = true
	`, email).Scan(
		&u.ID, &u.RoleID, &u.RoleName, &u.FullName, &u.Email,
		&u.PasswordHash, &u.Phone, &u.IsActive, &u.AvatarURL,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &u, err
}

func (r *UserRepo) FindByID(ctx context.Context, id string) (*models.User, error) {
	var u models.User
	err := r.db.QueryRowContext(ctx, `
		SELECT u.id, u.role_id, r.name, u.full_name, u.email,
		       COALESCE(u.phone,''), u.is_active,
		       COALESCE(u.avatar_url,''), u.created_at, u.updated_at
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE u.id = $1
	`, id).Scan(
		&u.ID, &u.RoleID, &u.RoleName, &u.FullName, &u.Email,
		&u.Phone, &u.IsActive, &u.AvatarURL,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &u, err
}

func (r *UserRepo) List(ctx context.Context) ([]models.User, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT u.id, u.role_id, r.name, u.full_name, u.email,
		       COALESCE(u.phone,''), u.is_active,
		       COALESCE(u.avatar_url,''), u.created_at, u.updated_at
		FROM users u
		JOIN roles r ON r.id = u.role_id
		ORDER BY u.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		rows.Scan(&u.ID, &u.RoleID, &u.RoleName, &u.FullName, &u.Email,
			&u.Phone, &u.IsActive, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt)
		users = append(users, u)
	}
	return users, nil
}

func (r *UserRepo) Create(ctx context.Context, req models.CreateUserRequest) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("bcrypt: %w", err)
	}
	var id string
	err = r.db.QueryRowContext(ctx, `
		INSERT INTO users (role_id, full_name, email, password_hash, phone)
		VALUES ($1, $2, $3, $4, $5) RETURNING id
	`, req.RoleID, req.FullName, req.Email, string(hash), req.Phone).Scan(&id)
	return id, err
}

func (r *UserRepo) ToggleActive(ctx context.Context, id string) (bool, error) {
	var isActive bool
	err := r.db.QueryRowContext(ctx, `
		UPDATE users SET is_active = NOT is_active WHERE id = $1 RETURNING is_active
	`, id).Scan(&isActive)
	return isActive, err
}

func (r *UserRepo) StoreRefreshToken(ctx context.Context, userID, token string, ttl interface{}) error {
	hash, _ := bcrypt.GenerateFromPassword([]byte(token), bcrypt.MinCost)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, NOW() + $3::interval)
	`, userID, string(hash), ttl)
	return err
}

func (r *UserRepo) ValidateRefreshToken(ctx context.Context, userID, token string) (bool, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT token_hash FROM refresh_tokens
		WHERE user_id = $1 AND expires_at > NOW()
	`, userID)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var hash string
		rows.Scan(&hash)
		if bcrypt.CompareHashAndPassword([]byte(hash), []byte(token)) == nil {
			return true, nil
		}
	}
	return false, nil
}

func (r *UserRepo) DeleteRefreshTokens(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM refresh_tokens WHERE user_id = $1`, userID)
	return err
}
