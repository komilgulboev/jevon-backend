package repository

import (
	"context"
	"database/sql"
	"fmt"

	"jevon/internal/models"
)

type ProjectRepo struct {
	db *sql.DB
}

func NewProjectRepo(db *sql.DB) *ProjectRepo {
	return &ProjectRepo{db: db}
}

func (r *ProjectRepo) List(ctx context.Context, userID, roleName, status string) ([]models.Project, error) {
	query := `
		SELECT p.id, p.project_number, p.title,
		       COALESCE(p.client_name,''), COALESCE(p.client_phone,''),
		       p.status, p.priority,
		       CAST(p.deadline AS TEXT),
		       COALESCE(CAST(p.created_by AS TEXT),''),
		       CAST(p.created_at AS TEXT),
		       COALESCE(p.current_stage,''),
		       COALESCE(ROUND(
		           COUNT(t.id) FILTER (WHERE t.status='done') * 100.0
		           / NULLIF(COUNT(t.id),0)
		       )::int, 0) AS progress,
		       COUNT(t.id) AS total_tasks
		FROM projects p
		LEFT JOIN tasks t ON t.project_id = p.id
		WHERE p.status != 'cancelled'`

	args := []interface{}{}
	n := 1

	if roleName == "master" || roleName == "assistant" {
		query += fmt.Sprintf(" AND p.id IN (SELECT project_id FROM project_members WHERE user_id = $%d)", n)
		args = append(args, userID)
		n++
	}
	if status != "" {
		query += fmt.Sprintf(" AND p.status = $%d", n)
		args = append(args, status)
	}
	query += " GROUP BY p.id ORDER BY p.project_number DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.Project
	for rows.Next() {
		var p models.Project
		var deadline, createdBy, createdAt, currentStage sql.NullString
		rows.Scan(
			&p.ID, &p.ProjectNumber, &p.Title,
			&p.ClientName, &p.ClientPhone,
			&p.Status, &p.Priority,
			&deadline, &createdBy, &createdAt,
			&currentStage,
			&p.Progress, &p.TotalTasks,
		)
		if deadline.Valid {
			p.Deadline = &deadline.String
		}
		p.CreatedBy = createdBy.String
		p.CurrentStage = currentStage.String
		result = append(result, p)
	}
	return result, nil
}

func (r *ProjectRepo) Create(ctx context.Context, req models.CreateProjectRequest, createdBy string) (string, error) {
	if req.Status == "" {
		req.Status = "new"
	}
	if req.Priority == "" {
		req.Priority = "medium"
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	var id string
	err = tx.QueryRowContext(ctx, `
		INSERT INTO projects (title, description, client_name, client_phone, status, priority, deadline, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,NULLIF($7,'')::date,$8) RETURNING id
	`, req.Title, req.Description, req.ClientName, req.ClientPhone,
		req.Status, req.Priority, req.Deadline, createdBy,
	).Scan(&id)
	if err != nil {
		return "", err
	}

	for _, uid := range req.MemberIDs {
		tx.ExecContext(ctx, `
			INSERT INTO project_members (project_id, user_id) VALUES ($1,$2) ON CONFLICT DO NOTHING
		`, id, uid)
	}
	tx.ExecContext(ctx, `
		INSERT INTO project_members (project_id, user_id) VALUES ($1,$2) ON CONFLICT DO NOTHING
	`, id, createdBy)

	return id, tx.Commit()
}

func (r *ProjectRepo) Update(ctx context.Context, id string, req models.UpdateProjectRequest) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE projects SET
			title        = COALESCE($1, title),
			status       = COALESCE($2, status),
			priority     = COALESCE($3, priority),
			deadline     = COALESCE(NULLIF($4,'')::date, deadline),
			client_name  = COALESCE($5, client_name),
			client_phone = COALESCE($6, client_phone)
		WHERE id = $7
	`, req.Title, req.Status, req.Priority, req.Deadline,
		req.ClientName, req.ClientPhone, id)
	return err
}

func (r *ProjectRepo) SoftDelete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE projects SET status = 'cancelled' WHERE id = $1`, id)
	return err
}