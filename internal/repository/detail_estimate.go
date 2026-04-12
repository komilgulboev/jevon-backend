package repository

import (
	"context"
	"database/sql"
	"fmt"
)

var ServiceTypeLabels = map[string]string{
	"cnc":      "ЧПУ",
	"painting": "Покраска",
	"soft":     "Мягкая мебель",
	"cutting":  "Распил",
}

var ServiceTypeSubtitles = map[string]string{
	"cnc":      "От идеи к идеальной детали.",
	"painting": "От эскиза до идеального цвета.",
	"soft":     "От идеи к идеальной детали.",
	"cutting":  "От чертежа до готовой детали.",
}

type DetailEstimateRepo struct {
	db *sql.DB
}

func NewDetailEstimateRepo(db *sql.DB) *DetailEstimateRepo {
	return &DetailEstimateRepo{db: db}
}

type DetailEstimateRow struct {
	ID          string  `json:"id"`
	OrderID     string  `json:"order_id"`
	ServiceType string  `json:"service_type"`
	RowOrder    int     `json:"row_order"`
	DetailName  string  `json:"detail_name"`
	WidthMM     float64 `json:"width_mm"`
	HeightMM    float64 `json:"height_mm"`
	Quantity    int     `json:"quantity"`
	AreaM2      float64 `json:"area_m2"`
	UnitPrice   float64 `json:"unit_price"`
	TotalPrice  float64 `json:"total_price"`
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
}

type EstimateSettings struct {
	ID              string `json:"id"`
	OrderID         string `json:"order_id"`
	ServiceType     string `json:"service_type"`
	SectionTitle    string `json:"section_title"`
	SectionSubtitle string `json:"section_subtitle"`
	Deadline        string `json:"deadline"`
	DeliveryDate    string `json:"delivery_date"`
	Notes           string `json:"notes"`
}

type SaveDetailEstimateRequest struct {
	ServiceType string                   `json:"service_type" binding:"required"`
	Settings    EstimateSettings         `json:"settings"`
	Rows        []DetailEstimateRowInput `json:"rows"`
}

