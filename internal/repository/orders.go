package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"jevon/internal/models"
)

type OrderRepo struct {
	db *sql.DB
}

func NewOrderRepo(db *sql.DB) *OrderRepo {
	return &OrderRepo{db: db}
}

// ── Клиенты ──────────────────────────────────────────────

func (r *OrderRepo) ClientList(ctx context.Context, search string) ([]models.Client, error) {
	query := `
		SELECT id, full_name, phone, COALESCE(phone2,''), COALESCE(company,''),
		       COALESCE(address,''), COALESCE(notes,''), is_active, created_at
		FROM clients WHERE is_active = true`
	args := []interface{}{}
	if search != "" {
		query += ` AND (full_name ILIKE $1 OR phone ILIKE $1 OR phone2 ILIKE $1 OR company ILIKE $1)`
		args = append(args, "%"+search+"%")
	}
	query += " ORDER BY full_name"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.Client
	for rows.Next() {
		var c models.Client
		rows.Scan(&c.ID, &c.FullName, &c.Phone, &c.Phone2,
			&c.Company, &c.Address, &c.Notes, &c.IsActive, &c.CreatedAt)
		result = append(result, c)
	}
	return result, nil
}

func (r *OrderRepo) ClientCreate(ctx context.Context, req models.CreateClientRequest, createdBy string) (string, error) {
	var id string
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO clients (full_name, phone, phone2, company, address, notes, created_by)
		VALUES ($1,$2,NULLIF($3,''),NULLIF($4,''),$5,$6,NULLIF($7,'')::uuid)
		RETURNING id
	`, req.FullName, req.Phone, req.Phone2, req.Company,
		req.Address, req.Notes, createdBy,
	).Scan(&id)
	return id, err
}

func (r *OrderRepo) ClientUpdate(ctx context.Context, id string, req models.UpdateClientRequest) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE clients SET
			full_name = COALESCE($1, full_name),
			phone     = COALESCE($2, phone),
			phone2    = COALESCE($3, phone2),
			company   = COALESCE($4, company),
			address   = COALESCE($5, address),
			notes     = COALESCE($6, notes)
		WHERE id = $7
	`, req.FullName, req.Phone, req.Phone2, req.Company,
		req.Address, req.Notes, id)
	return err
}

func (r *OrderRepo) findOrCreateClient(ctx context.Context, clientID, clientName, clientPhone, createdBy string) string {
	if clientID != "" {
		return clientID
	}
	if clientPhone == "" && clientName == "" {
		return ""
	}
	if clientPhone != "" {
		var existingID string
		err := r.db.QueryRowContext(ctx, `
			SELECT id FROM clients WHERE phone = $1 OR phone2 = $1 LIMIT 1
		`, clientPhone).Scan(&existingID)
		if err == nil && existingID != "" {
			return existingID
		}
	}
	name := clientName
	if name == "" {
		name = clientPhone
	}
	var newID string
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO clients (full_name, phone, created_by)
		VALUES ($1, NULLIF($2,''), NULLIF($3,'')::uuid)
		RETURNING id
	`, name, clientPhone, createdBy).Scan(&newID)
	if err != nil {
		return ""
	}
	return newID
}

// ── Прайслист ─────────────────────────────────────────────

func (r *OrderRepo) PriceList(ctx context.Context, orderType string) ([]models.PriceItem, error) {
	query := `
		SELECT id, order_type, service_type, name, unit, price, currency, is_active, COALESCE(notes,'')
		FROM price_list WHERE is_active = true`
	args := []interface{}{}
	if orderType != "" {
		query += " AND order_type = $1"
		args = append(args, orderType)
	}
	query += " ORDER BY order_type, service_type, name"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.PriceItem
	for rows.Next() {
		var p models.PriceItem
		rows.Scan(&p.ID, &p.OrderType, &p.ServiceType, &p.Name,
			&p.Unit, &p.Price, &p.Currency, &p.IsActive, &p.Notes)
		result = append(result, p)
	}
	return result, nil
}

func (r *OrderRepo) PriceUpdate(ctx context.Context, id int, req models.UpdatePriceRequest) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE price_list SET
			price     = COALESCE($1, price),
			is_active = COALESCE($2, is_active),
			notes     = COALESCE($3, notes)
		WHERE id = $4
	`, req.Price, req.IsActive, req.Notes, id)
	return err
}

// ── Заказы ───────────────────────────────────────────────

