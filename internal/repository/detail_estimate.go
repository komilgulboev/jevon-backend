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

var DetailServiceToOrderType = map[string]string{
	"cnc":      "cnc",
	"painting": "painting",
	"soft":     "soft_fabric",
	"cutting":  "cutting",
}

type DetailEstimateRepo struct {
	db         *sql.DB
	stagesRepo *EstimateSectionStagesRepo
}

func NewDetailEstimateRepo(db *sql.DB) *DetailEstimateRepo {
	return &DetailEstimateRepo{
		db:         db,
		stagesRepo: NewEstimateSectionStagesRepo(db),
	}
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
			COALESCE(product_id::text, '')  AS product_id,
			COALESCE(product_name, '')       AS product_name
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

// SaveSection — сохраняет один раздел сметы.
// Возвращает totalPrice раздела для создания дочернего заказа.
func (r *DetailEstimateRepo) SaveSection(ctx context.Context, orderID, savedBy string, req SaveDetailEstimateRequest) (float64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// 1. Удаляем старые строки этого раздела
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM order_detail_estimates
		WHERE order_id = $1::uuid AND service_type = $2
	`, orderID, req.ServiceType); err != nil {
		return 0, fmt.Errorf("delete rows: %w", err)
	}

	// 2. Вставляем новые строки
	var totalM2 float64
	for i, row := range req.Rows {
		if row.DetailName == "" {
			continue
		}
		if row.Quantity <= 0 {
			row.Quantity = 1
		}
		m2 := (row.WidthMM / 1000.0) * (row.HeightMM / 1000.0) * float64(row.Quantity)
		totalM2 += m2

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO order_detail_estimates
				(order_id, service_type, row_order, detail_name,
				 width_mm, height_mm, quantity, unit_price,
				 created_by, product_id, product_name)
			VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8,
			        NULLIF($9,'')::uuid,
			        NULLIF($10::text,'')::uuid, $11)
		`, orderID, req.ServiceType, i,
			row.DetailName, row.WidthMM, row.HeightMM,
			row.Quantity, row.UnitPrice,
			savedBy, row.ProductID, row.ProductName,
		); err != nil {
			return 0, fmt.Errorf("insert row %d: %w", i, err)
		}
	}

// Для строк с нулевыми размерами (cutting из EstimateTable) считаем quantity*unit_price
var totalPrice float64
tx.QueryRowContext(ctx, `
    SELECT COALESCE(SUM(
        CASE
            WHEN width_mm > 0 AND height_mm > 0 THEN total_price
            ELSE quantity * unit_price
        END
    ), 0)
    FROM order_detail_estimates
    WHERE order_id = $1::uuid AND service_type = $2
`, orderID, req.ServiceType).Scan(&totalPrice)

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
		return 0, fmt.Errorf("upsert settings: %w", err)
	}

	// 4. Логируем в историю
	label := ServiceTypeLabels[req.ServiceType]
	comment := fmt.Sprintf("📐 Смета %s: деталей %d | %.2f м² | %.0f сом.",
		label, len(req.Rows), totalM2, totalPrice)
	tx.ExecContext(ctx, `
		INSERT INTO order_history (order_id, from_stage, to_stage, changed_by, comment)
		VALUES ($1::uuid, 'estimate', 'estimate', NULLIF($2,'')::uuid, $3)
	`, orderID, savedBy, comment)

	// 5. Обновляем estimated_cost только для external заказов
tx.ExecContext(ctx, `
    UPDATE orders SET estimated_cost = (
        SELECT COALESCE(SUM(total_price), 0)
        FROM order_detail_estimates
        WHERE order_id = $1::uuid
    )
    WHERE id = $1::uuid
      AND order_type = 'external'
`, orderID)

	// 6. Синхронизируем этапы сметы
	r.stagesRepo.EnsureStagesForSection(ctx, orderID, req.ServiceType)

	// 7. Для external-заказов: авто-синхронизация расхода «Материалы» в workshop_expenses
	var orderType string
	_ = tx.QueryRowContext(ctx,
		`SELECT order_type FROM orders WHERE id = $1::uuid`, orderID,
	).Scan(&orderType)
	if orderType == "external" {
		var matTotal float64
		_ = tx.QueryRowContext(ctx, `
			SELECT COALESCE(SUM(total_price), 0)
			FROM order_estimate_materials
			WHERE order_id = $1
		`, orderID).Scan(&matTotal)
		syncMaterialsExpense(ctx, tx, orderID, matTotal)
	}

	if err := tx.Commit(); err != nil {
    return 0, err
}

// Для external-заказов: добавляем этапы динамически из сметы
r.EnsureExternalStagesFromEstimate(ctx, orderID)

return totalPrice, nil
}

