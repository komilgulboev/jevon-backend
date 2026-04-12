package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"jevon/internal/models"

	"golang.org/x/crypto/bcrypt"
)

type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

// ─── Auth ──────────────────────────────────────────────────

// FindByLogin — сначала ищет по телефону, потом по email
func (r *UserRepo) FindByLogin(ctx context.Context, login string) (*models.User, error) {
	u, err := r.findByField(ctx, "phone", login)
	if err != nil {
		return nil, err
	}
	if u != nil {
		return u, nil
	}
	return r.findByField(ctx, "email", login)
}

// FindByEmail — оставляем для совместимости (используется в Refresh)
func (r *UserRepo) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	return r.findByField(ctx, "email", email)
}

func (r *UserRepo) findByField(ctx context.Context, field, value string) (*models.User, error) {
	var u models.User
	query := fmt.Sprintf(`
		SELECT u.id, u.role_id, r.name,
		       u.full_name, COALESCE(u.last_name,''),
		       COALESCE(u.email,''), u.password_hash,
		       COALESCE(u.phone,''), u.is_active,
		       COALESCE(u.avatar_url,''), u.created_at, u.updated_at,
		       COALESCE(u.whatsapp,''), COALESCE(u.telegram,''),
		       COALESCE(u.address,''),
		       u.salary, u.hourly_rate,
		       COALESCE(u.contract_type,'none')
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE u.%s = $1 AND u.is_active = true`, field)

	err := r.db.QueryRowContext(ctx, query, value).Scan(
		&u.ID, &u.RoleID, &u.RoleName,
		&u.FullName, &u.LastName,
		&u.Email, &u.PasswordHash,
		&u.Phone, &u.IsActive,
		&u.AvatarURL, &u.CreatedAt, &u.UpdatedAt,
		&u.WhatsApp, &u.Telegram, &u.Address,
		&u.Salary, &u.HourlyRate, &u.ContractType,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &u, err
}

// ─── CRUD ──────────────────────────────────────────────────

func (r *UserRepo) FindByID(ctx context.Context, id string) (*models.User, error) {
	var u models.User
	err := r.db.QueryRowContext(ctx, `
		SELECT u.id, u.role_id, r.name,
		       u.full_name, COALESCE(u.last_name,''),
		       COALESCE(u.email,''), COALESCE(u.phone,''),
		       u.is_active, COALESCE(u.avatar_url,''),
		       u.created_at, u.updated_at,
		       COALESCE(u.whatsapp,''), COALESCE(u.telegram,''),
		       COALESCE(u.address,''),
		       u.salary, u.hourly_rate,
		       COALESCE(u.contract_type,'none')
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE u.id = $1
	`, id).Scan(
		&u.ID, &u.RoleID, &u.RoleName,
		&u.FullName, &u.LastName,
		&u.Email, &u.Phone,
		&u.IsActive, &u.AvatarURL,
		&u.CreatedAt, &u.UpdatedAt,
		&u.WhatsApp, &u.Telegram, &u.Address,
		&u.Salary, &u.HourlyRate, &u.ContractType,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &u, err
}

func (r *UserRepo) List(ctx context.Context) ([]models.User, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT u.id, u.role_id, r.name,
		       u.full_name, COALESCE(u.last_name,''),
		       COALESCE(u.email,''), COALESCE(u.phone,''),
		       u.is_active, COALESCE(u.avatar_url,''),
		       u.created_at, u.updated_at,
		       COALESCE(u.whatsapp,''), COALESCE(u.telegram,''),
		       COALESCE(u.address,''),
		       u.salary, u.hourly_rate,
		       COALESCE(u.contract_type,'none')
		FROM users u
		JOIN roles r ON r.id = u.role_id
		ORDER BY u.full_name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		rows.Scan(
			&u.ID, &u.RoleID, &u.RoleName,
			&u.FullName, &u.LastName,
			&u.Email, &u.Phone,
			&u.IsActive, &u.AvatarURL,
			&u.CreatedAt, &u.UpdatedAt,
			&u.WhatsApp, &u.Telegram, &u.Address,
			&u.Salary, &u.HourlyRate, &u.ContractType,
		)
		users = append(users, u)
	}
	return users, nil
}

func (r *UserRepo) RoleList(ctx context.Context) ([]models.Role, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, COALESCE(description,'') FROM roles ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []models.Role
	for rows.Next() {
		var role models.Role
		rows.Scan(&role.ID, &role.Name, &role.Description)
		roles = append(roles, role)
	}
	return roles, nil
}

func (r *UserRepo) Create(ctx context.Context, req models.CreateUserRequest) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("bcrypt: %w", err)
	}
	contractType := req.ContractType
	if contractType == "" {
		contractType = "none"
	}
	var id string
	err = r.db.QueryRowContext(ctx, `
		INSERT INTO users
			(role_id, full_name, last_name, email, phone, password_hash,
			 whatsapp, telegram, address, salary, hourly_rate, contract_type)
		VALUES
			($1, $2, NULLIF($3,''), NULLIF($4,''), NULLIF($5,''), $6,
			 NULLIF($7,''), NULLIF($8,''), NULLIF($9,''), $10, $11, $12)
		RETURNING id
	`,
		req.RoleID,
		strings.TrimSpace(req.FullName),
		strings.TrimSpace(req.LastName),
		strings.TrimSpace(req.Email),
		strings.TrimSpace(req.Phone),
		string(hash),
		strings.TrimSpace(req.WhatsApp),
		strings.TrimSpace(req.Telegram),
		strings.TrimSpace(req.Address),
		req.Salary,
		req.HourlyRate,
		contractType,
	).Scan(&id)
	return id, err
}

