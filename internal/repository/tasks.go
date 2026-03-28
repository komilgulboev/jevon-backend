package repository

import (
	"context"
	"database/sql"
	"fmt"

	"jevon/internal/models"
)

// ── Tasks ─────────────────────────────────────────────────

type TaskRepo struct {
	db *sql.DB
}

func NewTaskRepo(db *sql.DB) *TaskRepo {
	return &TaskRepo{db: db}
}

func (r *TaskRepo) List(ctx context.Context, userID, roleName, projectID, assignedTo, status string) ([]models.Task, error) {
	query := `
		SELECT t.id, t.project_id, p.title,
		       t.title, COALESCE(t.description,''), t.status, t.priority,
		       COALESCE(CAST(t.due_date AS TEXT),''),
		       CAST(t.created_at AS TEXT),
		       COALESCE(u.full_name,''),
		       COALESCE(CAST(t.assigned_to AS TEXT),'')
		FROM tasks t
		JOIN projects p ON p.id = t.project_id
		LEFT JOIN users u ON u.id = t.assigned_to
		WHERE 1=1`

	args := []interface{}{}
	n := 1

	if roleName == "master" || roleName == "assistant" {
		query += fmt.Sprintf(" AND CAST(t.assigned_to AS TEXT) = $%d", n)
		args = append(args, userID)
		n++
	}
	if projectID != "" {
		query += fmt.Sprintf(" AND CAST(t.project_id AS TEXT) = $%d", n)
		args = append(args, projectID)
		n++
	}
	if assignedTo != "" {
		query += fmt.Sprintf(" AND CAST(t.assigned_to AS TEXT) = $%d", n)
		args = append(args, assignedTo)
		n++
	}
	if status != "" {
		query += fmt.Sprintf(" AND t.status = $%d", n)
		args = append(args, status)
	}
	query += " ORDER BY t.due_date ASC NULLS LAST, t.created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.Task
	for rows.Next() {
		var t models.Task
		var dueDate sql.NullString
		rows.Scan(&t.ID, &t.ProjectID, &t.ProjectTitle,
			&t.Title, &t.Description, &t.Status, &t.Priority,
			&dueDate, &t.AssignedTo, &t.AssignedToName, &t.AssignedTo)
		if dueDate.Valid {
			t.DueDate = &dueDate.String
		}
		result = append(result, t)
	}
	return result, nil
}

func (r *TaskRepo) Create(ctx context.Context, req models.CreateTaskRequest) (string, error) {
	if req.Status == "" {
		req.Status = "todo"
	}
	if req.Priority == "" {
		req.Priority = "medium"
	}
	var id string
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO tasks (project_id, title, description, assigned_to, status, priority, due_date)
		VALUES ($1,$2,$3,NULLIF($4,'')::uuid,$5,$6,NULLIF($7,'')::date)
		RETURNING id
	`, req.ProjectID, req.Title, req.Description, req.AssignedTo,
		req.Status, req.Priority, req.DueDate,
	).Scan(&id)
	return id, err
}

func (r *TaskRepo) Update(ctx context.Context, id string, req models.UpdateTaskRequest) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE tasks SET
			title       = COALESCE($1, title),
			description = COALESCE($2, description),
			status      = COALESCE($3, status),
			priority    = COALESCE($4, priority),
			assigned_to = COALESCE(NULLIF($5,'')::uuid, assigned_to),
			due_date    = COALESCE(NULLIF($6,'')::date, due_date)
		WHERE id = $7
	`, req.Title, req.Description, req.Status, req.Priority,
		req.AssignedTo, req.DueDate, id)
	return err
}

func (r *TaskRepo) UpdateStatus(ctx context.Context, id, status string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE tasks SET status = $1 WHERE id = $2`, status, id)
	return err
}

func (r *TaskRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM tasks WHERE id = $1`, id)
	return err
}

// ── Dashboard ─────────────────────────────────────────────

type DashboardRepo struct {
	db *sql.DB
}

func NewDashboardRepo(db *sql.DB) *DashboardRepo {
	return &DashboardRepo{db: db}
}

func (r *DashboardRepo) Stats(ctx context.Context) (models.DashboardStats, error) {
	var s models.DashboardStats

	r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE status IN ('new','in_progress','on_hold')),
			COUNT(*) FILTER (WHERE status = 'done')
		FROM projects
	`).Scan(&s.ActiveProjects, &s.ProjectsDone)

	r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE status = 'in_progress'),
			COUNT(*) FILTER (WHERE status = 'done'),
			COUNT(*) FILTER (WHERE status != 'done' AND due_date < CURRENT_DATE)
		FROM tasks
	`).Scan(&s.TasksInProgress, &s.TasksDone, &s.TasksOverdue)

	r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM users WHERE is_active = true`,
	).Scan(&s.TotalEmployees)

	return s, nil
}
