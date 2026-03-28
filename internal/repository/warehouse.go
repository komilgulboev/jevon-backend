package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"jevon/internal/models"

	"github.com/google/uuid"
)

type WarehouseRepo struct {
	db *sql.DB
}

func NewWarehouseRepo(db *sql.DB) *WarehouseRepo {
	return &WarehouseRepo{db: db}
}

// ─── Единицы измерения ────────────────────────────────────────

func (r *WarehouseRepo) UnitList(ctx context.Context) ([]models.Unit, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name FROM units ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var units []models.Unit
	for rows.Next() {
		var u models.Unit
		if err := rows.Scan(&u.ID, &u.Name); err != nil {
			return nil, err
		}
		units = append(units, u)
	}
	return units, nil
}

// ─── Категории ────────────────────────────────────────────────

func (r *WarehouseRepo) CategoryList(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT category FROM warehouse_items
		WHERE category IS NOT NULL AND category != ''
		ORDER BY category`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cats []string
	for rows.Next() {
		var cat string
		rows.Scan(&cat)
		cats = append(cats, cat)
	}
	return cats, nil
}

// ─── Номенклатура ─────────────────────────────────────────────

func (r *WarehouseRepo) ItemList(ctx context.Context, category, search, active string) ([]models.WarehouseItem, error) {
	query := `
		SELECT b.id, b.name,
			COALESCE(b.article,  '') AS article,
			COALESCE(b.category, '') AS category,
			wi.unit_id,
			COALESCE(b.unit,     '') AS unit,
			wi.min_stock,
			COALESCE(wi.notes,   '') AS notes,
			wi.is_active,
			b.total_in, b.total_out, b.balance, b.avg_price
		FROM warehouse_balance b
		JOIN warehouse_items wi ON wi.id = b.id
		WHERE 1=1`

	args := []interface{}{}
	i := 1
	if category != "" {
		query += fmt.Sprintf(` AND b.category = $%d`, i)
		args = append(args, category)
		i++
	}
	if active == "true" {
		query += ` AND wi.is_active = TRUE`
	} else if active == "false" {
		query += ` AND wi.is_active = FALSE`
	}
	if search != "" {
		query += fmt.Sprintf(` AND (b.name ILIKE $%d OR b.article ILIKE $%d)`, i, i)
		args = append(args, "%"+search+"%")
		i++
	}
	query += ` ORDER BY b.category, b.name`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []models.WarehouseItem
	for rows.Next() {
		var it models.WarehouseItem
		if err := rows.Scan(
			&it.ID, &it.Name, &it.Article, &it.Category,
			&it.UnitID, &it.UnitName, &it.MinStock, &it.Notes, &it.IsActive,
			&it.TotalIn, &it.TotalOut, &it.Balance, &it.AvgPrice,
		); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, nil
}

func (r *WarehouseRepo) ItemByID(ctx context.Context, id string) (*models.WarehouseItem, error) {
	var it models.WarehouseItem
	err := r.db.QueryRowContext(ctx, `
		SELECT b.id, b.name,
			COALESCE(b.article,  '') AS article,
			COALESCE(b.category, '') AS category,
			wi.unit_id,
			COALESCE(b.unit,     '') AS unit,
			wi.min_stock,
			COALESCE(wi.notes,   '') AS notes,
			wi.is_active,
			b.total_in, b.total_out, b.balance, b.avg_price
		FROM warehouse_balance b
		JOIN warehouse_items wi ON wi.id = b.id
		WHERE b.id = $1`, id).Scan(
		&it.ID, &it.Name, &it.Article, &it.Category,
		&it.UnitID, &it.UnitName, &it.MinStock, &it.Notes, &it.IsActive,
		&it.TotalIn, &it.TotalOut, &it.Balance, &it.AvgPrice,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &it, err
}

func (r *WarehouseRepo) ItemCreate(ctx context.Context, req models.CreateWarehouseItemRequest) (string, error) {
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	id := uuid.New().String()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO warehouse_items (id, name, article, category, unit_id, min_stock, notes, is_active)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		id, strings.TrimSpace(req.Name), strings.TrimSpace(req.Article),
		strings.TrimSpace(req.Category), req.UnitID, req.MinStock,
		strings.TrimSpace(req.Notes), isActive,
	)
	return id, err
}

func (r *WarehouseRepo) ItemUpdate(ctx context.Context, id string, req models.UpdateWarehouseItemRequest) error {
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	res, err := r.db.ExecContext(ctx, `
		UPDATE warehouse_items
		SET name=$1, article=$2, category=$3, unit_id=$4, min_stock=$5, notes=$6, is_active=$7
		WHERE id=$8`,
		strings.TrimSpace(req.Name), strings.TrimSpace(req.Article),
		strings.TrimSpace(req.Category), req.UnitID, req.MinStock,
		strings.TrimSpace(req.Notes), isActive, id,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("not found")
	}
	return nil
}

func (r *WarehouseRepo) ItemDelete(ctx context.Context, id string) (string, error) {
	var cnt int
	r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM (
			SELECT item_id FROM warehouse_receipt_items WHERE item_id=$1
			UNION ALL
			SELECT item_id FROM warehouse_expenses WHERE item_id=$1
		) t`, id).Scan(&cnt)
	if cnt > 0 {
		_, err := r.db.ExecContext(ctx, `UPDATE warehouse_items SET is_active=FALSE WHERE id=$1`, id)
		return "deactivated", err
	}
	res, err := r.db.ExecContext(ctx, `DELETE FROM warehouse_items WHERE id=$1`, id)
	if err != nil {
		return "", err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return "", fmt.Errorf("not found")
	}
	return "deleted", nil
}