func (r *OrderRepo) OrderList(ctx context.Context, userID, roleName, orderType, status, paymentStatus string) ([]models.Order, error) {
	query := `
		SELECT
			o.id, o.order_number, o.order_type,
			COALESCE(CAST(o.client_id AS TEXT),''),
			COALESCE(c.full_name, o.client_name, ''),
			COALESCE(c.phone, o.client_phone, ''),
			o.title, COALESCE(o.description,''), COALESCE(o.address,''),
			COALESCE(o.location_url,''),
			COALESCE(o.current_stage,''), o.status, o.priority,
			CAST(o.deadline AS TEXT),
			CAST(o.started_at AS TEXT),
			COALESCE(o.estimated_cost,0), COALESCE(o.final_cost,0),
			COALESCE(o.paid_amount,0), o.payment_status,
			COALESCE(CAST(o.manager_id AS TEXT),''),
			COALESCE(u.full_name,''),
			COALESCE(CAST(o.created_by AS TEXT),''),
			o.created_at
		FROM orders o
		LEFT JOIN clients c ON c.id = o.client_id
		LEFT JOIN users u   ON u.id = o.manager_id
		WHERE o.status != 'cancelled'`

	args := []interface{}{}
	n := 1

	if roleName != "admin" && roleName != "supervisor" && roleName != "manager" {
		query += fmt.Sprintf(`
			AND o.id IN (
				SELECT DISTINCT order_id FROM order_stages
				WHERE assigned_to = $%d
			)`, n)
		args = append(args, userID)
		n++
	}

	if orderType != "" {
		query += fmt.Sprintf(" AND o.order_type = $%d", n)
		args = append(args, orderType)
		n++
	}
	if status != "" {
		query += fmt.Sprintf(" AND o.status = $%d", n)
		args = append(args, status)
		n++
	}
	if paymentStatus != "" {
		query += fmt.Sprintf(" AND o.payment_status = $%d", n)
		args = append(args, paymentStatus)
		n++
	}

	query += " ORDER BY o.order_number DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.Order
	for rows.Next() {
		var o models.Order
		var deadline, startedAt sql.NullString
		rows.Scan(
			&o.ID, &o.OrderNumber, &o.OrderType,
			&o.ClientID, &o.ClientName, &o.ClientPhone,
			&o.Title, &o.Description, &o.Address, &o.LocationURL,
			&o.CurrentStage, &o.Status, &o.Priority,
			&deadline, &startedAt,
			&o.EstimatedCost, &o.FinalCost,
			&o.PaidAmount, &o.PaymentStatus,
			&o.ManagerID, &o.ManagerName,
			&o.CreatedBy, &o.CreatedAt,
		)
		if deadline.Valid {
			o.Deadline = &deadline.String
		}
		result = append(result, o)
	}
	return result, nil
}

