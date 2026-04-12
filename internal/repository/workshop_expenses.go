package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"jevon/internal/models"
)

type WorkshopExpenseRepo struct {
	db *sql.DB
}

func NewWorkshopExpenseRepo(db *sql.DB) *WorkshopExpenseRepo {
	return &WorkshopExpenseRepo{db: db}
}

func (r *WorkshopExpenseRepo) List(ctx context.Context, from, to, category string) ([]models.WorkshopExpense, float64, error) {
	query := `
		SELECT e.id::text,
		       CAST(e.expense_date AS TEXT),
		       e.category,
		       COALESCE(e.description,''),
		       e.amount, e.method,
		       COALESCE(CAST(e.created_by AS TEXT),''),
		       COALESCE(u.full_name,''),
		       COALESCE(CAST(e.order_id AS TEXT),''),
		       COALESCE(o.order_number, 0),
		       COALESCE(o.title,''),
		       COALESCE(CAST(e.project_id AS TEXT),''),
		       COALESCE(p.title,''),
		       COALESCE(CAST(e.linked_user_id AS TEXT),''),
		       COALESCE(lu.full_name,''),
		       e.created_at
		FROM workshop_expenses e
		LEFT JOIN users u    ON u.id  = e.created_by
		LEFT JOIN orders o   ON o.id  = e.order_id
		LEFT JOIN projects p ON p.id  = e.project_id
		LEFT JOIN users lu   ON lu.id = e.linked_user_id
		WHERE 1=1`

	args := []interface{}{}
	n := 1

	if from != "" {
		query += fmt.Sprintf(" AND e.expense_date >= $%d", n)
		args = append(args, from); n++
	}
	if to != "" {
		query += fmt.Sprintf(" AND e.expense_date <= $%d", n)
		args = append(args, to); n++
	}
	if category != "" {
		query += fmt.Sprintf(" AND e.category = $%d", n)
		args = append(args, category); n++
	}

	query += " ORDER BY e.expense_date DESC, e.created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var result []models.WorkshopExpense
	var total float64
	for rows.Next() {
		var e models.WorkshopExpense
		rows.Scan(
			&e.ID, &e.ExpenseDate, &e.Category, &e.Description,
			&e.Amount, &e.Method, &e.CreatedBy, &e.CreatorName,
			&e.OrderID, &e.OrderNumber, &e.OrderTitle,
			&e.ProjectID, &e.ProjectTitle,
			&e.LinkedUserID, &e.LinkedUserName,
			&e.CreatedAt,
		)
		total += e.Amount
		result = append(result, e)
	}
	return result, total, nil
}

func (r *WorkshopExpenseRepo) Create(ctx context.Context, req models.CreateWorkshopExpenseRequest, createdBy string) (string, error) {
	if req.Method == "" {
		req.Method = "cash"
	}
	expDate := req.ExpenseDate
	if expDate == "" {
		expDate = "CURRENT_DATE"
	}

	var id string
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO workshop_expenses
			(expense_date, category, description, amount, method, created_by,
			 order_id, project_id, linked_user_id)
		VALUES
			(COALESCE(NULLIF($1,'')::date, CURRENT_DATE),
			 $2, $3, $4, $5, NULLIF($6,'')::uuid,
			 NULLIF($7,'')::uuid, NULLIF($8,'')::uuid, NULLIF($9,'')::uuid)
		RETURNING id
	`, req.ExpenseDate, req.Category, req.Description, req.Amount, req.Method, createdBy,
		req.OrderID, req.ProjectID, req.LinkedUserID,
	).Scan(&id)

	// Если привязан к заказу — дублируем в order_expenses
	if err == nil && req.OrderID != "" {
		r.db.ExecContext(ctx, `
			INSERT INTO order_expenses (order_id, name, amount, expense_date, description, method, created_by)
			VALUES ($1::uuid, $2, $3, COALESCE(NULLIF($4,'')::date, CURRENT_DATE), $5, $6, NULLIF($7,'')::uuid)
		`, req.OrderID, req.Category+": "+req.Description, req.Amount,
			req.ExpenseDate, req.Description, req.Method, createdBy)
	}

	return id, err
}

func (r *WorkshopExpenseRepo) Update(ctx context.Context, id string, req models.UpdateWorkshopExpenseRequest) error {
	setClauses := []string{}
	args := []interface{}{}
	i := 1

	if req.ExpenseDate != nil { setClauses = append(setClauses, fmt.Sprintf("expense_date=$%d::date", i)); args = append(args, *req.ExpenseDate); i++ }
	if req.Category    != nil { setClauses = append(setClauses, fmt.Sprintf("category=$%d",      i)); args = append(args, *req.Category);    i++ }
	if req.Description != nil { setClauses = append(setClauses, fmt.Sprintf("description=$%d",   i)); args = append(args, *req.Description); i++ }
	if req.Amount      != nil { setClauses = append(setClauses, fmt.Sprintf("amount=$%d",        i)); args = append(args, *req.Amount);      i++ }
	if req.Method      != nil { setClauses = append(setClauses, fmt.Sprintf("method=$%d",        i)); args = append(args, *req.Method);      i++ }

	if len(setClauses) == 0 {
		return nil
	}
	args = append(args, id)
	_, err := r.db.ExecContext(ctx,
		fmt.Sprintf("UPDATE workshop_expenses SET %s WHERE id=$%d", strings.Join(setClauses, ","), i),
		args...)
	return err
}

func (r *WorkshopExpenseRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM workshop_expenses WHERE id = $1`, id)
	return err
}

func (r *WorkshopExpenseRepo) Categories(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT category FROM workshop_expenses ORDER BY category
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cats []string
	for rows.Next() {
		var c string
		rows.Scan(&c)
		cats = append(cats, c)
	}
	return cats, nil
}

func (r *WorkshopExpenseRepo) Stats(ctx context.Context, from, to string) (float64, map[string]float64, error) {
	query := `
		SELECT category, SUM(amount)
		FROM workshop_expenses WHERE 1=1`
	args := []interface{}{}
	n := 1
	if from != "" { query += fmt.Sprintf(" AND expense_date >= $%d", n); args = append(args, from); n++ }
	if to   != "" { query += fmt.Sprintf(" AND expense_date <= $%d", n); args = append(args, to);   n++ }
	query += " GROUP BY category ORDER BY SUM(amount) DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()

	byCategory := map[string]float64{}
	var total float64
	for rows.Next() {
		var cat string
		var sum float64
		rows.Scan(&cat, &sum)
		byCategory[cat] = sum
		total += sum
	}
	return total, byCategory, nil
}

var _ = sql.ErrNoRows