// ─── Поставщики ───────────────────────────────────────────────

func (r *WarehouseRepo) SupplierList(ctx context.Context, search, active string) ([]models.Supplier, error) {
	query := `
		SELECT
			s.id, s.name,
			COALESCE(s.phone,    '') AS phone,
			COALESCE(s.email,    '') AS email,
			COALESCE(s.address,  '') AS address,
			COALESCE(s.whatsapp, '') AS whatsapp,
			COALESCE(s.telegram, '') AS telegram,
			COALESCE(s.notes,    '') AS notes,
			s.is_active,
			s.created_at::text,
			COALESCE(d.total_debt,   0) AS total_debt,
			COALESCE(d.total_amount, 0) AS total_amount,
			COALESCE(d.total_paid,   0) AS total_paid,
			COALESCE(d.unpaid_count, 0) AS unpaid_count
		FROM suppliers s
		LEFT JOIN supplier_debt d ON d.supplier_id = s.id
		WHERE 1=1`

	args := []interface{}{}
	i := 1
	if search != "" {
		query += fmt.Sprintf(` AND (s.name ILIKE $%d OR s.phone ILIKE $%d OR s.email ILIKE $%d)`, i, i, i)
		args = append(args, "%"+search+"%")
		i++
	}
	if active == "true" {
		query += ` AND s.is_active = TRUE`
	} else if active == "false" {
		query += ` AND s.is_active = FALSE`
	}
	query += ` ORDER BY s.name`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var suppliers []models.Supplier
	for rows.Next() {
		var s models.Supplier
		if err := rows.Scan(
			&s.ID, &s.Name, &s.Phone, &s.Email,
			&s.Address, &s.WhatsApp, &s.Telegram,
			&s.Notes, &s.IsActive, &s.CreatedAt,
			&s.TotalDebt, &s.TotalAmount, &s.TotalPaid, &s.UnpaidCount,
		); err != nil {
			return nil, err
		}
		suppliers = append(suppliers, s)
	}
	return suppliers, nil
}