func (r *OrderRepo) OrderByID(ctx context.Context, id string) (*models.Order, error) {
	var o models.Order
	var deadline, startedAt, finishedAt sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT
			o.id, o.order_number, o.order_type,
			COALESCE(CAST(o.client_id AS TEXT),''),
			COALESCE(c.full_name, o.client_name, ''),
			COALESCE(c.phone, o.client_phone, ''),
			o.title, COALESCE(o.description,''), COALESCE(o.address,''),
			COALESCE(o.location_url,''),
			COALESCE(o.current_stage,''), o.status, o.priority,
			CAST(o.deadline AS TEXT),
			CAST(o.started_at AS TEXT),
			CAST(o.finished_at AS TEXT),
			COALESCE(o.estimated_cost,0), COALESCE(o.final_cost,0),
			COALESCE(o.paid_amount,0), o.payment_status,
			COALESCE(CAST(o.driver_id AS TEXT),''),
			COALESCE(o.vehicle,''),
			COALESCE(o.distance_km,0), COALESCE(o.fuel_expense,0),
			COALESCE(CAST(o.manager_id AS TEXT),''),
			COALESCE(u.full_name,''),
			COALESCE(CAST(o.created_by AS TEXT),''),
			o.created_at
		FROM orders o
		LEFT JOIN clients c ON c.id = o.client_id
		LEFT JOIN users u   ON u.id = o.manager_id
		WHERE o.id = $1
	`, id).Scan(
		&o.ID, &o.OrderNumber, &o.OrderType,
		&o.ClientID, &o.ClientName, &o.ClientPhone,
		&o.Title, &o.Description, &o.Address, &o.LocationURL,
		&o.CurrentStage, &o.Status, &o.Priority,
		&deadline, &startedAt, &finishedAt,
		&o.EstimatedCost, &o.FinalCost,
		&o.PaidAmount, &o.PaymentStatus,
		&o.DriverID, &o.Vehicle,
		&o.DistanceKm, &o.FuelExpense,
		&o.ManagerID, &o.ManagerName,
		&o.CreatedBy, &o.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if deadline.Valid   { o.Deadline   = &deadline.String   }
	if startedAt.Valid  { o.StartedAt  = &startedAt.String  }
	if finishedAt.Valid { o.FinishedAt = &finishedAt.String }
	return &o, err
}

func (r *OrderRepo) OrderCreate(ctx context.Context, req models.CreateOrderRequest, createdBy string) (string, error) {
	if req.Priority == "" {
		req.Priority = "medium"
	}
	clientID := r.findOrCreateClient(ctx, req.ClientID, req.ClientName, req.ClientPhone, createdBy)

	var id string
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO orders
			(order_type, client_id, client_name, client_phone,
			 title, description, address, location_url, priority, deadline,
			 estimated_cost, manager_id, created_by, status, current_stage)
		VALUES
			($1,
			 NULLIF($2,'')::uuid, NULLIF($3,''), NULLIF($4,''),
			 $5, $6, $7, NULLIF($8,''), $9, NULLIF($10,'')::date,
			 $11, NULLIF($12,'')::uuid, NULLIF($13,'')::uuid,
			 'new', 'intake')
		RETURNING id
	`, req.OrderType,
		clientID, req.ClientName, req.ClientPhone,
		req.Title, req.Description, req.Address, req.LocationURL,
		req.Priority, req.Deadline,
		req.EstimatedCost, req.ManagerID, createdBy,
	).Scan(&id)
	return id, err
}

func (r *OrderRepo) OrderUpdate(ctx context.Context, id string, req models.UpdateOrderRequest, updatedBy string) error {
	var oldTitle, oldStatus, oldPriority, oldAddress string
	var oldFinalCost, oldEstimatedCost float64
	r.db.QueryRowContext(ctx, `
		SELECT title, COALESCE(status,''), COALESCE(priority,''),
		       COALESCE(address,''), COALESCE(final_cost,0), COALESCE(estimated_cost,0)
		FROM orders WHERE id = $1
	`, id).Scan(&oldTitle, &oldStatus, &oldPriority, &oldAddress, &oldFinalCost, &oldEstimatedCost)

	_, err := r.db.ExecContext(ctx, `
		UPDATE orders SET
			title          = COALESCE($1, title),
			description    = COALESCE($2, description),
			address        = COALESCE($3, address),
			location_url   = COALESCE($4, location_url),
			status         = COALESCE($5, status),
			priority       = COALESCE($6, priority),
			deadline       = COALESCE(NULLIF($7,'')::date, deadline),
			estimated_cost = COALESCE($8, estimated_cost),
			final_cost     = COALESCE($9, final_cost),
			manager_id     = COALESCE(NULLIF($10,'')::uuid, manager_id),
			driver_id      = COALESCE(NULLIF($11,'')::uuid, driver_id),
			vehicle        = COALESCE($12, vehicle),
			distance_km    = COALESCE($13, distance_km),
			fuel_expense   = COALESCE($14, fuel_expense)
		WHERE id = $15
	`, req.Title, req.Description, req.Address, req.LocationURL,
		req.Status, req.Priority, req.Deadline,
		req.EstimatedCost, req.FinalCost,
		req.ManagerID, req.DriverID, req.Vehicle,
		req.DistanceKm, req.FuelExpense, id)

	if err != nil {
		return err
	}

	var changes []string
	if req.Title != nil && *req.Title != oldTitle {
		changes = append(changes, fmt.Sprintf("Название: «%s»", *req.Title))
	}
	if req.Address != nil && *req.Address != oldAddress {
		changes = append(changes, fmt.Sprintf("Адрес: %s", *req.Address))
	}
	if req.LocationURL != nil && *req.LocationURL != "" {
		changes = append(changes, "Локация обновлена")
	}
	if req.FinalCost != nil && *req.FinalCost != oldFinalCost {
		changes = append(changes, fmt.Sprintf("Итоговая сумма: %.0f сом.", *req.FinalCost))
	}
	if req.EstimatedCost != nil && *req.EstimatedCost != oldEstimatedCost {
		changes = append(changes, fmt.Sprintf("Предв. сумма: %.0f сом.", *req.EstimatedCost))
	}
	if req.Status != nil && *req.Status != oldStatus {
		statusLabels := map[string]string{
			"new": "Новый", "in_progress": "В работе",
			"on_hold": "Ожидание", "done": "Готово", "cancelled": "Отменён",
		}
		label := statusLabels[*req.Status]
		if label == "" { label = *req.Status }
		changes = append(changes, fmt.Sprintf("Статус: %s", label))
	}
	if req.Priority != nil && *req.Priority != oldPriority {
		priorityLabels := map[string]string{
			"low": "Низкий", "medium": "Средний",
			"high": "Высокий", "urgent": "Срочный",
		}
		label := priorityLabels[*req.Priority]
		if label == "" { label = *req.Priority }
		changes = append(changes, fmt.Sprintf("Приоритет: %s", label))
	}
	if req.Deadline != nil && *req.Deadline != "" {
		changes = append(changes, fmt.Sprintf("Срок: %s", *req.Deadline))
	}

	if len(changes) > 0 {
		comment := "✏️ Изменён заказ: " + strings.Join(changes, " | ")
		r.db.ExecContext(ctx, `
			INSERT INTO order_history (order_id, from_stage, to_stage, changed_by, comment)
			VALUES ($1, 'edit', 'edit', NULLIF($2,'')::uuid, $3)
		`, id, updatedBy, comment)
	}

	return nil
}

