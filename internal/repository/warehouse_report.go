package repository

import (
	"context"
	"fmt"
)

// ─── Отчёт по складу ─────────────────────────────────────────

type WarehouseReportRow struct {
	Date          string  `json:"date"`
	Type          string  `json:"type"` // "in" | "out"
	ItemID        string  `json:"item_id"`
	ItemName      string  `json:"item_name"`
	Category      string  `json:"category"`
	Unit          string  `json:"unit"`
	Quantity      float64 `json:"quantity"`
	Price         float64 `json:"price"`
	Total         float64 `json:"total"`
	ReceiptNumber string  `json:"receipt_number"`
	SupplierName  string  `json:"supplier_name"`
	OrderNumber   string  `json:"order_number"`
	OrderTitle    string  `json:"order_title"`
	Notes         string  `json:"notes"`
}

type WarehouseReportSummary struct {
	TotalIn     float64 `json:"total_in"`
	TotalOut    float64 `json:"total_out"`
	TotalInQty  float64 `json:"total_in_qty"`
	TotalOutQty float64 `json:"total_out_qty"`
	RowsIn      int     `json:"rows_in"`
	RowsOut     int     `json:"rows_out"`
}

type WarehouseReport struct {
	Rows    []WarehouseReportRow   `json:"rows"`
	Summary WarehouseReportSummary `json:"summary"`
}

func (r *WarehouseRepo) Report(ctx context.Context, dateFrom, dateTo, itemID, category string) (*WarehouseReport, error) {
	// ── Приходы ──
	inQuery := `
		SELECT
			ri.id,
			wr.receipt_date::text     AS date,
			'in'                      AS type,
			ri.item_id::text          AS item_id,
			wi.name                   AS item_name,
			COALESCE(wi.category, '') AS category,
			COALESCE(u.name, '')      AS unit,
			ri.quantity,
			ri.price,
			ri.total,
			COALESCE(wr.number, '')   AS receipt_number,
			COALESCE(s.name, '')      AS supplier_name,
			''                        AS order_number,
			''                        AS order_title,
			COALESCE(ri.notes, '')    AS notes
		FROM warehouse_receipt_items ri
		JOIN warehouse_receipts wr ON wr.id = ri.receipt_id
		JOIN warehouse_items wi    ON wi.id = ri.item_id
		LEFT JOIN units u          ON u.id = wi.unit_id
		LEFT JOIN suppliers s      ON s.id = wr.supplier_id
		WHERE 1=1`

	inArgs := []interface{}{}
	ii := 1
	if dateFrom != "" {
		inQuery += fmt.Sprintf(` AND wr.receipt_date >= $%d::date`, ii)
		inArgs = append(inArgs, dateFrom)
		ii++
	}
	if dateTo != "" {
		inQuery += fmt.Sprintf(` AND wr.receipt_date <= $%d::date`, ii)
		inArgs = append(inArgs, dateTo)
		ii++
	}
	if itemID != "" {
		inQuery += fmt.Sprintf(` AND ri.item_id = $%d::uuid`, ii)
		inArgs = append(inArgs, itemID)
		ii++
	}
	if category != "" {
		inQuery += fmt.Sprintf(` AND wi.category = $%d`, ii)
		inArgs = append(inArgs, category)
		ii++
	}
	_ = ii

	// ── Расходы ──
	outQuery := `
		SELECT
			we.id,
			we.expense_date::text        AS date,
			'out'                        AS type,
			we.item_id::text             AS item_id,
			wi.name                      AS item_name,
			COALESCE(wi.category, '')    AS category,
			COALESCE(u.name, '')         AS unit,
			we.quantity,
			COALESCE(we.price, 0)        AS price,
			COALESCE(we.quantity * we.price, 0) AS total,
			''                           AS receipt_number,
			''                           AS supplier_name,
			COALESCE(
				CASE
					WHEN o.order_type = 'workshop' THEN 'В-' || o.order_number::text
					WHEN o.order_type = 'external' THEN 'Б-' || o.order_number::text
					ELSE o.order_number::text
				END,
				''
			)                            AS order_number,
			COALESCE(o.title, '')        AS order_title,
			COALESCE(we.notes, '')       AS notes
		FROM warehouse_expenses we
		JOIN warehouse_items wi    ON wi.id = we.item_id
		LEFT JOIN units u          ON u.id = wi.unit_id
		LEFT JOIN orders o         ON o.id = we.order_id
		WHERE 1=1`

	outArgs := []interface{}{}
	oi := 1
	if dateFrom != "" {
		outQuery += fmt.Sprintf(` AND we.expense_date >= $%d::date`, oi)
		outArgs = append(outArgs, dateFrom)
		oi++
	}
	if dateTo != "" {
		outQuery += fmt.Sprintf(` AND we.expense_date <= $%d::date`, oi)
		outArgs = append(outArgs, dateTo)
		oi++
	}
	if itemID != "" {
		outQuery += fmt.Sprintf(` AND we.item_id = $%d::uuid`, oi)
		outArgs = append(outArgs, itemID)
		oi++
	}
	if category != "" {
		outQuery += fmt.Sprintf(` AND wi.category = $%d`, oi)
		outArgs = append(outArgs, category)
		oi++
	}
	_ = oi

	var rows []WarehouseReportRow
	var summary WarehouseReportSummary

	// Загружаем приходы
	inRows, err := r.db.QueryContext(ctx, inQuery, inArgs...)
	if err != nil {
		return nil, fmt.Errorf("receipts query: %w", err)
	}
	defer inRows.Close()
	for inRows.Next() {
		var row WarehouseReportRow
		var id string
		if err := inRows.Scan(
			&id, &row.Date, &row.Type,
			&row.ItemID, &row.ItemName, &row.Category, &row.Unit,
			&row.Quantity, &row.Price, &row.Total,
			&row.ReceiptNumber, &row.SupplierName,
			&row.OrderNumber, &row.OrderTitle, &row.Notes,
		); err != nil {
			return nil, err
		}
		rows = append(rows, row)
		summary.TotalIn += row.Total
		summary.TotalInQty += row.Quantity
		summary.RowsIn++
	}

	// Загружаем расходы
	outRows, err := r.db.QueryContext(ctx, outQuery, outArgs...)
	if err != nil {
		return nil, fmt.Errorf("expenses query: %w", err)
	}
	defer outRows.Close()
	for outRows.Next() {
		var row WarehouseReportRow
		var id string
		if err := outRows.Scan(
			&id, &row.Date, &row.Type,
			&row.ItemID, &row.ItemName, &row.Category, &row.Unit,
			&row.Quantity, &row.Price, &row.Total,
			&row.ReceiptNumber, &row.SupplierName,
			&row.OrderNumber, &row.OrderTitle, &row.Notes,
		); err != nil {
			return nil, err
		}
		rows = append(rows, row)
		summary.TotalOut += row.Total
		summary.TotalOutQty += row.Quantity
		summary.RowsOut++
	}

	// Сортируем по дате
	for i := 0; i < len(rows); i++ {
		for j := i + 1; j < len(rows); j++ {
			if rows[i].Date > rows[j].Date {
				rows[i], rows[j] = rows[j], rows[i]
			}
		}
	}

	if rows == nil {
		rows = []WarehouseReportRow{}
	}

	return &WarehouseReport{Rows: rows, Summary: summary}, nil
}