func (r *WarehouseRepo) SupplierByID(ctx context.Context, id string) (*models.Supplier, error) {
	var s models.Supplier
	err := r.db.QueryRowContext(ctx, `
		SELECT
			s.id, s.name,
			COALESCE(s.phone,    '') AS phone,
			COALESCE(s.email,    '') AS email,
			COALESCE(s.address,  '') AS address,
			COALESCE(s.whatsapp, '') AS whatsapp,
			COALESCE(s.telegram, '') AS telegram,
			COALESCE(s.notes,    '') AS notes,
			s.is_active,
			s.created_at::text,
			COALESCE(d.total_debt,   0) AS total_debt,
			COALESCE(d.total_amount, 0) AS total_amount,
			COALESCE(d.total_paid,   0) AS total_paid,
			COALESCE(d.unpaid_count, 0) AS unpaid_count
		FROM suppliers s
		LEFT JOIN supplier_debt d ON d.supplier_id = s.id
		WHERE s.id = $1`, id).Scan(
		&s.ID, &s.Name, &s.Phone, &s.Email,
		&s.Address, &s.WhatsApp, &s.Telegram,
		&s.Notes, &s.IsActive, &s.CreatedAt,
		&s.TotalDebt, &s.TotalAmount, &s.TotalPaid, &s.UnpaidCount,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &s, err
}

func (r *WarehouseRepo) SupplierCreate(ctx context.Context, req models.CreateSupplierRequest) (string, error) {
	id := uuid.New().String()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO suppliers (id, name, phone, email, address, whatsapp, telegram, notes)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		id,
		strings.TrimSpace(req.Name), strings.TrimSpace(req.Phone),
		strings.TrimSpace(req.Email), strings.TrimSpace(req.Address),
		strings.TrimSpace(req.WhatsApp), strings.TrimSpace(req.Telegram),
		strings.TrimSpace(req.Notes),
	)
	return id, err
}

func (r *WarehouseRepo) SupplierUpdate(ctx context.Context, id string, req models.UpdateSupplierRequest) error {
	setClauses := []string{}
	args := []interface{}{}
	i := 1

	fields := []struct{ col, val string }{
		{"phone", strings.TrimSpace(req.Phone)},
		{"email", strings.TrimSpace(req.Email)},
		{"address", strings.TrimSpace(req.Address)},
		{"whatsapp", strings.TrimSpace(req.WhatsApp)},
		{"telegram", strings.TrimSpace(req.Telegram)},
		{"notes", strings.TrimSpace(req.Notes)},
	}
	if req.Name != "" {
		setClauses = append(setClauses, fmt.Sprintf("name=$%d", i))
		args = append(args, strings.TrimSpace(req.Name))
		i++
	}
	for _, f := range fields {
		setClauses = append(setClauses, fmt.Sprintf("%s=$%d", f.col, i))
		args = append(args, f.val)
		i++
	}
	if req.IsActive != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_active=$%d", i))
		args = append(args, *req.IsActive)
		i++
	}
	if len(setClauses) == 0 {
		return nil
	}
	args = append(args, id)
	res, err := r.db.ExecContext(ctx,
		fmt.Sprintf("UPDATE suppliers SET %s WHERE id=$%d", strings.Join(setClauses, ", "), i),
		args...,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("not found")
	}
	return nil
}