func (r *OrderRepo) OrderCancel(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE orders SET status = 'cancelled' WHERE id = $1`, id)
	return err
}

// ── Этапы ────────────────────────────────────────────────

func (r *OrderRepo) StagesByOrder(ctx context.Context, orderID string) ([]models.OrderStage, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			os.id, os.order_id, os.stage, os.stage_order, os.status,
			COALESCE(CAST(os.assigned_to AS TEXT),''),
			COALESCE(u.full_name,''),
			os.started_at, os.finished_at,
			COALESCE(os.notes,''), os.updated_at
		FROM order_stages os
		LEFT JOIN users u ON u.id = os.assigned_to
		WHERE os.order_id = $1
		ORDER BY os.stage_order
	`, orderID)
	if err != nil { return nil, err }
	defer rows.Close()

	var result []models.OrderStage
	for rows.Next() {
		var s models.OrderStage
		rows.Scan(
			&s.ID, &s.OrderID, &s.Stage, &s.StageOrder, &s.Status,
			&s.AssignedTo, &s.AssigneeName,
			&s.StartedAt, &s.FinishedAt,
			&s.Notes, &s.UpdatedAt,
		)
		result = append(result, s)
	}
	return result, nil
}

func (r *OrderRepo) StageUpdate(ctx context.Context, stageID string, req models.UpdateOrderStageRequest) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE order_stages SET
			status      = COALESCE($1, status),
			assigned_to = COALESCE(NULLIF($2,'')::uuid, assigned_to),
			notes       = COALESCE($3, notes),
			started_at  = CASE
				WHEN $1 = 'in_progress' AND started_at IS NULL THEN NOW()
				ELSE started_at
			END
		WHERE id = $4
	`, req.Status, req.AssignedTo, req.Notes, stageID)
	return err
}

func (r *OrderRepo) StageComplete(ctx context.Context, stageID string, req models.CompleteOrderStageRequest) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE order_stages SET
			status = 'done',
			notes  = COALESCE(NULLIF($1,''), notes)
		WHERE id = $2
	`, req.Notes, stageID)
	return err
}

// ── Оплаты ───────────────────────────────────────────────