func (r *UserRepo) Update(ctx context.Context, id string, req models.UpdateUserRequest) error {
	setClauses := []string{}
	args := []interface{}{}
	i := 1

	add := func(clause string, val interface{}) {
		setClauses = append(setClauses, fmt.Sprintf(clause, i))
		args = append(args, val)
		i++
	}

	if req.RoleID != nil       { add("role_id=$%d", *req.RoleID) }
	if req.FullName != nil     { add("full_name=$%d", strings.TrimSpace(*req.FullName)) }
	if req.LastName != nil     { add("last_name=$%d", strings.TrimSpace(*req.LastName)) }
	if req.Phone != nil        { add("phone=NULLIF($%d,'')", strings.TrimSpace(*req.Phone)) }
	if req.Email != nil        { add("email=NULLIF($%d,'')", strings.TrimSpace(*req.Email)) }
	if req.WhatsApp != nil     { add("whatsapp=NULLIF($%d,'')", strings.TrimSpace(*req.WhatsApp)) }
	if req.Telegram != nil     { add("telegram=NULLIF($%d,'')", strings.TrimSpace(*req.Telegram)) }
	if req.Address != nil      { add("address=$%d", strings.TrimSpace(*req.Address)) }
	if req.Salary != nil       { add("salary=$%d", *req.Salary) }
	if req.HourlyRate != nil   { add("hourly_rate=$%d", *req.HourlyRate) }
	if req.ContractType != nil { add("contract_type=$%d", *req.ContractType) }

	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		add("password_hash=$%d", string(hash))
	}

	if len(setClauses) == 0 {
		return nil
	}

	args = append(args, id)
	query := fmt.Sprintf(
		"UPDATE users SET %s, updated_at=NOW() WHERE id=$%d",
		strings.Join(setClauses, ", "), i,
	)
	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("not found")
	}
	return nil
}

func (r *UserRepo) ToggleActive(ctx context.Context, id string) (bool, error) {
	var isActive bool
	err := r.db.QueryRowContext(ctx, `
		UPDATE users SET is_active = NOT is_active WHERE id = $1 RETURNING is_active
	`, id).Scan(&isActive)
	return isActive, err
}

func (r *UserRepo) UpdateAvatar(ctx context.Context, userID, avatarURL string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET avatar_url=$1 WHERE id=$2`, avatarURL, userID)
	return err
}

// ─── Токены ────────────────────────────────────────────────

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