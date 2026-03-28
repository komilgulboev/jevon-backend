package repository

import (
	"context"
	"database/sql"
	"fmt"
)

// ── Модели ────────────────────────────────────────────────

type ServiceCatalogItem struct {
	ID        int     `json:"id"`
	GroupName string  `json:"group_name"`
	Name      string  `json:"name"`
	Unit      string  `json:"unit"`
	UnitSpec  string  `json:"unit_spec"`
	Price     float64 `json:"price"`
	IsActive  bool    `json:"is_active"`
	SortOrder int     `json:"sort_order"`
}

type ColorItem struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
}

type EstimateServiceRow struct {
	ID         string  `json:"id"`
	OrderID    string  `json:"order_id"`
	CatalogID  *int    `json:"catalog_id"`
	Name       string  `json:"name"`
	Color      string  `json:"color"`
	Article    string  `json:"article"`
	Quantity   float64 `json:"quantity"`
	Unit       string  `json:"unit"`
	UnitSpec   string  `json:"unit_spec"`
	UnitPrice  float64 `json:"unit_price"`
	TotalPrice float64 `json:"total_price"`
	SortOrder  int     `json:"sort_order"`
}

type EstimateMaterialRow struct {
	ID         string  `json:"id"`
	OrderID    string  `json:"order_id"`
	Name       string  `json:"name"`
	Quantity   float64 `json:"quantity"`
	Unit       string  `json:"unit"`
	UnitPrice  float64 `json:"unit_price"`
	TotalPrice float64 `json:"total_price"`
	SortOrder  int     `json:"sort_order"`
}

type UpsertEstimateServiceRequest struct {
	ID        string  `json:"id"`
	CatalogID *int    `json:"catalog_id"`
	Name      string  `json:"name"     binding:"required"`
	Color     string  `json:"color"`
	Article   string  `json:"article"`
	Quantity  float64 `json:"quantity"`
	Unit      string  `json:"unit"`
	UnitSpec  string  `json:"unit_spec"`
	UnitPrice float64 `json:"unit_price"`
	SortOrder int     `json:"sort_order"`
}

type UpsertEstimateMaterialRequest struct {
	ID        string  `json:"id"`
	Name      string  `json:"name" binding:"required"`
	Quantity  float64 `json:"quantity"`
	Unit      string  `json:"unit"`
	UnitPrice float64 `json:"unit_price"`
	SortOrder int     `json:"sort_order"`
}

type SaveEstimateRequest struct {
	Services  []UpsertEstimateServiceRequest  `json:"services"`
	Materials []UpsertEstimateMaterialRequest `json:"materials"`
	Notes     string                          `json:"notes"`
}

// ── Репозиторий ───────────────────────────────────────────

type EstimateRepo struct {
	db *sql.DB
}

func NewEstimateRepo(db *sql.DB) *EstimateRepo {
	return &EstimateRepo{db: db}
}

// ── Каталог услуг ─────────────────────────────────────────

var GroupLabels = map[string]string{
	"design":   "Чертёж",
	"sawing":   "Распил",
	"edging":   "Кромкование",
	"drilling": "Присадка",
	"milling":  "Фрезеровка",
	"gluing":   "Склейка",
	"packing":  "Упаковка",
	"other":    "Другое",
}

func (r *EstimateRepo) CatalogList(ctx context.Context) (map[string][]ServiceCatalogItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, group_name, name, unit, COALESCE(unit_spec,''), price, is_active, sort_order
		FROM cutting_service_catalog
		WHERE is_active = true
		ORDER BY group_name, sort_order, name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]ServiceCatalogItem)
	for rows.Next() {
		var item ServiceCatalogItem
		rows.Scan(&item.ID, &item.GroupName, &item.Name, &item.Unit,
			&item.UnitSpec, &item.Price, &item.IsActive, &item.SortOrder)
		result[item.GroupName] = append(result[item.GroupName], item)
	}
	return result, nil
}

func (r *EstimateRepo) CatalogFlat(ctx context.Context) ([]ServiceCatalogItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, group_name, name, unit, COALESCE(unit_spec,''), price, is_active, sort_order
		FROM cutting_service_catalog
		WHERE is_active = true
		ORDER BY group_name, sort_order, name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ServiceCatalogItem
	for rows.Next() {
		var item ServiceCatalogItem
		rows.Scan(&item.ID, &item.GroupName, &item.Name, &item.Unit,
			&item.UnitSpec, &item.Price, &item.IsActive, &item.SortOrder)
		result = append(result, item)
	}
	return result, nil
}

func (r *EstimateRepo) CatalogCreate(ctx context.Context, req ServiceCatalogItem) (int, error) {
	var id int
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO cutting_service_catalog (group_name, name, unit, unit_spec, price, sort_order)
		VALUES ($1,$2,$3,NULLIF($4,''),$5,$6)
		RETURNING id
	`, req.GroupName, req.Name, req.Unit, req.UnitSpec, req.Price, req.SortOrder,
	).Scan(&id)
	return id, err
}

func (r *EstimateRepo) CatalogUpdate(ctx context.Context, id int, req ServiceCatalogItem) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE cutting_service_catalog SET
			group_name = $1, name = $2, unit = $3,
			unit_spec  = NULLIF($4,''), price = $5,
			sort_order = $6, is_active = $7
		WHERE id = $8
	`, req.GroupName, req.Name, req.Unit, req.UnitSpec,
		req.Price, req.SortOrder, req.IsActive, id)
	return err
}