func (r *OrderRepo) PaymentsByOrder(ctx context.Context, orderID string) ([]models.OrderPayment, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			p.id, p.order_id, p.amount, p.payment_type,
			p.paid_at, COALESCE(p.notes,''),
			COALESCE(CAST(p.received_by AS TEXT),''),
			COALESCE(u.full_name,''), p.created_at
		FROM order_payments p
		LEFT JOIN users u ON u.id = p.received_by
		WHERE p.order_id = $1
		ORDER BY p.paid_at DESC
	`, orderID)
	if err != nil { return nil, err }
	defer rows.Close()

	var result []models.OrderPayment
	for rows.Next() {
		var p models.OrderPayment
		rows.Scan(
			&p.ID, &p.OrderID, &p.Amount, &p.PaymentType,
			&p.PaidAt, &p.Notes, &p.ReceivedBy, &p.ReceiverName, &p.CreatedAt,
		)
		result = append(result, p)
	}
	return result, nil
}

func (r *OrderRepo) PaymentCreate(ctx context.Context, orderID, receivedBy string, req models.CreatePaymentRequest) (string, error) {
	if req.PaymentType == "" { req.PaymentType = "cash" }

	payTypeLabels := map[string]string{
		"cash": "Наличные", "card": "Карта",
		"transfer": "Перевод", "other": "Другое",
	}
	payLabel := payTypeLabels[req.PaymentType]
	if payLabel == "" { payLabel = req.PaymentType }

	var id string
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO order_payments (order_id, amount, payment_type, notes, received_by)
		VALUES ($1,$2,$3,$4,NULLIF($5,'')::uuid)
		RETURNING id
	`, orderID, req.Amount, req.PaymentType, req.Notes, receivedBy,
	).Scan(&id)

	if err != nil {
		return "", err
	}

	comment := fmt.Sprintf("💰 Оплата: %.0f сом. | %s", req.Amount, payLabel)
	if req.Notes != "" {
		comment += " | " + req.Notes
	}
	r.db.ExecContext(ctx, `
		INSERT INTO order_history (order_id, from_stage, to_stage, changed_by, comment)
		VALUES ($1, 'payment', 'payment', NULLIF($2,'')::uuid, $3)
	`, orderID, receivedBy, comment)

	return id, nil
}

// ── Расчёты ───────────────────────────────────────────────

func (r *OrderRepo) CalculationByOrder(ctx context.Context, orderID string) (*models.OrderCalculation, error) {
	var c models.OrderCalculation
	err := r.db.QueryRowContext(ctx, `
		SELECT
			id, order_id,
			COALESCE(CAST(stage_id AS TEXT),''),
			COALESCE(total_area_m2,0),
			COALESCE(paint_kg,0), COALESCE(primer_kg,0),
			COALESCE(paint_type,''), COALESCE(cnc_type,''),
			COALESCE(cnc_time_hours,0), COALESCE(sheet_count,0),
			COALESCE(cut_length_m,0),
			COALESCE(client_material,false),
			COALESCE(client_material_desc,''),
			COALESCE(calculated_cost,0),
			COALESCE(CAST(calculated_by AS TEXT),''),
			CAST(calculated_at AS TEXT),
			COALESCE(notes,'')
		FROM order_calculations
		WHERE order_id = $1
		ORDER BY calculated_at DESC
		LIMIT 1
	`, orderID).Scan(
		&c.ID, &c.OrderID, &c.StageID,
		&c.TotalAreaM2, &c.PaintKg, &c.PrimerKg,
		&c.PaintType, &c.CNCType, &c.CNCTimeHours,
		&c.SheetCount, &c.CutLengthM,
		&c.ClientMaterial, &c.ClientMatDesc,
		&c.CalculatedCost, &c.CalculatedBy, &c.CalculatedAt,
		&c.Notes,
	)
	if err == sql.ErrNoRows { return nil, nil }
	return &c, err
}

func (r *OrderRepo) CalculationCreate(ctx context.Context, orderID, calcBy string, req models.CreateCalculationRequest) (string, error) {
	paintKg  := req.TotalAreaM2 * 0.35
	primerKg := req.TotalAreaM2 * 0.40

	var pricePerM2 float64
	serviceType := req.CNCType
	if serviceType == "" { serviceType = "painting" }
	r.db.QueryRowContext(ctx, `
		SELECT COALESCE(price, 0) FROM price_list
		WHERE service_type = $1 AND is_active = true LIMIT 1
	`, serviceType).Scan(&pricePerM2)

	calculatedCost := req.TotalAreaM2 * pricePerM2

	var id string
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO order_calculations
			(order_id, stage_id, total_area_m2, paint_kg, primer_kg,
			 paint_type, cnc_type, cnc_time_hours,
			 sheet_count, cut_length_m,
			 client_material, client_material_desc,
			 calculated_cost, calculated_by, notes)
		VALUES
			($1, NULLIF($2,'')::uuid, $3, $4, $5,
			 NULLIF($6,''), NULLIF($7,''), $8,
			 $9, $10, $11, $12, $13, NULLIF($14,'')::uuid, $15)
		RETURNING id
	`, orderID, req.StageID, req.TotalAreaM2, paintKg, primerKg,
		req.PaintType, req.CNCType, req.CNCTimeHours,
		req.SheetCount, req.CutLengthM,
		req.ClientMaterial, req.ClientMatDesc,
		calculatedCost, calcBy, req.Notes,
	).Scan(&id)

	if err == nil {
		r.db.ExecContext(ctx,
			`UPDATE orders SET estimated_cost = $1 WHERE id = $2`,
			calculatedCost, orderID)
	}
	return id, err
}

// ── Комментарии ───────────────────────────────────────────

func (r *OrderRepo) CommentsByOrder(ctx context.Context, orderID string) ([]models.OrderComment, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			c.id, c.order_id,
			COALESCE(CAST(c.stage_id AS TEXT),''),
			c.text,
			COALESCE(CAST(c.author_id AS TEXT),''),
			COALESCE(u.full_name,''),
			c.created_at
		FROM order_comments c
		LEFT JOIN users u ON u.id = c.author_id
		WHERE c.order_id = $1
		ORDER BY c.created_at DESC
	`, orderID)
	if err != nil { return nil, err }
	defer rows.Close()

	var result []models.OrderComment
	for rows.Next() {
		var c models.OrderComment
		rows.Scan(
			&c.ID, &c.OrderID, &c.StageID,
			&c.Text, &c.AuthorID, &c.AuthorName, &c.CreatedAt,
		)
		result = append(result, c)
	}
	return result, nil
}

