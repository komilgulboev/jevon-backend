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
		SELECT
			p.id, p.project_number, p.title,
			COALESCE(p.client_id::text, '')    AS client_id,
			COALESCE(p.client_name, '')        AS client_name,
			COALESCE(p.client_phone, '')       AS client_phone,
			p.status, p.priority,
			COALESCE(CAST(p.deadline AS TEXT), '') AS deadline,
			COALESCE(CAST(p.created_by AS TEXT), '') AS created_by,
			CAST(p.created_at AS TEXT),
			COALESCE(p.notes, ''),
			COUNT(DISTINCT po.order_id) AS order_count
		FROM projects p
		LEFT JOIN project_orders po ON po.project_id = p.id
		WHERE p.status != 'cancelled'`

	args := []interface{}{}
	n := 1

	if status != "" {
		query += fmt.Sprintf(" AND p.status = $%d", n)
		args = append(args, status)
		n++
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
		rows.Scan(
			&p.ID, &p.ProjectNumber, &p.Title,
			&p.ClientID, &p.ClientName, &p.ClientPhone,
			&p.Status, &p.Priority,
			&p.Deadline, &p.CreatedBy, &p.CreatedAt,
			&p.Notes, &p.OrderCount,
		)
		result = append(result, p)
	}
	return result, nil
}

func (r *ProjectRepo) GetByID(ctx context.Context, id string) (*models.Project, error) {
	var p models.Project
	err := r.db.QueryRowContext(ctx, `
		SELECT
			p.id, p.project_number, p.title,
			COALESCE(p.client_id::text, '')    AS client_id,
			COALESCE(p.client_name, '')        AS client_name,
			COALESCE(p.client_phone, '')       AS client_phone,
			p.status, p.priority,
			COALESCE(CAST(p.deadline AS TEXT), '') AS deadline,
			COALESCE(CAST(p.created_by AS TEXT), '') AS created_by,
			CAST(p.created_at AS TEXT),
			COALESCE(p.notes, ''),
			COUNT(DISTINCT po.order_id) AS order_count
		FROM projects p
		LEFT JOIN project_orders po ON po.project_id = p.id
		WHERE p.id = $1
		GROUP BY p.id
	`, id).Scan(
		&p.ID, &p.ProjectNumber, &p.Title,
		&p.ClientID, &p.ClientName, &p.ClientPhone,
		&p.Status, &p.Priority,
		&p.Deadline, &p.CreatedBy, &p.CreatedAt,
		&p.Notes, &p.OrderCount,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &p, err
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
		INSERT INTO projects
			(title, description, client_id, client_name, client_phone,
			 status, priority, deadline, notes, created_by)
		VALUES
			($1, $2, NULLIF($3,'')::uuid, $4, $5,
			 $6, $7, NULLIF($8,'')::date, $9, $10)
		RETURNING id
	`, req.Title, req.Description,
		req.ClientID, req.ClientName, req.ClientPhone,
		req.Status, req.Priority, req.Deadline, req.Notes, createdBy,
	).Scan(&id)
	if err != nil {
		return "", err
	}

	// Добавляем заказы в проект
	for _, orderID := range req.OrderIDs {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO project_orders (project_id, order_id)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, id, orderID)
		if err != nil {
			return "", err
		}
	}

	return id, tx.Commit()
}

func (r *ProjectRepo) Update(ctx context.Context, id string, req models.UpdateProjectRequest) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE projects SET
			title        = COALESCE(NULLIF($1,''), title),
			status       = COALESCE(NULLIF($2,''), status),
			priority     = COALESCE(NULLIF($3,''), priority),
			deadline     = COALESCE(NULLIF($4,'')::date, deadline),
			client_name  = COALESCE(NULLIF($5,''), client_name),
			client_phone = COALESCE(NULLIF($6,''), client_phone),
			notes        = COALESCE($7, notes)
		WHERE id = $8
	`, req.Title, req.Status, req.Priority, req.Deadline,
		req.ClientName, req.ClientPhone, req.Notes, id)
	return err
}

func (r *ProjectRepo) SoftDelete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE projects SET status = 'cancelled' WHERE id = $1`, id)
	return err
}

// ── Заказы проекта ────────────────────────────────────────────

func (r *ProjectRepo) GetProjectOrders(ctx context.Context, projectID string) ([]models.ProjectOrder, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			o.id, o.order_number, o.order_type,
			COALESCE(o.title, ''),
			COALESCE(o.client_name, ''),
			COALESCE(o.client_phone, ''),
			o.status, o.payment_status,
			COALESCE(o.final_cost, 0),
			COALESCE(o.paid_amount, 0),
			COALESCE(CAST(o.deadline AS TEXT), ''),
			COALESCE(o.current_stage, ''),
			-- Прогресс: done этапы / всего этапов
			COALESCE(
				ROUND(
					COUNT(os2.id) FILTER (WHERE os2.status = 'done') * 100.0
					/ NULLIF(COUNT(os2.id), 0)
				)::int, 0
			) AS progress,
			COUNT(os2.id) AS total_stages,
			COUNT(os2.id) FILTER (WHERE os2.status = 'done') AS done_stages
		FROM project_orders po
		JOIN orders o ON o.id = po.order_id
		LEFT JOIN order_stages os2 ON os2.order_id = o.id
		WHERE po.project_id = $1
		GROUP BY o.id
		ORDER BY o.order_number
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.ProjectOrder
	for rows.Next() {
		var po models.ProjectOrder
		rows.Scan(
			&po.ID, &po.OrderNumber, &po.OrderType,
			&po.Title, &po.ClientName, &po.ClientPhone,
			&po.Status, &po.PaymentStatus,
			&po.FinalCost, &po.PaidAmount,
			&po.Deadline, &po.CurrentStage,
			&po.Progress, &po.TotalStages, &po.DoneStages,
		)
		result = append(result, po)
	}
	return result, nil
}

func (r *ProjectRepo) AddOrderToProject(ctx context.Context, projectID, orderID string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO project_orders (project_id, order_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, projectID, orderID)
	return err
}

func (r *ProjectRepo) RemoveOrderFromProject(ctx context.Context, projectID, orderID string) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM project_orders WHERE project_id = $1 AND order_id = $2
	`, projectID, orderID)
	return err
}