// DeleteSection — удаляет раздел сметы
func (r *DetailEstimateRepo) DeleteSection(ctx context.Context, orderID, serviceType string) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM order_detail_estimates
		WHERE order_id = $1::uuid AND service_type = $2
	`, orderID, serviceType)
	r.stagesRepo.DeleteBySection(ctx, orderID, serviceType)
	return err
}

// CopyToChildOrder — копирует строки сметы из родительского заказа в дочерний
func (r *DetailEstimateRepo) CopyToChildOrder(ctx context.Context, parentOrderID, childOrderID, serviceType string) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM order_detail_estimates
		WHERE order_id = $1::uuid AND service_type = $2
	`, childOrderID, serviceType)
	if err != nil {
		return fmt.Errorf("delete child rows: %w", err)
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO order_detail_estimates
			(order_id, service_type, row_order, detail_name,
			 width_mm, height_mm, quantity, unit_price,
			 product_id, product_name)
		SELECT
			$2::uuid, service_type, row_order, detail_name,
			width_mm, height_mm, quantity, unit_price,
			product_id, product_name
		FROM order_detail_estimates
		WHERE order_id = $1::uuid AND service_type = $3
	`, parentOrderID, childOrderID, serviceType)
	if err != nil {
		return fmt.Errorf("copy rows: %w", err)
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO order_estimate_settings
			(order_id, service_type, section_title, section_subtitle, deadline, delivery_date, notes)
		SELECT
			$2::uuid, service_type, section_title, section_subtitle, deadline, delivery_date, notes
		FROM order_estimate_settings
		WHERE order_id = $1::uuid AND service_type = $3
		ON CONFLICT (order_id, service_type) DO UPDATE SET
			section_title    = EXCLUDED.section_title,
			section_subtitle = EXCLUDED.section_subtitle,
			deadline         = EXCLUDED.deadline,
			delivery_date    = EXCLUDED.delivery_date,
			notes            = EXCLUDED.notes
	`, parentOrderID, childOrderID, serviceType)
	if err != nil {
		return fmt.Errorf("copy settings: %w", err)
	}

	return nil
}
// EnsureExternalStagesFromEstimate — добавляет этапы в external-заказ
// динамически на основе сохранённых разделов сметы.
// Вызывается после каждого SaveSection.
//
// Порядок этапов external:
//   1-intake  2-design  3-production
//   4-sawing  5-edging  6-drilling  7-painting
//   8-packing  9-handover
func (r *DetailEstimateRepo) EnsureExternalStagesFromEstimate(ctx context.Context, orderID string) error {
	// Проверяем тип заказа
	var orderType string
	err := r.db.QueryRowContext(ctx,
		`SELECT order_type FROM orders WHERE id = $1::uuid`, orderID,
	).Scan(&orderType)
	if err != nil || orderType != "external" {
		return nil
	}

	// Какие service_type есть в смете этого заказа
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT service_type
		FROM order_detail_estimates
		WHERE order_id = $1::uuid
	`, orderID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	hasService := map[string]bool{}
	for rows.Next() {
		var st string
		if rows.Scan(&st) == nil {
			hasService[st] = true
		}
	}

	// Фиксированный порядок всех возможных этапов external
	type stageEntry struct {
		stage string
		order int
	}
	allStages := []stageEntry{
		{"intake",      1},
		{"design",      2},
		{"production",  3},
		{"sawing",      4}, // только если есть cutting в смете
		{"edging",      5}, // только если есть cutting в смете
		{"drilling",    6}, // только если есть cutting в смете
		{"painting",    7}, // только если есть painting в смете
		{"packing",     8},
		{"handover",    9},
	}

	// Какие этапы добавлять динамически
	dynamicMap := map[string]string{
		"sawing":   "cutting",  // этап sawing появляется если есть секция cutting
		"edging":   "cutting",
		"drilling": "cutting",
		"painting": "painting", // этап painting появляется если есть секция painting
	}

	for _, e := range allStages {
		// Для динамических этапов проверяем наличие секции в смете
		if requiredService, isDynamic := dynamicMap[e.stage]; isDynamic {
			if !hasService[requiredService] {
				continue
			}
		}

		// INSERT с правильным stage_order, обновляем порядок если этап уже есть
		r.db.ExecContext(ctx, `
			INSERT INTO order_stages (order_id, stage, stage_order, status)
			VALUES ($1::uuid, $2, $3, 'pending')
			ON CONFLICT (order_id, stage) DO UPDATE SET
				stage_order = EXCLUDED.stage_order
		`, orderID, e.stage, e.order)
	}

	// Если нет ни одного in_progress — активируем первый pending
	var countInProgress int
	r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM order_stages
		WHERE order_id = $1::uuid AND status = 'in_progress'
	`, orderID).Scan(&countInProgress)

	if countInProgress == 0 {
		r.db.ExecContext(ctx, `
			UPDATE order_stages SET status = 'in_progress'
			WHERE order_id = $1::uuid
			  AND stage_order = (
			  	SELECT MIN(stage_order) FROM order_stages
			  	WHERE order_id = $1::uuid AND status = 'pending'
			  )
		`, orderID)
	}

	return nil
}