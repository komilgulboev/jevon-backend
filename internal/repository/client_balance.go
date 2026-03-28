package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ─── Модели ───────────────────────────────────────────────────

type ClientDebt struct {
	ClientID      string  `db:"client_id"     json:"client_id"`
	ClientName    string  `db:"client_name"   json:"client_name"`
	Phone         string  `db:"phone"         json:"phone"`
	TotalOrders   int     `db:"total_orders"  json:"total_orders"`
	TotalAmount   float64 `db:"total_amount"  json:"total_amount"`
	TotalPaid     float64 `db:"total_paid"    json:"total_paid"`
	TotalDebt     float64 `db:"total_debt"    json:"total_debt"`
	CreditBalance float64 `db:"credit_balance" json:"credit_balance"`
	NetDebt       float64 `db:"net_debt"      json:"net_debt"`
}

type ClientPaymentRecord struct {
	ID                string  `json:"id"`
	ClientID          string  `json:"client_id"`
	OrderID           string  `json:"order_id"`
	OrderNumber       int     `json:"order_number"`
	OrderTitle        string  `json:"order_title"`
	Amount            float64 `json:"amount"`
	PaymentMethod     string  `json:"payment_method"`
	PaidAt            string  `json:"paid_at"`
	Notes             string  `json:"notes"`
	IsClientPayment   bool    `json:"is_client_payment"`
	CreatedAt         string  `json:"created_at"`
}

type ClientOrder struct {
	ID            string  `json:"id"`
	OrderNumber   int     `json:"order_number"`
	OrderType     string  `json:"order_type"`
	Title         string  `json:"title"`
	Status        string  `json:"status"`
	FinalCost     float64 `json:"final_cost"`
	PaidAmount    float64 `json:"paid_amount"`
	Debt          float64 `json:"debt"`
	PaymentStatus string  `json:"payment_status"`
	CreatedAt     string  `json:"created_at"`
}

type CreateClientPaymentRequest struct {
	Amount        float64 `json:"amount"         binding:"required,gt=0"`
	PaymentMethod string  `json:"payment_method" binding:"required"`
	PaidAt        string  `json:"paid_at"`
	Notes         string  `json:"notes"`
}

type ClientPaymentDistribution struct {
	OrderID     string  `json:"order_id"`
	OrderNumber int     `json:"order_number"`
	OrderTitle  string  `json:"order_title"`
	Applied     float64 `json:"applied"`
	Remaining   float64 `json:"remaining"`
	Status      string  `json:"status"` // paid / partial / balance
}

type ClientPaymentResult struct {
	TotalPaid        float64                     `json:"total_paid"`
	DebtBefore       float64                     `json:"debt_before"`
	DebtAfter        float64                     `json:"debt_after"`
	CreditBalance    float64                     `json:"credit_balance"`
	Distribution     []ClientPaymentDistribution `json:"distribution"`
}

// ─── Репозиторий ──────────────────────────────────────────────

type ClientBalanceRepo struct {
	db *sql.DB
}

func NewClientBalanceRepo(db *sql.DB) *ClientBalanceRepo {
	return &ClientBalanceRepo{db: db}
}

// ClientDebtList — список клиентов с долгом/балансом
func (r *ClientBalanceRepo) ClientDebtList(ctx context.Context, search, debtFilter string) ([]ClientDebt, error) {
	query := `
		SELECT
			cd.client_id,
			cd.client_name,
			COALESCE(cd.phone, '') AS phone,
			cd.total_orders,
			cd.total_amount,
			cd.total_paid,
			cd.total_debt,
			cd.credit_balance,
			cd.net_debt
		FROM client_debt cd
		WHERE 1=1`

	args := []interface{}{}
	i := 1

	if search != "" {
		query += fmt.Sprintf(` AND (cd.client_name ILIKE $%d OR cd.phone ILIKE $%d)`, i, i)
		args = append(args, "%"+search+"%")
		i++
	}

	switch debtFilter {
	case "debt":
		query += ` AND cd.net_debt > 0`
	case "credit":
		query += ` AND cd.credit_balance > 0 AND cd.net_debt <= 0`
	case "clear":
		query += ` AND cd.net_debt <= 0 AND cd.credit_balance <= 0`
	}

	query += ` ORDER BY cd.net_debt DESC, cd.client_name`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ClientDebt
	for rows.Next() {
		var cd ClientDebt
		if err := rows.Scan(
			&cd.ClientID, &cd.ClientName, &cd.Phone,
			&cd.TotalOrders, &cd.TotalAmount, &cd.TotalPaid,
			&cd.TotalDebt, &cd.CreditBalance, &cd.NetDebt,
		); err != nil {
			return nil, err
		}
		result = append(result, cd)
	}
	if result == nil {
		result = []ClientDebt{}
	}
	return result, nil
}