func (r *WarehouseRepo) SupplierDelete(ctx context.Context, id string) (string, error) {
	var cnt int
	r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM warehouse_receipts WHERE supplier_id=$1`, id).Scan(&cnt)
	if cnt > 0 {
		_, err := r.db.ExecContext(ctx, `UPDATE suppliers SET is_active=FALSE WHERE id=$1`, id)
		return "deactivated", err
	}
	res, err := r.db.ExecContext(ctx, `DELETE FROM suppliers WHERE id=$1`, id)
	if err != nil {
		return "", err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return "", fmt.Errorf("not found")
	}
	return "deleted", nil
}

// ─── Общие платежи поставщику ─────────────────────────────────

// SupplierPaymentHistory — все платежи поставщика (общие + по накладным)
func (r *WarehouseRepo) SupplierPaymentHistory(ctx context.Context, supplierID string) ([]models.SupplierPayment, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			wp.id,
			COALESCE(wp.supplier_id::text, '') AS supplier_id,
			wp.receipt_id,
			wp.amount,
			COALESCE(wp.payment_method, 'cash') AS payment_method,
			wp.paid_at::text,
			COALESCE(wp.notes, '')              AS notes,
			COALESCE(wp.is_supplier_payment, FALSE) AS is_supplier_payment,
			wp.created_by,
			wp.created_at::text,
			COALESCE(wr.number, '')             AS receipt_number
		FROM warehouse_payments wp
		LEFT JOIN warehouse_receipts wr ON wr.id = wp.receipt_id
		WHERE wp.supplier_id = $1
		   OR (wp.receipt_id IN (
		        SELECT id FROM warehouse_receipts WHERE supplier_id = $1
		      ) AND wp.supplier_id IS NULL)
		ORDER BY wp.paid_at DESC, wp.created_at DESC`, supplierID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var payments []models.SupplierPayment
	for rows.Next() {
		var p models.SupplierPayment
		var sid string
		if err := rows.Scan(
			&p.ID, &sid, &p.ReceiptID,
			&p.Amount, &p.PaymentMethod, &p.PaidAt,
			&p.Notes, &p.IsSupplierPayment,
			&p.CreatedBy, &p.CreatedAt, &p.ReceiptNumber,
		); err != nil {
			return nil, err
		}
		if sid != "" {
			p.SupplierID = sid
		}
		payments = append(payments, p)
	}
	if payments == nil {
		payments = []models.SupplierPayment{}
	}
	return payments, nil
}

// SupplierPaymentCreate — создаёт общий платёж и распределяет по накладным
func (r *WarehouseRepo) SupplierPaymentCreate(
	ctx context.Context,
	supplierID, userID string,
	req models.CreateSupplierPaymentRequest,
) (*models.SupplierPaymentResult, error) {

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	paidAt := req.PaidAt
	if paidAt == "" {
		paidAt = "today"
	}
	method := req.PaymentMethod
	if method == "" {
		method = "cash"
	}

	// Считаем долг до платежа
	var debtBefore float64
	tx.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(total_amount - paid_amount), 0)
		FROM warehouse_receipts
		WHERE supplier_id = $1 AND payment_status != 'paid'`, supplierID).Scan(&debtBefore)

	// Получаем неоплаченные накладные ORDER BY receipt_date ASC (старые первые)
	type unpaidReceipt struct {
		id     string
		number string
		debt   float64
	}
	recRows, err := tx.QueryContext(ctx, `
		SELECT id, COALESCE(number,''), total_amount - paid_amount AS debt
		FROM warehouse_receipts
		WHERE supplier_id = $1
		  AND payment_status != 'paid'
		  AND total_amount > paid_amount
		ORDER BY receipt_date ASC, created_at ASC`, supplierID)
	if err != nil {
		return nil, err
	}
	defer recRows.Close()

	var unpaid []unpaidReceipt
	for recRows.Next() {
		var u unpaidReceipt
		recRows.Scan(&u.id, &u.number, &u.debt)
		unpaid = append(unpaid, u)
	}
	recRows.Close()

	// Распределяем платёж по накладным
	remaining := req.Amount
	distribution := []models.PaymentDistribution{}

	for _, rec := range unpaid {
		if remaining <= 0 {
			break
		}

		var apply float64
		var status string
		if remaining >= rec.debt {
			apply = rec.debt
			status = "paid"
		} else {
			apply = remaining
			status = "partial"
		}

		// Создаём платёж привязанный к накладной
		pid := uuid.New().String()
		_, err = tx.ExecContext(ctx, `
			INSERT INTO warehouse_payments
				(id, receipt_id, supplier_id, amount, payment_method, paid_at, notes, created_by, is_supplier_payment)
			VALUES ($1,$2,$3,$4,$5,$6::date,$7,$8,TRUE)`,
			pid, rec.id, supplierID, apply, method, paidAt,
			strings.TrimSpace(req.Notes), userID,
		)
		if err != nil {
			return nil, err
		}

		// Триггер обновит paid_amount и payment_status накладной автоматически
		distribution = append(distribution, models.PaymentDistribution{
			ReceiptID:     rec.id,
			ReceiptNumber: rec.number,
			Applied:       apply,
			Remaining:     rec.debt - apply,
			Status:        status,
		})

		remaining -= apply
	}

	// Если остался остаток — сохраняем как общий платёж без накладной
	if remaining > 0 {
		pid := uuid.New().String()
		_, err = tx.ExecContext(ctx, `
			INSERT INTO warehouse_payments
				(id, receipt_id, supplier_id, amount, payment_method, paid_at, notes, created_by, is_supplier_payment)
			VALUES ($1,NULL,$2,$3,$4,$5::date,$6,$7,TRUE)`,
			pid, supplierID, remaining, method, paidAt,
			strings.TrimSpace(req.Notes), userID,
		)
		if err != nil {
			return nil, err
		}
		distribution = append(distribution, models.PaymentDistribution{
			ReceiptID:     "",
			ReceiptNumber: "Переплата",
			Applied:       remaining,
			Remaining:     0,
			Status:        "overpaid",
		})
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// Считаем долг после
	var debtAfter float64
	r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(total_amount - paid_amount), 0)
		FROM warehouse_receipts
		WHERE supplier_id = $1 AND payment_status != 'paid'`, supplierID).Scan(&debtAfter)

	return &models.SupplierPaymentResult{
		TotalPaid:       req.Amount,
		TotalDebtBefore: debtBefore,
		TotalDebtAfter:  debtAfter,
		Distribution:    distribution,
	}, nil
}

