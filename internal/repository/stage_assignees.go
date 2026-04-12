package repository

import (
	"context"
	"database/sql"
	"fmt"

	"jevon/internal/models"
)

type StageAssigneeRepo struct {
	db *sql.DB
}

func NewStageAssigneeRepo(db *sql.DB) *StageAssigneeRepo {
	return &StageAssigneeRepo{db: db}
}

// ── Этапы проектов ────────────────────────────────────────────

func (r *StageAssigneeRepo) ProjectStageAssignees(ctx context.Context, stageID string) ([]models.StageAssignee, error) {
	return r.queryAssignees(ctx, "project_stage_assignees", stageID)
}

func (r *StageAssigneeRepo) ProjectStageSync(ctx context.Context, stageID, assignedBy string, userIDs []string, percents []float64) error {
	return r.syncAssignees(ctx, "project_stage_assignees", stageID, assignedBy, userIDs, percents)
}

// ── Этапы заказов ─────────────────────────────────────────────

func (r *StageAssigneeRepo) OrderStageAssignees(ctx context.Context, stageID string) ([]models.StageAssignee, error) {
	return r.queryAssignees(ctx, "order_stage_assignees", stageID)
}

func (r *StageAssigneeRepo) OrderStageSync(ctx context.Context, stageID, assignedBy string, userIDs []string, percents []float64) error {
	return r.syncAssignees(ctx, "order_stage_assignees", stageID, assignedBy, userIDs, percents)
}

// ── Общие методы ─────────────────────────────────────────────

func (r *StageAssigneeRepo) queryAssignees(ctx context.Context, table, stageID string) ([]models.StageAssignee, error) {
	query := `
		SELECT a.id::text, a.stage_id::text, a.user_id::text,
		       CONCAT(u.full_name, CASE WHEN u.last_name IS NOT NULL AND u.last_name != '' THEN ' ' || u.last_name ELSE '' END),
		       ro.name,
		       COALESCE(u.avatar_url,''),
		       COALESCE(a.assembly_percent, 0),
		       a.assigned_at
		FROM ` + table + ` a
		JOIN users u  ON u.id = a.user_id
		JOIN roles ro ON ro.id = u.role_id
		WHERE a.stage_id = $1
		ORDER BY a.assigned_at ASC`

	rows, err := r.db.QueryContext(ctx, query, stageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.StageAssignee
	for rows.Next() {
		var a models.StageAssignee
		rows.Scan(&a.ID, &a.StageID, &a.UserID, &a.FullName, &a.RoleName,
			&a.AvatarURL, &a.AssemblyPercent, &a.AssignedAt)
		result = append(result, a)
	}
	return result, nil
}

func (r *StageAssigneeRepo) syncAssignees(ctx context.Context, table, stageID, assignedBy string, userIDs []string, percents []float64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE stage_id = $1`, stageID)
	if err != nil {
		return err
	}

	for i, uid := range userIDs {
		if uid == "" {
			continue
		}
		percent := 0.0
		if i < len(percents) {
			percent = percents[i]
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO `+table+` (stage_id, user_id, assigned_by, assembly_percent)
			VALUES ($1::uuid, $2::uuid, NULLIF($3,'')::uuid, $4)
			ON CONFLICT (stage_id, user_id) DO UPDATE SET assembly_percent = EXCLUDED.assembly_percent
		`, stageID, uid, assignedBy, percent)
		if err != nil {
			return err
		}
	}

	mainTable := "project_stages"
	if table == "order_stage_assignees" {
		mainTable = "order_stages"
	}
	if len(userIDs) > 0 && userIDs[0] != "" {
		tx.ExecContext(ctx, `UPDATE `+mainTable+` SET assigned_to=$1::uuid WHERE id=$2::uuid`,
			userIDs[0], stageID)
	} else {
		tx.ExecContext(ctx, `UPDATE `+mainTable+` SET assigned_to=NULL WHERE id=$1::uuid`, stageID)
	}

	return tx.Commit()
}

// ── Табель ────────────────────────────────────────────────────

func (r *StageAssigneeRepo) TimesheetList(ctx context.Context, from, to, userID string) ([]models.Timesheet, error) {
	query := `
		SELECT t.id::text, t.user_id::text,
		       CONCAT(u.full_name, CASE WHEN u.last_name IS NOT NULL AND u.last_name != '' THEN ' ' || u.last_name ELSE '' END),
		       ro.name,
		       CAST(t.work_date AS TEXT),
		       t.hours,
		       COALESCE(CAST(t.check_in AS TEXT),''),
		       COALESCE(CAST(t.check_out AS TEXT),''),
		       COALESCE(t.source_type,''),
		       COALESCE(CAST(t.source_id AS TEXT),''),
		       COALESCE(t.notes,''),
		       t.created_at
		FROM timesheets t
		JOIN users u  ON u.id  = t.user_id
		JOIN roles ro ON ro.id = u.role_id
		WHERE 1=1`

	args := []interface{}{}
	n := 1

	if from != "" {
		query += ` AND t.work_date >= $` + itoa(n) + `::date`
		args = append(args, from)
		n++
	}
	if to != "" {
		query += ` AND t.work_date <= $` + itoa(n) + `::date`
		args = append(args, to)
		n++
	}
	if userID != "" {
		query += ` AND t.user_id = $` + itoa(n) + `::uuid`
		args = append(args, userID)
		n++
	}

	query += " ORDER BY t.work_date DESC, u.full_name"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.Timesheet
	for rows.Next() {
		var ts models.Timesheet
		rows.Scan(&ts.ID, &ts.UserID, &ts.FullName, &ts.RoleName,
			&ts.WorkDate, &ts.Hours, &ts.CheckIn, &ts.CheckOut,
			&ts.SourceType, &ts.SourceID, &ts.Notes, &ts.CreatedAt)
		result = append(result, ts)
	}
	return result, nil
}