type DetailEstimateRowInput struct {
	DetailName  string  `json:"detail_name"`
	WidthMM     float64 `json:"width_mm"`
	HeightMM    float64 `json:"height_mm"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
}

type DetailEstimateSection struct {
	ServiceType string              `json:"service_type"`
	Label       string              `json:"label"`
	Settings    *EstimateSettings   `json:"settings"`
	Rows        []DetailEstimateRow `json:"rows"`
	TotalAreaM2 float64             `json:"total_area_m2"`
	TotalPrice  float64             `json:"total_price"`
}

// GetByOrder — возвращает все разделы сметы заказа
func (r *DetailEstimateRepo) GetByOrder(ctx context.Context, orderID string) ([]DetailEstimateSection, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			id, order_id, service_type, row_order,
			detail_name, width_mm, height_mm, quantity, area_m2,
			unit_price, total_price,
			COALESCE(product_id::text, '')   AS product_id,
			COALESCE(product_name, '')        AS product_name
		FROM order_detail_estimates
		WHERE order_id = $1::uuid
		ORDER BY service_type, row_order
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sectionsMap := map[string]*DetailEstimateSection{}
	var order []string

	for rows.Next() {
		var row DetailEstimateRow
		if err := rows.Scan(
			&row.ID, &row.OrderID, &row.ServiceType, &row.RowOrder,
			&row.DetailName, &row.WidthMM, &row.HeightMM, &row.Quantity,
			&row.AreaM2, &row.UnitPrice, &row.TotalPrice,
			&row.ProductID, &row.ProductName,
		); err != nil {
			return nil, err
		}
		if _, ok := sectionsMap[row.ServiceType]; !ok {
			sectionsMap[row.ServiceType] = &DetailEstimateSection{
				ServiceType: row.ServiceType,
				Label:       ServiceTypeLabels[row.ServiceType],
				Rows:        []DetailEstimateRow{},
			}
			order = append(order, row.ServiceType)
		}
		s := sectionsMap[row.ServiceType]
		s.Rows = append(s.Rows, row)
		s.TotalAreaM2 += row.AreaM2
		s.TotalPrice += row.TotalPrice
	}

	// Загружаем настройки
	sRows, err := r.db.QueryContext(ctx, `
		SELECT id, order_id, service_type,
			COALESCE(section_title,''), COALESCE(section_subtitle,''),
			COALESCE(deadline::text,''), COALESCE(delivery_date::text,''),
			COALESCE(notes,'')
		FROM order_estimate_settings
		WHERE order_id = $1::uuid
	`, orderID)
	if err == nil {
		defer sRows.Close()
		for sRows.Next() {
			var s EstimateSettings
			sRows.Scan(&s.ID, &s.OrderID, &s.ServiceType,
				&s.SectionTitle, &s.SectionSubtitle,
				&s.Deadline, &s.DeliveryDate, &s.Notes)
			if sec, ok := sectionsMap[s.ServiceType]; ok {
				sc := s
				sec.Settings = &sc
			}
		}
	}

	result := make([]DetailEstimateSection, 0, len(order))
	for _, k := range order {
		result = append(result, *sectionsMap[k])
	}
	return result, nil
}

// SaveSection — сохраняет один раздел сметы
func (r *DetailEstimateRepo) SaveSection(ctx context.Context, orderID, savedBy string, req SaveDetailEstimateRequest) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Удаляем старые строки этого раздела
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM order_detail_estimates
		WHERE order_id = $1::uuid AND service_type = $2
	`, orderID, req.ServiceType); err != nil {
		return fmt.Errorf("delete rows: %w", err)
	}

	// 2. Вставляем новые строки
	var totalM2, totalPrice float64
	for i, row := range req.Rows {
		if row.DetailName == "" {
			continue
		}
		if row.Quantity <= 0 {
			row.Quantity = 1
		}
		m2 := (row.WidthMM / 1000.0) * (row.HeightMM / 1000.0) * float64(row.Quantity)
		tp := m2 * row.UnitPrice
		totalM2 += m2
		totalPrice += tp

		// product_id может быть пустым
		var productID interface{}
		if row.ProductID != "" {
			productID = row.ProductID
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO order_detail_estimates
				(order_id, service_type, row_order, detail_name,
				 width_mm, height_mm, quantity, unit_price, created_by,
				 product_id, product_name)
			VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8,
			        NULLIF($9,'')::uuid,
			        $10::uuid, $11)
		`, orderID, req.ServiceType, i,
			row.DetailName, row.WidthMM, row.HeightMM,
			row.Quantity, row.UnitPrice, savedBy,
			productID, row.ProductName,
		); err != nil {
			return fmt.Errorf("insert row %d: %w", i, err)
		}
	}

	// 3. Upsert настроек раздела
	s := req.Settings
	if _, err = tx.ExecContext(ctx, `
		INSERT INTO order_estimate_settings
			(order_id, service_type, section_title, section_subtitle,
			 deadline, delivery_date, notes)
		VALUES ($1::uuid, $2, $3, $4,
		        NULLIF($5,'')::date,
		        NULLIF($6,'')::date,
		        $7)
		ON CONFLICT (order_id, service_type) DO UPDATE SET
			section_title    = EXCLUDED.section_title,
			section_subtitle = EXCLUDED.section_subtitle,
			deadline         = EXCLUDED.deadline,
			delivery_date    = EXCLUDED.delivery_date,
			notes            = EXCLUDED.notes
	`, orderID, req.ServiceType,
		s.SectionTitle, s.SectionSubtitle,
		s.Deadline, s.DeliveryDate, s.Notes,
	); err != nil {
		return fmt.Errorf("upsert settings: %w", err)
	}

	// 4. Логируем в историю
	label := ServiceTypeLabels[req.ServiceType]
	comment := fmt.Sprintf("📐 Смета %s: деталей %d | %.2f м² | %.0f сом.",
		label, len(req.Rows), totalM2, totalPrice)
	tx.ExecContext(ctx, `
		INSERT INTO order_history (order_id, from_stage, to_stage, changed_by, comment)
		VALUES ($1::uuid, 'estimate', 'estimate', NULLIF($2,'')::uuid, $3)
	`, orderID, savedBy, comment)

	// 5. Обновляем estimated_cost
	tx.ExecContext(ctx, `
		UPDATE orders SET estimated_cost = (
			SELECT COALESCE(SUM(total_price), 0)
			FROM order_detail_estimates
			WHERE order_id = $1::uuid
		)
		WHERE id = $1::uuid
	`, orderID)

	return tx.Commit()
}

// DeleteSection — удаляет раздел сметы
func (r *DetailEstimateRepo) DeleteSection(ctx context.Context, orderID, serviceType string) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM order_detail_estimates
		WHERE order_id = $1::uuid AND service_type = $2
	`, orderID, serviceType)
	return err
}