func (r *WarehouseRepo) SupplierPaymentDelete(ctx context.Context, paymentID string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM warehouse_payments WHERE id=$1`, paymentID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("not found")
	}
	return nil
}

// ─── Приходные накладные ──────────────────────────────────────

func (r *WarehouseRepo) ReceiptList(ctx context.Context, supplierID, search string) ([]models.Receipt, error) {
	query := `
		SELECT
			wr.id,
			COALESCE(wr.number, '')               AS number,
			wr.supplier_id,
			COALESCE(s.name, '')                  AS supplier_name,
			wr.receipt_date::text,
			wr.total_amount,
			COALESCE(wr.paid_amount, 0)           AS paid_amount,
			COALESCE(wr.payment_status, 'unpaid') AS payment_status,
			COALESCE(wr.notes, '')                AS notes,
			wr.created_by,
			wr.created_at::text
		FROM warehouse_receipts wr
		LEFT JOIN suppliers s ON s.id = wr.supplier_id
		WHERE 1=1`

	args := []interface{}{}
	i := 1
	if supplierID != "" {
		query += fmt.Sprintf(` AND wr.supplier_id = $%d`, i)
		args = append(args, supplierID)
		i++
	}
	if search != "" {
		query += fmt.Sprintf(` AND (wr.number ILIKE $%d OR s.name ILIKE $%d)`, i, i)
		args = append(args, "%"+search+"%")
		i++
	}
	query += ` ORDER BY wr.receipt_date DESC, wr.created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var receipts []models.Receipt
	for rows.Next() {
		var rec models.Receipt
		if err := rows.Scan(
			&rec.ID, &rec.Number, &rec.SupplierID, &rec.SupplierName,
			&rec.ReceiptDate, &rec.TotalAmount, &rec.PaidAmount, &rec.PaymentStatus,
			&rec.Notes, &rec.CreatedBy, &rec.CreatedAt,
		); err != nil {
			return nil, err
		}
		rec.Items    = []models.ReceiptItem{}
		rec.Payments = []models.ReceiptPayment{}
		receipts = append(receipts, rec)
	}
	return receipts, nil
}