func (r *StageAssigneeRepo) TimesheetSummary(ctx context.Context, from, to string) ([]models.TimesheetSummary, error) {
	query := `
		SELECT
			u.id::text,
			CONCAT(u.full_name, CASE WHEN u.last_name IS NOT NULL AND u.last_name != '' THEN ' ' || u.last_name ELSE '' END),
			ro.name,
			COALESCE(SUM(t.hours), 0)          AS total_hours,
			COUNT(DISTINCT t.work_date)         AS work_days,
			u.salary,
			u.hourly_rate,
			-- Бонус от сборки: % от суммы заказа
			COALESCE((
				SELECT SUM(
					(osa.assembly_percent / 100.0) *
					COALESCE(o.final_cost, o.estimated_cost, 0)
				)
				FROM order_stage_assignees osa
				JOIN order_stages os ON os.id = osa.stage_id
				JOIN orders o ON o.id = os.order_id
				WHERE osa.user_id = u.id
				  AND os.stage = 'assembly'
				  AND os.status = 'done'
				  AND osa.assembly_percent > 0
				  AND ($1 = '' OR os.finished_at::date >= $1::date)
				  AND ($2 = '' OR os.finished_at::date <= $2::date)
			), 0) AS assembly_bonus,
			-- Уже выплачено авансом
			COALESCE((
				SELECT SUM(amount) FROM salary_payments
				WHERE user_id = u.id AND payment_type = 'advance'
				  AND ($1 = '' OR period_from >= $1::date)
				  AND ($2 = '' OR period_to <= $2::date)
			), 0) AS paid_advance,
			-- Уже выплачено зарплатой
			COALESCE((
				SELECT SUM(amount) FROM salary_payments
				WHERE user_id = u.id AND payment_type = 'salary'
				  AND ($1 = '' OR period_from >= $1::date)
				  AND ($2 = '' OR period_to <= $2::date)
			), 0) AS paid_salary
		FROM users u
		JOIN roles ro ON ro.id = u.role_id
		LEFT JOIN timesheets t ON t.user_id = u.id
			AND ($1 = '' OR t.work_date >= $1::date)
			AND ($2 = '' OR t.work_date <= $2::date)
		WHERE u.is_active = true
		GROUP BY u.id, u.full_name, u.last_name, ro.name, u.salary, u.hourly_rate
		ORDER BY u.full_name`

	rows, err := r.db.QueryContext(ctx, query, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.TimesheetSummary
	for rows.Next() {
		var s models.TimesheetSummary
		rows.Scan(&s.UserID, &s.FullName, &s.RoleName,
			&s.TotalHours, &s.WorkDays,
			&s.Salary, &s.HourlyRate,
			&s.AssemblyBonus,
			&s.PaidAdvance, &s.PaidSalary)

		// Расчёт базовой зарплаты
		if s.HourlyRate != nil && s.TotalHours > 0 {
			s.Calculated = s.TotalHours * (*s.HourlyRate)
		} else if s.Salary != nil {
			s.Calculated = *s.Salary
		}

		s.TotalToPay = s.Calculated + s.AssemblyBonus
		s.TotalPaid = s.PaidAdvance + s.PaidSalary
		s.Remaining = s.TotalToPay - s.TotalPaid

		result = append(result, s)
	}
	return result, nil
}

func (r *StageAssigneeRepo) TimesheetCreate(ctx context.Context, req models.CreateTimesheetRequest, createdBy string) (string, error) {
	if req.Hours == 0 {
		req.Hours = 8
	}
	var id string
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO timesheets (user_id, work_date, hours, check_in, check_out, source_type, source_id, notes, created_by)
		VALUES ($1::uuid, $2::date, $3,
		        NULLIF($4,'')::time, NULLIF($5,'')::time,
		        NULLIF($6,''), NULLIF($7,'')::uuid, NULLIF($8,''), NULLIF($9,'')::uuid)
		ON CONFLICT (user_id, work_date, source_type, source_id) DO UPDATE
		  SET hours = EXCLUDED.hours,
		      check_in = EXCLUDED.check_in,
		      check_out = EXCLUDED.check_out
		RETURNING id
	`, req.UserID, req.WorkDate, req.Hours,
		req.CheckIn, req.CheckOut,
		req.SourceType, req.SourceID, req.Notes, createdBy,
	).Scan(&id)
	return id, err
}

func (r *StageAssigneeRepo) TimesheetDelete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM timesheets WHERE id = $1`, id)
	return err
}