func (r *OrderRepo) CommentCreate(ctx context.Context, orderID, authorID string, req models.CreateCommentRequest) (string, error) {
	var id string
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO order_comments (order_id, stage_id, text, author_id)
		VALUES ($1, NULLIF($2,'')::uuid, $3, NULLIF($4,'')::uuid)
		RETURNING id
	`, orderID, req.StageID, req.Text, authorID,
	).Scan(&id)
	return id, err
}

// ── История ───────────────────────────────────────────────

func (r *OrderRepo) History(ctx context.Context, orderID string) ([]models.OrderHistory, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			h.id, h.order_id,
			COALESCE(h.from_stage,''), COALESCE(h.to_stage,''),
			COALESCE(CAST(h.changed_by AS TEXT),''),
			COALESCE(u.full_name,''),
			COALESCE(h.comment,''), h.created_at
		FROM order_history h
		LEFT JOIN users u ON u.id = h.changed_by
		WHERE h.order_id = $1
		ORDER BY h.created_at DESC
	`, orderID)
	if err != nil { return nil, err }
	defer rows.Close()

	var result []models.OrderHistory
	for rows.Next() {
		var h models.OrderHistory
		rows.Scan(
			&h.ID, &h.OrderID, &h.FromStage, &h.ToStage,
			&h.ChangedBy, &h.ChangerName, &h.Comment, &h.CreatedAt,
		)
		result = append(result, h)
	}
	return result, nil
}

// ── Stats ─────────────────────────────────────────────────

type OrderStats struct {
	TotalOrders   int     `json:"total_orders"`
	ActiveOrders  int     `json:"active_orders"`
	DoneOrders    int     `json:"done_orders"`
	UnpaidOrders  int     `json:"unpaid_orders"`
	TotalRevenue  float64 `json:"total_revenue"`
	TotalDebt     float64 `json:"total_debt"`
	WorkshopCount int     `json:"workshop_count"`
	CuttingCount  int     `json:"cutting_count"`
	PaintingCount int     `json:"painting_count"`
	CNCCount      int     `json:"cnc_count"`
}