func (r *WarehouseRepo) ReceiptByID(ctx context.Context, id string) (*models.Receipt, error) {
	var rec models.Receipt
	err := r.db.QueryRowContext(ctx, `
		SELECT
			wr.id,
			COALESCE(wr.number, '')               AS number,
			wr.supplier_id,
			COALESCE(s.name, '')                  AS supplier_name,
			wr.receipt_date::text,
			wr.total_amount,
			COALESCE(wr.paid_amount, 0)           AS paid_amount,
			COALESCE(wr.payment_status, 'unpaid') AS payment_status,
			COALESCE(wr.notes, '')                AS notes,
			wr.created_by,
			wr.created_at::text
		FROM warehouse_receipts wr
		LEFT JOIN suppliers s ON s.id = wr.supplier_id
		WHERE wr.id = $1`, id).Scan(
		&rec.ID, &rec.Number, &rec.SupplierID, &rec.SupplierName,
		&rec.ReceiptDate, &rec.TotalAmount, &rec.PaidAmount, &rec.PaymentStatus,
		&rec.Notes, &rec.CreatedBy, &rec.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	items, err := r.receiptItems(ctx, id)
	if err != nil {
		return nil, err
	}
	rec.Items = items
	payments, err := r.ReceiptPaymentList(ctx, id)
	if err != nil {
		return nil, err
	}
	rec.Payments = payments
	return &rec, nil
}

func (r *WarehouseRepo) receiptItems(ctx context.Context, receiptID string) ([]models.ReceiptItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT ri.id, ri.receipt_id, ri.item_id,
			COALESCE(wi.name, '') AS item_name,
			COALESCE(u.name,  '') AS unit,
			ri.quantity, ri.price, ri.total,
			COALESCE(ri.notes, '') AS notes
		FROM warehouse_receipt_items ri
		JOIN warehouse_items wi ON wi.id = ri.item_id
		LEFT JOIN units u ON u.id = wi.unit_id
		WHERE ri.receipt_id = $1
		ORDER BY wi.name`, receiptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []models.ReceiptItem
	for rows.Next() {
		var it models.ReceiptItem
		if err := rows.Scan(
			&it.ID, &it.ReceiptID, &it.ItemID, &it.ItemName,
			&it.Unit, &it.Quantity, &it.Price, &it.Total, &it.Notes,
		); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	if items == nil {
		items = []models.ReceiptItem{}
	}
	return items, nil
}

func (r *WarehouseRepo) ReceiptCreate(ctx context.Context, req models.CreateReceiptRequest, userID string) (string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	receiptDate := req.ReceiptDate
	if receiptDate == "" {
		receiptDate = "today"
	}
	receiptID := uuid.New().String()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO warehouse_receipts (id, number, supplier_id, receipt_date, notes, created_by)
		VALUES ($1,$2,$3,$4::date,$5,$6)`,
		receiptID, strings.TrimSpace(req.Number), req.SupplierID,
		receiptDate, strings.TrimSpace(req.Notes), userID,
	)
	if err != nil {
		return "", err
	}
	totalAmount := 0.0
	for _, item := range req.Items {
		itemID := uuid.New().String()
		_, err = tx.ExecContext(ctx, `
			INSERT INTO warehouse_receipt_items (id, receipt_id, item_id, quantity, price, notes)
			VALUES ($1,$2,$3,$4,$5,$6)`,
			itemID, receiptID, item.ItemID, item.Quantity, item.Price,
			strings.TrimSpace(item.Notes),
		)
		if err != nil {
			return "", err
		}
		totalAmount += item.Quantity * item.Price
	}
	_, err = tx.ExecContext(ctx,
		`UPDATE warehouse_receipts SET total_amount=$1 WHERE id=$2`, totalAmount, receiptID)
	if err != nil {
		return "", err
	}
	return receiptID, tx.Commit()
}