func (r *StageAssigneeRepo) AutoFillFromOrderStages(ctx context.Context, from, to string) (int, error) {
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO timesheets (user_id, work_date, hours, source_type, source_id, notes)
		SELECT
			osa.user_id,
			COALESCE(os.finished_at::date, CURRENT_DATE) AS work_date,
			8 AS hours,
			'order_stage' AS source_type,
			os.id AS source_id,
			'Авто: этап заказа ' || o.order_number
		FROM order_stage_assignees osa
		JOIN order_stages os ON os.id = osa.stage_id
		JOIN orders o ON o.id = os.order_id
		WHERE os.status = 'done'
		  AND ($1 = '' OR COALESCE(os.finished_at::date, CURRENT_DATE) >= $1::date)
		  AND ($2 = '' OR COALESCE(os.finished_at::date, CURRENT_DATE) <= $2::date)
		ON CONFLICT (user_id, work_date, source_type, source_id) DO NOTHING
	`, from, to)
	if err != nil {
		return 0, err
	}
	n, _ := result.RowsAffected()
	return int(n), nil
}

// ── Выплаты зарплат ───────────────────────────────────────────

func (r *StageAssigneeRepo) SalaryPaymentList(ctx context.Context, from, to, userID string) ([]models.SalaryPayment, error) {
	query := `
		SELECT sp.id::text, sp.user_id::text,
		       CONCAT(u.full_name, CASE WHEN u.last_name IS NOT NULL AND u.last_name != '' THEN ' ' || u.last_name ELSE '' END),
		       sp.amount,
		       CAST(sp.period_from AS TEXT), CAST(sp.period_to AS TEXT),
		       sp.payment_type, sp.method,
		       COALESCE(sp.notes,''),
		       COALESCE(CAST(sp.paid_by AS TEXT),''),
		       COALESCE(CONCAT(pb.full_name, ' ', COALESCE(pb.last_name,'')), ''),
		       sp.paid_at
		FROM salary_payments sp
		JOIN users u ON u.id = sp.user_id
		LEFT JOIN users pb ON pb.id = sp.paid_by
		WHERE 1=1`

	args := []interface{}{}
	n := 1
	if from != "" {
		query += ` AND sp.period_from >= $` + itoa(n) + `::date`
		args = append(args, from); n++
	}
	if to != "" {
		query += ` AND sp.period_to <= $` + itoa(n) + `::date`
		args = append(args, to); n++
	}
	if userID != "" {
		query += ` AND sp.user_id = $` + itoa(n) + `::uuid`
		args = append(args, userID); n++
	}
	query += " ORDER BY sp.paid_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.SalaryPayment
	for rows.Next() {
		var sp models.SalaryPayment
		rows.Scan(&sp.ID, &sp.UserID, &sp.FullName,
			&sp.Amount, &sp.PeriodFrom, &sp.PeriodTo,
			&sp.PaymentType, &sp.Method, &sp.Notes,
			&sp.PaidBy, &sp.PaidByName, &sp.PaidAt)
		result = append(result, sp)
	}
	return result, nil
}

func (r *StageAssigneeRepo) SalaryPaymentCreate(ctx context.Context, req models.CreateSalaryPaymentRequest, paidBy string) (string, error) {
	if req.PaymentType == "" {
		req.PaymentType = "salary"
	}
	if req.Method == "" {
		req.Method = "cash"
	}
	var id string
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO salary_payments (user_id, amount, period_from, period_to, payment_type, method, notes, paid_by)
		VALUES ($1::uuid, $2, $3::date, $4::date, $5, $6, NULLIF($7,''), NULLIF($8,'')::uuid)
		RETURNING id
	`, req.UserID, req.Amount, req.PeriodFrom, req.PeriodTo,
		req.PaymentType, req.Method, req.Notes, paidBy,
	).Scan(&id)
	return id, err
}

func (r *StageAssigneeRepo) SalaryPaymentDelete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM salary_payments WHERE id = $1`, id)
	return err
}

// GetUserSalaryInfo — для проверки лимита аванса
func (r *StageAssigneeRepo) GetUserSalaryInfo(ctx context.Context, userID, periodFrom, periodTo string) (salary float64, hourlyRate float64, paidAdvance float64, err error) {
	r.db.QueryRowContext(ctx, `
		SELECT COALESCE(salary,0), COALESCE(hourly_rate,0) FROM users WHERE id = $1
	`, userID).Scan(&salary, &hourlyRate)

	r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(amount),0) FROM salary_payments
		WHERE user_id = $1 AND payment_type = 'advance'
		  AND period_from >= $2::date AND period_to <= $3::date
	`, userID, periodFrom, periodTo).Scan(&paidAdvance)

	return
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}

var _ = sql.ErrNoRows