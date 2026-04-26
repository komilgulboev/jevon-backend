package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type OutgoingInvoiceRepo struct {
	db *sql.DB
}

func NewOutgoingInvoiceRepo(db *sql.DB) *OutgoingInvoiceRepo {
	return &OutgoingInvoiceRepo{db: db}
}

// ── Модели ────────────────────────────────────────────────────

type OutgoingInvoice struct {
	ID            string                `json:"id"`
	InvoiceNumber string                `json:"invoice_number"`
	InvoiceType   string                `json:"invoice_type"`
	OrderID       string                `json:"order_id"`
	OrderNumber   int                   `json:"order_number"`
	ClientName    string                `json:"client_name"`
	Notes         string                `json:"notes"`
	TotalCost     float64               `json:"total_cost"`
	TotalPrice    float64               `json:"total_price"`
	Status        string                `json:"status"`
	CreatedBy     string                `json:"created_by"`
	CreatorName   string                `json:"creator_name"`
	ConfirmedAt   *string               `json:"confirmed_at"`
	CreatedAt     time.Time             `json:"created_at"`
	Items         []OutgoingInvoiceItem `json:"items,omitempty"`
}

type OutgoingInvoiceItem struct {
	ID         string  `json:"id"`
	InvoiceID  string  `json:"invoice_id"`
	ItemID     string  `json:"item_id"`
	ItemName   string  `json:"item_name"`
	Unit       string  `json:"unit"`
	Quantity   float64 `json:"quantity"`
	CostPrice  float64 `json:"cost_price"`
	SalePrice  float64 `json:"sale_price"`
	TotalCost  float64 `json:"total_cost"`
	TotalPrice float64 `json:"total_price"`
}

type DeficitItem struct {
	ItemID    string  `json:"item_id"`
	ItemName  string  `json:"item_name"`
	Unit      string  `json:"unit"`
	Required  float64 `json:"required"`
	Available float64 `json:"available"`
	Shortage  float64 `json:"shortage"`
}

type CreateOutgoingInvoiceItem struct {
	ItemID    string  `json:"item_id"`
	ItemName  string  `json:"item_name"`
	Unit      string  `json:"unit"`
	Quantity  float64 `json:"quantity" binding:"required"`
	SalePrice float64 `json:"sale_price"`
}

type CreateOutgoingInvoiceRequest struct {
	InvoiceType string                      `json:"invoice_type" binding:"required"`
	OrderID     string                      `json:"order_id"`
	ClientName  string                      `json:"client_name"`
	Notes       string                      `json:"notes"`
	Items       []CreateOutgoingInvoiceItem `json:"items" binding:"required"`
}

type ConfirmResult struct {
	Status       string        `json:"status"`
	DeficitItems []DeficitItem `json:"deficit_items"`
	Message      string        `json:"message"`
}

// OutgoingInvoiceItemRequest — запрос на добавление/обновление позиции
type OutgoingInvoiceItemRequest struct {
	ItemID    string  `json:"item_id"`
	ItemName  string  `json:"item_name"`
	Unit      string  `json:"unit"`
	Quantity  float64 `json:"quantity"`
	SalePrice float64 `json:"sale_price"`
}

// ── Генерация номера накладной ────────────────────────────────

