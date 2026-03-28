package repository

import (
	"context"
	"database/sql"
	"fmt"
)

// ── Модели ────────────────────────────────────────────────

type ServiceType = string

const (
	ServiceCNC      ServiceType = "cnc"
	ServicePainting ServiceType = "painting"
	ServiceSoft     ServiceType = "soft"
	ServiceCutting  ServiceType = "cutting"
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
	ServiceType string                  `json:"service_type" binding:"required"`
	Settings    EstimateSettings        `json:"settings"`
	Rows        []DetailEstimateRowInput `json:"rows"`
}

type DetailEstimateRowInput struct {
	DetailName string  `json:"detail_name"`
	WidthMM    float64 `json:"width_mm"`
	HeightMM   float64 `json:"height_mm"`
	Quantity   int     `json:"quantity"`
	UnitPrice  float64 `json:"unit_price"`
}

type DetailEstimateSection struct {
	ServiceType string             `json:"service_type"`
	Label       string             `json:"label"`
	Settings    *EstimateSettings  `json:"settings"`
	Rows        []DetailEstimateRow `json:"rows"`
	TotalAreaM2 float64            `json:"total_area_m2"`
	TotalPrice  float64            `json:"total_price"`
}

// ── Репозиторий ───────────────────────────────────────────

type DetailEstimateRepo struct {
	db *sql.DB
}

func NewDetailEstimateRepo(db *sql.DB) *DetailEstimateRepo {
	return &DetailEstimateRepo{db: db}
}

// GetByOrder — возвращает все разделы сметы для заказа
func (r *DetailEstimateRepo) GetByOrder(ctx context.Context, orderID string) ([]DetailEstimateSection, error) {
	// Загружаем все строки
	rows, err := r.db.QueryContext(ctx, `
		SELECT id::text, order_id::text, service_type, row_order,
		       detail_name,
		       COALESCE(width_mm,0), COALESCE(height_mm,0),
		       COALESCE(quantity,1),
		       COALESCE(area_m2,0), COALESCE(unit_price,0), COALESCE(total_price,0)
		FROM order_detail_estimates
		WHERE order_id = $1
		ORDER BY service_type, row_order
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sectionsMap := make(map[string]*DetailEstimateSection)
	for rows.Next() {
		var row DetailEstimateRow
		rows.Scan(
			&row.ID, &row.OrderID, &row.ServiceType, &row.RowOrder,
			&row.DetailName, &row.WidthMM, &row.HeightMM,
			&row.Quantity, &row.AreaM2, &row.UnitPrice, &row.TotalPrice,
		)
		if _, ok := sectionsMap[row.ServiceType]; !ok {
			sectionsMap[row.ServiceType] = &DetailEstimateSection{
				ServiceType: row.ServiceType,
				Label:       ServiceTypeLabels[row.ServiceType],
			}
		}
		sec := sectionsMap[row.ServiceType]
		sec.Rows = append(sec.Rows, row)
		sec.TotalAreaM2 += row.AreaM2
		sec.TotalPrice  += row.TotalPrice
	}

	// Загружаем настройки
	settingsRows, err := r.db.QueryContext(ctx, `
		SELECT id::text, order_id::text, service_type,
		       COALESCE(section_title,''), COALESCE(section_subtitle,''),
		       COALESCE(deadline,''), CAST(delivery_date AS TEXT),
		       COALESCE(notes,'')
		FROM order_estimate_settings
		WHERE order_id = $1
	`, orderID)
	if err == nil {
		defer settingsRows.Close()
		for settingsRows.Next() {
			var s EstimateSettings
			var deliveryDate sql.NullString
			settingsRows.Scan(
				&s.ID, &s.OrderID, &s.ServiceType,
				&s.SectionTitle, &s.SectionSubtitle,
				&s.Deadline, &deliveryDate, &s.Notes,
			)
			if deliveryDate.Valid {
				s.DeliveryDate = deliveryDate.String
			}
			if sec, ok := sectionsMap[s.ServiceType]; ok {
				sec.Settings = &s
			}
		}
	}

	// Собираем в срез по порядку
	order := []string{"cnc", "painting", "soft", "cutting"}
	var result []DetailEstimateSection
	for _, stype := range order {
		if sec, ok := sectionsMap[stype]; ok {
			result = append(result, *sec)
		}
	}
	return result, nil
}

// SaveSection — сохраняет один раздел сметы
func (r *DetailEstimateRepo) SaveSection(ctx context.Context, orderID, savedBy string, req SaveDetailEstimateRequest) error {

	// 1. Удаляем старые строки
	if _, err := r.db.ExecContext(ctx, `
		DELETE FROM order_detail_estimates
		WHERE order_id = $1::uuid AND service_type = $2
	`, orderID, req.ServiceType); err != nil {
		return fmt.Errorf("delete rows: %w", err)
	}

	// 2. Вставляем новые строки
	for i, row := range req.Rows {
		if row.DetailName == "" {
			continue
		}
		if row.Quantity <= 0 {
			row.Quantity = 1
		}
		if _, err := r.db.ExecContext(ctx, `
			INSERT INTO order_detail_estimates
				(order_id, service_type, row_order, detail_name,
				 width_mm, height_mm, quantity, unit_price, created_by)
			VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8, NULLIF($9,'')::uuid)
		`, orderID, req.ServiceType, i,
			row.DetailName, row.WidthMM, row.HeightMM,
			row.Quantity, row.UnitPrice, savedBy,
		); err != nil {
			return fmt.Errorf("insert row %d: %w", i, err)
		}
	}

	// 3. Сохраняем настройки раздела
	s := req.Settings
	if _, err := r.db.ExecContext(ctx, `
		INSERT INTO order_estimate_settings
			(order_id, service_type, section_title, section_subtitle, deadline, delivery_date, notes)
		VALUES (
			$1::uuid, $2, $3, $4, $5,
			CASE WHEN $6 = '' THEN NULL ELSE $6::date END,
			$7
		)
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

	// 4. Считаем итог для логирования
	var totalM2, totalPrice float64
	for _, row := range req.Rows {
		if row.DetailName == "" {
			continue
		}
		m2 := (row.WidthMM / 1000.0) * (row.HeightMM / 1000.0) * float64(row.Quantity)
		totalM2    += m2
		totalPrice += m2 * row.UnitPrice
	}

	// 5. Логируем в историю
	label := ServiceTypeLabels[req.ServiceType]
	comment := fmt.Sprintf("📐 Смета %s: деталей %d | %.2f м² | %.0f сом.",
		label, len(req.Rows), totalM2, totalPrice)
	r.db.ExecContext(ctx, `
		INSERT INTO order_history (order_id, from_stage, to_stage, changed_by, comment)
		VALUES ($1::uuid, 'estimate', 'estimate', NULLIF($2,'')::uuid, $3)
	`, orderID, savedBy, comment)

	// 6. Обновляем estimated_cost
	if totalPrice > 0 {
		r.db.ExecContext(ctx, `
			UPDATE orders SET estimated_cost = (
				SELECT COALESCE(SUM(total_price), 0)
				FROM order_detail_estimates
				WHERE order_id = $1::uuid
			)
			WHERE id = $1::uuid
		`, orderID)
	}

	return nil
}

// DeleteSection — удаляет раздел сметы
func (r *DetailEstimateRepo) DeleteSection(ctx context.Context, orderID, serviceType string) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM order_detail_estimates
		WHERE order_id = $1 AND service_type = $2
	`, orderID, serviceType)
	return err
}