func (r *EstimateRepo) CatalogDelete(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE cutting_service_catalog SET is_active = false WHERE id = $1`, id)
	return err
}

// ── Каталог цветов ────────────────────────────────────────

func (r *EstimateRepo) ColorList(ctx context.Context) ([]ColorItem, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, sort_order FROM color_catalog ORDER BY sort_order, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ColorItem
	for rows.Next() {
		var c ColorItem
		rows.Scan(&c.ID, &c.Name, &c.SortOrder)
		result = append(result, c)
	}
	return result, nil
}

// ── Смета заказа ─────────────────────────────────────────

func (r *EstimateRepo) EstimateByOrder(ctx context.Context, orderID string) ([]EstimateServiceRow, []EstimateMaterialRow, float64, float64, error) {
	// Услуги
	sRows, err := r.db.QueryContext(ctx, `
		SELECT id::text, order_id::text, catalog_id,
		       name, COALESCE(color,''), COALESCE(article,''),
		       quantity, unit, COALESCE(unit_spec,''),
		       unit_price, total_price, sort_order
		FROM order_estimate_services
		WHERE order_id = $1
		ORDER BY sort_order, created_at
	`, orderID)
	if err != nil {
		return nil, nil, 0, 0, err
	}
	defer sRows.Close()

	var services []EstimateServiceRow
	var totalServices float64
	for sRows.Next() {
		var s EstimateServiceRow
		sRows.Scan(
			&s.ID, &s.OrderID, &s.CatalogID,
			&s.Name, &s.Color, &s.Article,
			&s.Quantity, &s.Unit, &s.UnitSpec,
			&s.UnitPrice, &s.TotalPrice, &s.SortOrder,
		)
		totalServices += s.TotalPrice
		services = append(services, s)
	}

	// Материалы
	mRows, err := r.db.QueryContext(ctx, `
		SELECT id::text, order_id::text,
		       name, quantity, unit, unit_price, total_price, sort_order
		FROM order_estimate_materials
		WHERE order_id = $1
		ORDER BY sort_order, created_at
	`, orderID)
	if err != nil {
		return nil, nil, 0, 0, err
	}
	defer mRows.Close()

	var materials []EstimateMaterialRow
	var totalMaterials float64
	for mRows.Next() {
		var m EstimateMaterialRow
		mRows.Scan(
			&m.ID, &m.OrderID,
			&m.Name, &m.Quantity, &m.Unit,
			&m.UnitPrice, &m.TotalPrice, &m.SortOrder,
		)
		totalMaterials += m.TotalPrice
		materials = append(materials, m)
	}

	return services, materials, totalServices, totalMaterials, nil
}

// SaveEstimate — полная перезапись сметы (удаляем старое, вставляем новое)
func (r *EstimateRepo) SaveEstimate(ctx context.Context, orderID, savedBy string, req SaveEstimateRequest) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Удаляем старые строки
	tx.ExecContext(ctx, `DELETE FROM order_estimate_services  WHERE order_id = $1`, orderID)
	tx.ExecContext(ctx, `DELETE FROM order_estimate_materials WHERE order_id = $1`, orderID)

	// Вставляем услуги
	for i, s := range req.Services {
		if s.Name == "" { continue }
		if s.Unit == "" { s.Unit = "шт" }
		_, err := tx.ExecContext(ctx, `
			INSERT INTO order_estimate_services
				(order_id, catalog_id, name, color, article,
				 quantity, unit, unit_spec, unit_price, sort_order, created_by)
			VALUES
				($1, $2, $3, NULLIF($4,''), NULLIF($5,''),
				 $6, $7, NULLIF($8,''), $9, $10, NULLIF($11,'')::uuid)
		`, orderID,
			s.CatalogID, s.Name, s.Color, s.Article,
			s.Quantity, s.Unit, s.UnitSpec, s.UnitPrice, i, savedBy,
		)
		if err != nil {
			return fmt.Errorf("insert service: %w", err)
		}
	}

	// Вставляем материалы
	for i, m := range req.Materials {
		if m.Name == "" { continue }
		if m.Unit == "" { m.Unit = "шт" }
		_, err := tx.ExecContext(ctx, `
			INSERT INTO order_estimate_materials
				(order_id, name, quantity, unit, unit_price, sort_order, created_by)
			VALUES ($1,$2,$3,$4,$5,$6,NULLIF($7,'')::uuid)
		`, orderID, m.Name, m.Quantity, m.Unit, m.UnitPrice, i, savedBy,
		)
		if err != nil {
			return fmt.Errorf("insert material: %w", err)
		}
	}

	// Считаем итог и обновляем estimated_cost заказа
	var totalServices, totalMaterials float64
	for _, s := range req.Services {
		totalServices += s.Quantity * s.UnitPrice
	}
	for _, m := range req.Materials {
		totalMaterials += m.Quantity * m.UnitPrice
	}
	total := totalServices + totalMaterials
	if total > 0 {
		tx.ExecContext(ctx,
			`UPDATE orders SET estimated_cost = $1 WHERE id = $2`,
			total, orderID)
	}

	// Логируем в историю
	comment := fmt.Sprintf("📋 Смета обновлена: услуг %d | материалов %d | итого %.0f сом.",
		len(req.Services), len(req.Materials), total)
	tx.ExecContext(ctx, `
		INSERT INTO order_history (order_id, from_stage, to_stage, changed_by, comment)
		VALUES ($1, 'estimate', 'estimate', NULLIF($2,'')::uuid, $3)
	`, orderID, savedBy, comment)

	return tx.Commit()
}