func (r *OutgoingInvoiceRepo) generateNumber(ctx context.Context, invoiceType, orderID string) (string, error) {
	if invoiceType == "order" && orderID != "" {
		var orderNumber int
		err := r.db.QueryRowContext(ctx,
			`SELECT order_number FROM orders WHERE id = $1`, orderID,
		).Scan(&orderNumber)
		if err != nil {
			return "", err
		}
		var count int
		r.db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM outgoing_invoices WHERE order_id = $1`, orderID,
		).Scan(&count)
		if count == 0 {
			return fmt.Sprintf("ORD-%d", orderNumber), nil
		}
		return fmt.Sprintf("ORD-%d/%d", orderNumber, count+1), nil
	}

	var seq int
	err := r.db.QueryRowContext(ctx,
		`SELECT nextval('outgoing_invoice_ext_seq')`,
	).Scan(&seq)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("EXT-%03d", seq), nil
}

// ── CRUD ──────────────────────────────────────────────────────

func (r *OutgoingInvoiceRepo) List(ctx context.Context, status, orderID string) ([]OutgoingInvoice, error) {
	query := `
		SELECT oi.id::text, oi.invoice_number, oi.invoice_type,
		       COALESCE(oi.order_id::text,''), COALESCE(oi.order_number,0),
		       COALESCE(oi.client_name,''), COALESCE(oi.notes,''),
		       oi.total_cost, oi.total_price, oi.status,
		       COALESCE(oi.created_by::text,''),
		       COALESCE(u.full_name,''),
		       CAST(oi.confirmed_at AS TEXT),
		       oi.created_at
		FROM outgoing_invoices oi
		LEFT JOIN users u ON u.id = oi.created_by
		WHERE 1=1`

	args := []interface{}{}
	n := 1

	if status != "" {
		query += fmt.Sprintf(" AND oi.status = $%d", n)
		args = append(args, status)
		n++
	}
	if orderID != "" {
		query += fmt.Sprintf(" AND oi.order_id = $%d::uuid", n)
		args = append(args, orderID)
		n++
	}
	query += " ORDER BY oi.created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []OutgoingInvoice
	for rows.Next() {
		var inv OutgoingInvoice
		var confirmedAt sql.NullString
		rows.Scan(
			&inv.ID, &inv.InvoiceNumber, &inv.InvoiceType,
			&inv.OrderID, &inv.OrderNumber,
			&inv.ClientName, &inv.Notes,
			&inv.TotalCost, &inv.TotalPrice, &inv.Status,
			&inv.CreatedBy, &inv.CreatorName,
			&confirmedAt, &inv.CreatedAt,
		)
		if confirmedAt.Valid {
			inv.ConfirmedAt = &confirmedAt.String
		}
		result = append(result, inv)
	}
	return result, nil
}

func (r *OutgoingInvoiceRepo) GetByID(ctx context.Context, id string) (*OutgoingInvoice, error) {
	var inv OutgoingInvoice
	var confirmedAt sql.NullString

	err := r.db.QueryRowContext(ctx, `
		SELECT oi.id::text, oi.invoice_number, oi.invoice_type,
		       COALESCE(oi.order_id::text,''), COALESCE(oi.order_number,0),
		       COALESCE(oi.client_name,''), COALESCE(oi.notes,''),
		       oi.total_cost, oi.total_price, oi.status,
		       COALESCE(oi.created_by::text,''),
		       COALESCE(u.full_name,''),
		       CAST(oi.confirmed_at AS TEXT),
		       oi.created_at
		FROM outgoing_invoices oi
		LEFT JOIN users u ON u.id = oi.created_by
		WHERE oi.id = $1
	`, id).Scan(
		&inv.ID, &inv.InvoiceNumber, &inv.InvoiceType,
		&inv.OrderID, &inv.OrderNumber,
		&inv.ClientName, &inv.Notes,
		&inv.TotalCost, &inv.TotalPrice, &inv.Status,
		&inv.CreatedBy, &inv.CreatorName,
		&confirmedAt, &inv.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if confirmedAt.Valid {
		inv.ConfirmedAt = &confirmedAt.String
	}

	items, err := r.getItems(ctx, id)
	if err != nil {
		return nil, err
	}
	inv.Items = items
	return &inv, nil
}

func (r *OutgoingInvoiceRepo) getItems(ctx context.Context, invoiceID string) ([]OutgoingInvoiceItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id::text, invoice_id::text,
		       COALESCE(item_id::text, ''),
		       item_name, unit, quantity,
		       cost_price, sale_price, total_cost, total_price
		FROM outgoing_invoice_items
		WHERE invoice_id = $1
		ORDER BY created_at
	`, invoiceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []OutgoingInvoiceItem
	for rows.Next() {
		var item OutgoingInvoiceItem
		rows.Scan(
			&item.ID, &item.InvoiceID, &item.ItemID,
			&item.ItemName, &item.Unit, &item.Quantity,
			&item.CostPrice, &item.SalePrice, &item.TotalCost, &item.TotalPrice,
		)
		items = append(items, item)
	}
	return items, nil
}

// Create — создаёт накладную в статусе draft, НЕ списывает со склада.
func (r *OutgoingInvoiceRepo) Create(ctx context.Context, req CreateOutgoingInvoiceRequest, createdBy string) (*OutgoingInvoice, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	invoiceNumber, err := r.generateNumber(ctx, req.InvoiceType, req.OrderID)
	if err != nil {
		return nil, err
	}

	var orderNumber int
	if req.OrderID != "" {
		tx.QueryRowContext(ctx, `SELECT order_number FROM orders WHERE id = $1`, req.OrderID).Scan(&orderNumber)
	}

	var invoiceID string
	err = tx.QueryRowContext(ctx, `
		INSERT INTO outgoing_invoices
			(invoice_number, invoice_type, order_id, order_number,
			 client_name, notes, status, created_by)
		VALUES
			($1, $2, NULLIF($3,'')::uuid, NULLIF($4, 0),
			 NULLIF($5,''), NULLIF($6,''), 'draft', NULLIF($7,'')::uuid)
		RETURNING id
	`, invoiceNumber, req.InvoiceType, req.OrderID, orderNumber,
		req.ClientName, req.Notes, createdBy,
	).Scan(&invoiceID)
	if err != nil {
		return nil, err
	}

	var totalCost, totalPrice float64
	for _, item := range req.Items {
		var itemName, unit string
		var costPrice, salePrice float64

		if item.ItemID != "" {
			scanErr := tx.QueryRowContext(ctx, `
				SELECT
					b.name,
					COALESCE(u.name, b.unit, 'шт'),
					COALESCE(b.avg_price, 0),
					COALESCE(wi.sale_price, 0)
				FROM warehouse_balance b
				JOIN warehouse_items wi ON wi.id = b.id
				LEFT JOIN units u ON u.id = wi.unit_id
				WHERE b.id = $1
			`, item.ItemID).Scan(&itemName, &unit, &costPrice, &salePrice)
			if scanErr != nil {
				itemName  = item.ItemName
				unit      = item.Unit
				costPrice = 0
				salePrice = item.SalePrice
			}
		} else {
			itemName  = item.ItemName
			unit      = item.Unit
			if unit == "" {
				unit = "шт"
			}
			costPrice = 0
			salePrice = item.SalePrice
		}

		if item.SalePrice > 0 {
			salePrice = item.SalePrice
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO outgoing_invoice_items
				(invoice_id, item_id, item_name, unit, quantity, cost_price, sale_price)
			VALUES ($1::uuid, NULLIF($2,'')::uuid, $3, $4, $5, $6, $7)
		`, invoiceID, item.ItemID, itemName, unit, item.Quantity, costPrice, salePrice)
		if err != nil {
			return nil, err
		}

		totalCost  += item.Quantity * costPrice
		totalPrice += item.Quantity * salePrice
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE outgoing_invoices SET total_cost=$1, total_price=$2 WHERE id=$3
	`, totalCost, totalPrice, invoiceID)
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return r.GetByID(ctx, invoiceID)
}

// Confirm — проверяет остатки, списывает со склада, синхронизирует order_materials.
//
// Стратегии пометки order_materials:
//  1. По item_id (точное совпадение)
//  2. По точному имени если item_id не привязан
//  3. По похожему имени (similarity > 0.4) — исправляет опечатки и обновляет item_id
func (r *OutgoingInvoiceRepo) Confirm(ctx context.Context, id string) (*ConfirmResult, error) {
	inv, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if inv == nil {
		return nil, fmt.Errorf("накладная не найдена")
	}
	if inv.Status != "draft" && inv.Status != "pending_purchase" {
		return nil, fmt.Errorf("нельзя подтвердить накладную со статусом %s", inv.Status)
	}

	// Проверяем остатки только по позициям привязанным к складу
	var deficits []DeficitItem
	for _, item := range inv.Items {
		if item.ItemID == "" {
			continue
		}
		var balance float64
		var unit string
		err := r.db.QueryRowContext(ctx, `
			SELECT COALESCE(b.balance, 0), COALESCE(u.name, b.unit, 'шт')
			FROM warehouse_balance b
			JOIN warehouse_items wi ON wi.id = b.id
			LEFT JOIN units u ON u.id = wi.unit_id
			WHERE b.id = $1
		`, item.ItemID).Scan(&balance, &unit)
		if err != nil {
			deficits = append(deficits, DeficitItem{
				ItemID:    item.ItemID,
				ItemName:  item.ItemName,
				Unit:      item.Unit,
				Required:  item.Quantity,
				Available: 0,
				Shortage:  item.Quantity,
			})
			continue
		}
		if balance < item.Quantity {
			deficits = append(deficits, DeficitItem{
				ItemID:    item.ItemID,
				ItemName:  item.ItemName,
				Unit:      unit,
				Required:  item.Quantity,
				Available: balance,
				Shortage:  item.Quantity - balance,
			})
		}
	}

	if len(deficits) > 0 {
		_, err = r.db.ExecContext(ctx, `
			UPDATE outgoing_invoices
			SET status='pending_purchase', updated_at=NOW()
			WHERE id=$1
		`, id)
		if err != nil {
			return nil, err
		}
		return &ConfirmResult{
			Status:       "pending_purchase",
			DeficitItems: deficits,
			Message:      "Недостаточно товаров на складе. Накладная переведена в статус «Ожидание закупки».",
		}, nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Списываем со склада (только позиции с item_id)
	for _, item := range inv.Items {
		if item.ItemID == "" {
			continue
		}

		var costPrice float64
		tx.QueryRowContext(ctx,
			`SELECT COALESCE(avg_price,0) FROM warehouse_balance WHERE id=$1`,
			item.ItemID,
		).Scan(&costPrice)

		_, err = tx.ExecContext(ctx, `
			INSERT INTO warehouse_expenses
				(item_id, quantity, price, order_id, expense_date, notes, created_by)
			VALUES
				($1::uuid, $2, $3,
				 NULLIF($4,'')::uuid,
				 CURRENT_DATE,
				 $5,
				 NULLIF($6,'')::uuid)
		`, item.ItemID, item.Quantity, costPrice,
			inv.OrderID,
			"Расходная накладная №"+inv.InvoiceNumber,
			inv.CreatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("ошибка списания %s: %w", item.ItemName, err)
		}
	}

	// Синхронизируем order_materials тремя стратегиями
	if inv.OrderID != "" {
		for _, item := range inv.Items {

			// Стратегия 1 — по item_id (точное совпадение)
			if item.ItemID != "" {
				tx.ExecContext(ctx, `
					UPDATE order_materials
					SET invoice_id     = $1::uuid,
					    invoice_number = $2
					WHERE order_id   = $3::uuid
					  AND item_id    = $4::uuid
					  AND invoice_id IS NULL
				`, inv.ID, inv.InvoiceNumber, inv.OrderID, item.ItemID)
			}

			// Стратегия 2 — по точному имени (для вручную добавленных без item_id)
			tx.ExecContext(ctx, `
				UPDATE order_materials
				SET invoice_id     = $1::uuid,
				    invoice_number = $2
				WHERE order_id          = $3::uuid
				  AND invoice_id        IS NULL
				  AND (item_id IS NULL OR item_id::text = '')
				  AND LOWER(TRIM(name)) = LOWER(TRIM($4))
			`, inv.ID, inv.InvoiceNumber, inv.OrderID, item.ItemName)

			// Стратегия 3 — по похожему имени (similarity) + обновляем name и item_id
			// Исправляет опечатки (напр. "Лаак 10%" → "Лак 10%") и привязывает к номенклатуре
			if item.ItemID != "" {
				tx.ExecContext(ctx, `
					UPDATE order_materials
					SET name           = $1,
					    item_id        = $2::uuid,
					    invoice_id     = $3::uuid,
					    invoice_number = $4
					WHERE order_id   = $5::uuid
					  AND invoice_id IS NULL
					  AND (item_id IS NULL OR item_id::text = '')
					  AND similarity(LOWER(TRIM(name)), LOWER(TRIM($1))) > 0.4
				`, item.ItemName, item.ItemID, inv.ID, inv.InvoiceNumber, inv.OrderID)
			}
		}
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE outgoing_invoices
		SET status='confirmed', confirmed_at=NOW(), updated_at=NOW()
		WHERE id=$1
	`, id)
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return &ConfirmResult{
		Status:  "confirmed",
		Message: "Накладная подтверждена, товары списаны со склада.",
	}, nil
}

// Cancel — отменяет накладную. Возвращает товар и снимает пометку если была confirmed.
func (r *OutgoingInvoiceRepo) Cancel(ctx context.Context, id string) error {
	inv, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if inv == nil {
		return fmt.Errorf("накладная не найдена")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if inv.Status == "confirmed" {
		// Возвращаем товар на склад
		_, err = tx.ExecContext(ctx, `
			DELETE FROM warehouse_expenses
			WHERE notes = $1
		`, "Расходная накладная №"+inv.InvoiceNumber)
		if err != nil {
			return err
		}

		// Снимаем пометку "получено" с order_materials
		if inv.OrderID != "" {
			tx.ExecContext(ctx, `
				UPDATE order_materials
				SET invoice_id     = NULL,
				    invoice_number = NULL
				WHERE order_id   = $1::uuid
				  AND invoice_id = $2::uuid
			`, inv.OrderID, inv.ID)
		}
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE outgoing_invoices SET status='cancelled', updated_at=NOW() WHERE id=$1
	`, id)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// ── Управление позициями накладной ────────────────────────────

// ItemAdd — добавляет новую позицию в накладную (только draft)
func (r *OutgoingInvoiceRepo) ItemAdd(ctx context.Context, invoiceID, addedBy string, req OutgoingInvoiceItemRequest) (*OutgoingInvoice, error) {
	inv, err := r.GetByID(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	if inv == nil {
		return nil, fmt.Errorf("накладная не найдена")
	}
	if inv.Status != "draft" {
		return nil, fmt.Errorf("нельзя изменять подтверждённую накладную")
	}

	if req.Unit == "" {
		req.Unit = "шт"
	}

	var costPrice float64
	if req.ItemID != "" {
		r.db.QueryRowContext(ctx,
			`SELECT COALESCE(avg_price, 0) FROM warehouse_balance WHERE id = $1`,
			req.ItemID,
		).Scan(&costPrice)
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO outgoing_invoice_items
			(invoice_id, item_id, item_name, unit, quantity, cost_price, sale_price)
		VALUES ($1::uuid, NULLIF($2,'')::uuid, $3, $4, $5, $6, $7)
	`, invoiceID, req.ItemID, req.ItemName, req.Unit, req.Quantity, costPrice, req.SalePrice)
	if err != nil {
		return nil, err
	}

	r.recalcTotals(ctx, invoiceID)
	return r.GetByID(ctx, invoiceID)
}

// ItemUpdate — обновляет позицию накладной (привязка к номенклатуре, количество, цена)
func (r *OutgoingInvoiceRepo) ItemUpdate(ctx context.Context, invoiceID, itemID string, req OutgoingInvoiceItemRequest) (*OutgoingInvoice, error) {
	inv, err := r.GetByID(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	if inv == nil {
		return nil, fmt.Errorf("накладная не найдена")
	}
	if inv.Status != "draft" && inv.Status != "pending_purchase" {
		return nil, fmt.Errorf("нельзя изменять подтверждённую накладную")
	}

	var costPrice float64
	if req.ItemID != "" {
		r.db.QueryRowContext(ctx,
			`SELECT COALESCE(avg_price, 0) FROM warehouse_balance WHERE id = $1`,
			req.ItemID,
		).Scan(&costPrice)
	}

	_, err = r.db.ExecContext(ctx, `
		UPDATE outgoing_invoice_items SET
			item_id    = NULLIF($1,'')::uuid,
			item_name  = $2,
			unit       = $3,
			quantity   = $4,
			cost_price = $5,
			sale_price = $6
		WHERE id = $7::uuid AND invoice_id = $8::uuid
	`, req.ItemID, req.ItemName, req.Unit, req.Quantity, costPrice, req.SalePrice, itemID, invoiceID)
	if err != nil {
		return nil, err
	}

	r.recalcTotals(ctx, invoiceID)
	return r.GetByID(ctx, invoiceID)
}

// ItemDelete — удаляет позицию из накладной (только draft/pending_purchase)
func (r *OutgoingInvoiceRepo) ItemDelete(ctx context.Context, invoiceID, itemID string) (*OutgoingInvoice, error) {
	inv, err := r.GetByID(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	if inv == nil {
		return nil, fmt.Errorf("накладная не найдена")
	}
	if inv.Status != "draft" && inv.Status != "pending_purchase" {
		return nil, fmt.Errorf("нельзя изменять подтверждённую накладную")
	}

	_, err = r.db.ExecContext(ctx, `
		DELETE FROM outgoing_invoice_items
		WHERE id = $1::uuid AND invoice_id = $2::uuid
	`, itemID, invoiceID)
	if err != nil {
		return nil, err
	}

	r.recalcTotals(ctx, invoiceID)
	return r.GetByID(ctx, invoiceID)
}

// recalcTotals — пересчитывает total_cost и total_price накладной
func (r *OutgoingInvoiceRepo) recalcTotals(ctx context.Context, invoiceID string) {
	r.db.ExecContext(ctx, `
		UPDATE outgoing_invoices SET
			total_cost  = (
				SELECT COALESCE(SUM(quantity * cost_price), 0)
				FROM outgoing_invoice_items WHERE invoice_id = $1::uuid
			),
			total_price = (
				SELECT COALESCE(SUM(total_price), 0)
				FROM outgoing_invoice_items WHERE invoice_id = $1::uuid
			)
		WHERE id = $1::uuid
	`, invoiceID)
}