// ClientOrders — заказы клиента с долгом по каждому
func (r *ClientBalanceRepo) ClientOrders(ctx context.Context, clientID string) ([]ClientOrder, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			id::text,
			order_number,
			order_type,
			title,
			status,
			COALESCE(final_cost, estimated_cost, 0)           AS final_cost,
			COALESCE(paid_amount, 0)                          AS paid_amount,
			GREATEST(0,
				COALESCE(final_cost, estimated_cost, 0)
				- COALESCE(paid_amount, 0))                   AS debt,
			payment_status,
			created_at::text
		FROM orders
		WHERE client_id = $1
		  AND status != 'cancelled'
		ORDER BY created_at DESC`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ClientOrder
	for rows.Next() {
		var o ClientOrder
		if err := rows.Scan(
			&o.ID, &o.OrderNumber, &o.OrderType, &o.Title,
			&o.Status, &o.FinalCost, &o.PaidAmount, &o.Debt,
			&o.PaymentStatus, &o.CreatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, o)
	}
	if result == nil {
		result = []ClientOrder{}
	}
	return result, nil
}

// ClientPaymentHistory — история всех платежей клиента
func (r *ClientBalanceRepo) ClientPaymentHistory(ctx context.Context, clientID string) ([]ClientPaymentRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			p.id::text,
			COALESCE(p.client_id::text, '')        AS client_id,
			COALESCE(p.order_id::text, '')         AS order_id,
			COALESCE(o.order_number, 0)            AS order_number,
			COALESCE(o.title, '')                  AS order_title,
			p.amount,
			COALESCE(p.payment_method, p.payment_type, 'cash') AS payment_method,
			p.paid_at::text,
			COALESCE(p.notes, '')                  AS notes,
			COALESCE(p.is_client_payment, FALSE)   AS is_client_payment,
			p.created_at::text
		FROM order_payments p
		LEFT JOIN orders o ON o.id = p.order_id
		WHERE p.client_id = $1
		   OR p.order_id IN (
		        SELECT id FROM orders WHERE client_id = $1
		      )
		ORDER BY p.paid_at DESC, p.created_at DESC`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ClientPaymentRecord
	for rows.Next() {
		var p ClientPaymentRecord
		if err := rows.Scan(
			&p.ID, &p.ClientID, &p.OrderID,
			&p.OrderNumber, &p.OrderTitle,
			&p.Amount, &p.PaymentMethod,
			&p.PaidAt, &p.Notes,
			&p.IsClientPayment, &p.CreatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	if result == nil {
		result = []ClientPaymentRecord{}
	}
	return result, nil
}

// ClientPaymentCreate — вносит платёж и распределяет по заказам
func (r *ClientBalanceRepo) ClientPaymentCreate(
	ctx context.Context,
	clientID, userID string,
	req CreateClientPaymentRequest,
) (*ClientPaymentResult, error) {

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

	// Долг до платежа
	var debtBefore float64
	tx.QueryRowContext(ctx, `
		SELECT COALESCE(net_debt, 0) FROM client_debt WHERE client_id = $1`, clientID).Scan(&debtBefore)

	// Берём неоплаченные заказы от старых к новым
	type unpaidOrder struct {
		id     string
		number int
		title  string
		debt   float64
	}
	oRows, err := tx.QueryContext(ctx, `
		SELECT
			id::text,
			order_number,
			title,
			GREATEST(0, COALESCE(final_cost, estimated_cost, 0) - COALESCE(paid_amount, 0)) AS debt
		FROM orders
		WHERE client_id = $1
		  AND status != 'cancelled'
		  AND COALESCE(final_cost, estimated_cost, 0) > COALESCE(paid_amount, 0)
		ORDER BY created_at ASC`, clientID)
	if err != nil {
		return nil, err
	}
	defer oRows.Close()

	var unpaid []unpaidOrder
	for oRows.Next() {
		var u unpaidOrder
		oRows.Scan(&u.id, &u.number, &u.title, &u.debt)
		unpaid = append(unpaid, u)
	}
	oRows.Close()

	remaining := req.Amount
	distribution := []ClientPaymentDistribution{}

	for _, o := range unpaid {
		if remaining <= 0 {
			break
		}

		var apply float64
		var status string
		if remaining >= o.debt {
			apply = o.debt
			status = "paid"
		} else {
			apply = remaining
			status = "partial"
		}

		// Вставляем платёж
		pid := uuid.New().String()
		_, err = tx.ExecContext(ctx, `
			INSERT INTO order_payments
				(id, order_id, client_id, amount, payment_method, paid_at, notes, received_by, is_client_payment)
			VALUES ($1,$2,$3,$4,$5,$6::date,$7,$8,TRUE)`,
			pid, o.id, clientID, apply, method, paidAt,
			strings.TrimSpace(req.Notes), userID,
		)
		if err != nil {
			return nil, err
		}

		// Обновляем paid_amount заказа
		_, err = tx.ExecContext(ctx, `
			UPDATE orders
			SET paid_amount = COALESCE(paid_amount, 0) + $1,
			    payment_status = CASE
			        WHEN COALESCE(paid_amount, 0) + $1 >= COALESCE(final_cost, estimated_cost, 0) THEN 'paid'
			        WHEN COALESCE(paid_amount, 0) + $1 > 0 THEN 'partial'
			        ELSE 'unpaid'
			    END
			WHERE id = $2`, apply, o.id)
		if err != nil {
			return nil, err
		}

		// Логируем в историю заказа
		tx.ExecContext(ctx, `
			INSERT INTO order_history (order_id, from_stage, to_stage, changed_by, comment)
			VALUES ($1::uuid, 'payment', 'payment', NULLIF($2,'')::uuid, $3)`,
			o.id, userID,
			fmt.Sprintf("💰 Оплата (общий расчёт): %.0f сом. | %s", apply, methodLabel(method)),
		)

		distribution = append(distribution, ClientPaymentDistribution{
			OrderID:     o.id,
			OrderNumber: o.number,
			OrderTitle:  o.title,
			Applied:     apply,
			Remaining:   o.debt - apply,
			Status:      status,
		})

		remaining -= apply
	}

	// Остаток — на баланс клиента
	if remaining > 0 {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO client_balance (id, client_id, balance)
			VALUES ($1, $2, $3)
			ON CONFLICT (client_id)
			DO UPDATE SET balance = client_balance.balance + $3, updated_at = NOW()`,
			uuid.New().String(), clientID, remaining,
		)
		if err != nil {
			return nil, err
		}

		// Сохраняем как платёж без заказа
		pid := uuid.New().String()
		_, err = tx.ExecContext(ctx, `
			INSERT INTO order_payments
				(id, order_id, client_id, amount, payment_method, paid_at, notes, received_by, is_client_payment)
			VALUES ($1,NULL,$2,$3,$4,$5::date,$6,$7,TRUE)`,
			pid, clientID, remaining, method, paidAt,
			strings.TrimSpace(req.Notes), userID,
		)
		if err != nil {
			return nil, err
		}

		distribution = append(distribution, ClientPaymentDistribution{
			OrderID:    "",
			OrderTitle: "На баланс",
			Applied:    remaining,
			Remaining:  0,
			Status:     "balance",
		})
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// Долг после платежа
	var debtAfter, creditBalance float64
	r.db.QueryRowContext(ctx, `
		SELECT COALESCE(net_debt, 0), COALESCE(credit_balance, 0)
		FROM client_debt WHERE client_id = $1`, clientID).Scan(&debtAfter, &creditBalance)

	return &ClientPaymentResult{
		TotalPaid:     req.Amount,
		DebtBefore:    debtBefore,
		DebtAfter:     debtAfter,
		CreditBalance: creditBalance,
		Distribution:  distribution,
	}, nil
}

// ClientPaymentDelete — удалить платёж и откатить paid_amount заказа
func (r *ClientBalanceRepo) ClientPaymentDelete(ctx context.Context, paymentID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Получаем данные платежа
	var orderID sql.NullString
	var clientID string
	var amount float64
	var isClientPayment bool
	err = tx.QueryRowContext(ctx, `
		SELECT order_id, COALESCE(client_id::text,''), amount, COALESCE(is_client_payment, FALSE)
		FROM order_payments WHERE id = $1`, paymentID).Scan(
		&orderID, &clientID, &amount, &isClientPayment,
	)
	if err == sql.ErrNoRows {
		return fmt.Errorf("not found")
	}
	if err != nil {
		return err
	}

	// Удаляем платёж
	_, err = tx.ExecContext(ctx, `DELETE FROM order_payments WHERE id = $1`, paymentID)
	if err != nil {
		return err
	}

	// Откатываем paid_amount заказа
	if orderID.Valid && orderID.String != "" {
		_, err = tx.ExecContext(ctx, `
			UPDATE orders
			SET paid_amount = GREATEST(0, COALESCE(paid_amount, 0) - $1),
			    payment_status = CASE
			        WHEN GREATEST(0, COALESCE(paid_amount, 0) - $1) <= 0 THEN 'unpaid'
			        WHEN GREATEST(0, COALESCE(paid_amount, 0) - $1) >= COALESCE(final_cost, estimated_cost, 0) THEN 'paid'
			        ELSE 'partial'
			    END
			WHERE id = $2`, amount, orderID.String)
		if err != nil {
			return err
		}
	}

	// Откатываем баланс клиента если платёж был на баланс
	if isClientPayment && clientID != "" && (!orderID.Valid || orderID.String == "") {
		tx.ExecContext(ctx, `
			UPDATE client_balance
			SET balance = GREATEST(0, balance - $1), updated_at = NOW()
			WHERE client_id = $2`, amount, clientID)
	}

	return tx.Commit()
}

// ── helper ────────────────────────────────────────────────────

func methodLabel(method string) string {
	labels := map[string]string{
		"cash":   "Наличные",
		"card":   "Карта",
		"bank":   "Банковский перевод",
		"wallet": "Кошелёк",
		"other":  "Другое",
	}
	if l, ok := labels[method]; ok {
		return l
	}
	return method
}