func (r *WarehouseRepo) ReceiptUpdate(ctx context.Context, id string, req models.UpdateReceiptRequest) error {
	receiptDate := req.ReceiptDate
	if receiptDate == "" {
		receiptDate = "today"
	}
	res, err := r.db.ExecContext(ctx, `
		UPDATE warehouse_receipts
		SET number=$1, supplier_id=$2, receipt_date=$3::date, notes=$4
		WHERE id=$5`,
		strings.TrimSpace(req.Number), req.SupplierID, receiptDate,
		strings.TrimSpace(req.Notes), id,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("not found")
	}
	return nil
}

func (r *WarehouseRepo) ReceiptDelete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM warehouse_receipts WHERE id=$1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("not found")
	}
	return nil
}

func (r *WarehouseRepo) ReceiptItemAdd(ctx context.Context, receiptID string, req models.AddReceiptItemRequest) (string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	itemID := uuid.New().String()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO warehouse_receipt_items (id, receipt_id, item_id, quantity, price, notes)
		VALUES ($1,$2,$3,$4,$5,$6)`,
		itemID, receiptID, req.ItemID, req.Quantity, req.Price,
		strings.TrimSpace(req.Notes),
	)
	if err != nil {
		return "", err
	}
	if err := r.recalcReceiptTotal(ctx, tx, receiptID); err != nil {
		return "", err
	}
	return itemID, tx.Commit()
}

func (r *WarehouseRepo) ReceiptItemDelete(ctx context.Context, receiptID, itemID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx,
		`DELETE FROM warehouse_receipt_items WHERE id=$1 AND receipt_id=$2`, itemID, receiptID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("not found")
	}
	if err := r.recalcReceiptTotal(ctx, tx, receiptID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *WarehouseRepo) recalcReceiptTotal(ctx context.Context, tx *sql.Tx, receiptID string) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE warehouse_receipts
		SET total_amount = (
			SELECT COALESCE(SUM(total), 0)
			FROM warehouse_receipt_items
			WHERE receipt_id = $1
		)
		WHERE id = $1`, receiptID)
	return err
}

// ─── Платежи по накладным (конкретная накладная) ──────────────

func (r *WarehouseRepo) ReceiptPaymentList(ctx context.Context, receiptID string) ([]models.ReceiptPayment, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			id,
			receipt_id,
			supplier_id,
			amount,
			COALESCE(payment_method, 'cash') AS payment_method,
			paid_at::text,
			COALESCE(notes, '') AS notes,
			COALESCE(is_supplier_payment, FALSE) AS is_supplier_payment,
			created_by,
			created_at::text
		FROM warehouse_payments
		WHERE receipt_id = $1
		ORDER BY paid_at DESC, created_at DESC`, receiptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var payments []models.ReceiptPayment
	for rows.Next() {
		var p models.ReceiptPayment
		if err := rows.Scan(
			&p.ID, &p.ReceiptID, &p.SupplierID,
			&p.Amount, &p.PaymentMethod, &p.PaidAt,
			&p.Notes, &p.IsSupplierPayment,
			&p.CreatedBy, &p.CreatedAt,
		); err != nil {
			return nil, err
		}
		payments = append(payments, p)
	}
	if payments == nil {
		payments = []models.ReceiptPayment{}
	}
	return payments, nil
}

func (r *WarehouseRepo) ReceiptPaymentCreate(ctx context.Context, receiptID, userID string, req models.CreateReceiptPaymentRequest) (string, error) {
	paidAt := req.PaidAt
	if paidAt == "" {
		paidAt = "today"
	}
	method := req.PaymentMethod
	if method == "" {
		method = "cash"
	}
	id := uuid.New().String()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO warehouse_payments (id, receipt_id, amount, payment_method, paid_at, notes, created_by)
		VALUES ($1,$2,$3,$4,$5::date,$6,$7)`,
		id, receiptID, req.Amount, method, paidAt,
		strings.TrimSpace(req.Notes), userID,
	)
	return id, err
}

func (r *WarehouseRepo) ReceiptPaymentDelete(ctx context.Context, paymentID string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM warehouse_payments WHERE id=$1`, paymentID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("not found")
	}
	return nil
}

// helper
func warehouseItoa(i int) string {
	return strconv.Itoa(i)
}