func (r *OrderRepo) Stats(ctx context.Context) (*OrderStats, error) {
	var s OrderStats
	err := r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*)                                             AS total,
			COUNT(*) FILTER (WHERE status = 'in_progress')     AS active,
			COUNT(*) FILTER (WHERE status = 'done')            AS done,
			COUNT(*) FILTER (WHERE payment_status != 'paid' AND status != 'cancelled') AS unpaid,
			COALESCE(SUM(final_cost), 0)                       AS revenue,
			COALESCE(SUM(final_cost - paid_amount) FILTER (WHERE payment_status != 'paid' AND status != 'cancelled'), 0) AS debt,
			COUNT(*) FILTER (WHERE order_type = 'workshop')    AS workshop,
			COUNT(*) FILTER (WHERE order_type = 'cutting')     AS cutting,
			COUNT(*) FILTER (WHERE order_type = 'painting')    AS painting,
			COUNT(*) FILTER (WHERE order_type = 'cnc')         AS cnc
		FROM orders WHERE status != 'cancelled'
	`).Scan(
		&s.TotalOrders, &s.ActiveOrders, &s.DoneOrders,
		&s.UnpaidOrders, &s.TotalRevenue, &s.TotalDebt,
		&s.WorkshopCount, &s.CuttingCount, &s.PaintingCount, &s.CNCCount,
	)
	return &s, err
}

// ── Материалы заказа ─────────────────────────────────────

type OrderMaterial struct {
	ID         string  `json:"id"`
	OrderID    string  `json:"order_id"`
	StageID    string  `json:"stage_id"`
	StageName  string  `json:"stage_name"`
	Name       string  `json:"name"`
	Quantity   float64 `json:"quantity"`
	Unit       string  `json:"unit"`
	UnitPrice  float64 `json:"unit_price"`
	TotalPrice float64 `json:"total_price"`
	Supplier   string  `json:"supplier"`
	Notes      string  `json:"notes"`
	CreatedBy  string  `json:"created_by"`
	CreatedAt  string  `json:"created_at"`
}

type CreateOrderMaterialRequest struct {
	StageID   string  `json:"stage_id"`
	StageName string  `json:"stage_name"`
	Name      string  `json:"name" binding:"required"`
	Quantity  float64 `json:"quantity"`
	Unit      string  `json:"unit"`
	UnitPrice float64 `json:"unit_price"`
	Supplier  string  `json:"supplier"`
	Notes     string  `json:"notes"`
}

func (r *OrderRepo) MaterialsByOrder(ctx context.Context, orderID string) ([]OrderMaterial, float64, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			id, order_id,
			COALESCE(CAST(stage_id AS TEXT),''),
			COALESCE(stage_name,''),
			name, quantity, unit,
			COALESCE(unit_price,0), COALESCE(total_price,0),
			COALESCE(supplier,''), COALESCE(notes,''),
			COALESCE(CAST(created_by AS TEXT),''),
			CAST(created_at AS TEXT)
		FROM order_materials
		WHERE order_id = $1
		ORDER BY created_at ASC
	`, orderID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var result []OrderMaterial
	var total float64
	for rows.Next() {
		var m OrderMaterial
		rows.Scan(
			&m.ID, &m.OrderID, &m.StageID, &m.StageName,
			&m.Name, &m.Quantity, &m.Unit,
			&m.UnitPrice, &m.TotalPrice,
			&m.Supplier, &m.Notes, &m.CreatedBy, &m.CreatedAt,
		)
		total += m.TotalPrice
		result = append(result, m)
	}
	return result, total, nil
}

func (r *OrderRepo) MaterialCreate(ctx context.Context, orderID, createdBy string, req CreateOrderMaterialRequest) (string, error) {
	if req.Unit == "" { req.Unit = "шт" }

	var id string
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO order_materials
			(order_id, stage_id, stage_name, name, quantity, unit, unit_price, supplier, notes, created_by)
		VALUES
			($1, NULLIF($2,'')::uuid, NULLIF($3,''), $4, $5, $6, $7, NULLIF($8,''), NULLIF($9,''), NULLIF($10,'')::uuid)
		RETURNING id
	`, orderID, req.StageID, req.StageName, req.Name,
		req.Quantity, req.Unit, req.UnitPrice,
		req.Supplier, req.Notes, createdBy,
	).Scan(&id)

	if err != nil {
		return "", err
	}

	total := req.Quantity * req.UnitPrice
	comment := fmt.Sprintf("+ Материал: %s | %g %s", req.Name, req.Quantity, req.Unit)
	if total > 0 {
		comment += fmt.Sprintf(" | %.0f сом.", total)
	}
	if req.StageName != "" {
		comment += " | Этап: " + req.StageName
	}
	r.db.ExecContext(ctx, `
		INSERT INTO order_history (order_id, from_stage, to_stage, changed_by, comment)
		VALUES ($1, 'materials', 'materials', NULLIF($2,'')::uuid, $3)
	`, orderID, createdBy, comment)

	return id, nil
}

func (r *OrderRepo) MaterialDelete(ctx context.Context, materialID, deletedBy string) error {
	var name, unit, stageName, orderID string
	var quantity float64
	r.db.QueryRowContext(ctx, `
		SELECT order_id::text, name, quantity, unit, COALESCE(stage_name,'')
		FROM order_materials WHERE id = $1
	`, materialID).Scan(&orderID, &name, &quantity, &unit, &stageName)

	_, err := r.db.ExecContext(ctx, `DELETE FROM order_materials WHERE id = $1`, materialID)
	if err != nil {
		return err
	}

	if orderID != "" && name != "" {
		comment := fmt.Sprintf("- Удалён материал: %s | %g %s", name, quantity, unit)
		if stageName != "" {
			comment += " | Этап: " + stageName
		}
		r.db.ExecContext(ctx, `
			INSERT INTO order_history (order_id, from_stage, to_stage, changed_by, comment)
			VALUES ($1, 'materials', 'materials', NULLIF($2,'')::uuid, $3)
		`, orderID, deletedBy, comment)
	}

	return nil
}

// ── Каталог материалов ────────────────────────────────────

type CatalogMaterial struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Category string  `json:"category"`
	Unit     string  `json:"unit"`
	Price    float64 `json:"price"`
}

func (r *OrderRepo) MaterialsCatalog(ctx context.Context, search string) ([]CatalogMaterial, error) {
	query := `
		SELECT id::text, name, category, unit, COALESCE(price, 0)
		FROM materials_catalog
		WHERE is_active = true`
	args := []interface{}{}
	if search != "" {
		query += ` AND name ILIKE $1`
		args = append(args, "%"+search+"%")
	}
	query += " ORDER BY name LIMIT 50"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []CatalogMaterial
	for rows.Next() {
		var m CatalogMaterial
		rows.Scan(&m.ID, &m.Name, &m.Category, &m.Unit, &m.Price)
		result = append(result, m)
	}
	return result, nil
}

func (r *OrderRepo) LogHistory(ctx context.Context, orderID, fromStage, toStage, changedBy, comment string) {
	r.db.ExecContext(ctx, `
		INSERT INTO order_history (order_id, from_stage, to_stage, changed_by, comment)
		VALUES ($1::uuid, $2, $3, NULLIF($4,'')::uuid, $5)
	`, orderID, fromStage, toStage, changedBy, comment)
}

var _ = strings.Contains

// ── Расходы ───────────────────────────────────────────────

type Expense struct {
	ID          string  `json:"id"`
	OrderID     string  `json:"order_id"`
	Name        string  `json:"name"`
	Amount      float64 `json:"amount"`
	ExpenseDate string  `json:"expense_date"`
	Description string  `json:"description"`
	Method      string  `json:"method"`
	CreatedBy   string  `json:"created_by"`
	CreatedAt   string  `json:"created_at"`
}

type CreateExpenseRequest struct {
	Name        string  `json:"name"         binding:"required"`
	Amount      float64 `json:"amount"       binding:"required"`
	ExpenseDate string  `json:"expense_date"`
	Description string  `json:"description"`
	Method      string  `json:"method"`
}

func (r *OrderRepo) ExpensesList(ctx context.Context, orderID string) ([]Expense, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id::text, order_id::text, name,
		       amount,
		       COALESCE(CAST(expense_date AS TEXT),''),
		       COALESCE(description,''),
		       COALESCE(method,'cash'),
		       COALESCE(CAST(created_by AS TEXT),''),
		       created_at
		FROM order_expenses
		WHERE order_id = $1
		ORDER BY created_at DESC
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Expense
	for rows.Next() {
		var e Expense
		rows.Scan(&e.ID, &e.OrderID, &e.Name, &e.Amount,
			&e.ExpenseDate, &e.Description, &e.Method,
			&e.CreatedBy, &e.CreatedAt)
		result = append(result, e)
	}
	return result, nil
}

func (r *OrderRepo) ExpenseCreate(ctx context.Context, orderID, createdBy string, req CreateExpenseRequest) (string, error) {
	var id string
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO order_expenses (order_id, name, amount, expense_date, description, method, created_by)
		VALUES ($1::uuid, $2, $3, NULLIF($4,'')::date, $5, $6, NULLIF($7,'')::uuid)
		RETURNING id
	`, orderID, req.Name, req.Amount,
		req.ExpenseDate, req.Description,
		func() string { if req.Method == "" { return "cash" }; return req.Method }(),
		createdBy,
	).Scan(&id)
	return id, err
}

func (r *OrderRepo) ExpenseDelete(ctx context.Context, expenseID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM order_expenses WHERE id = $1`, expenseID)
	return err
}

func (r *OrderRepo) ExpensesTotal(ctx context.Context, orderID string) (float64, error) {
	var total float64
	err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM order_expenses WHERE order_id = $1
	`, orderID).Scan(&total)
	return total, err
}