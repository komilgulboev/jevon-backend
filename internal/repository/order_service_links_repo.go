package repository

import (
	"context"
	"database/sql"
	"fmt"

	"jevon/internal/models"
)

// ServiceLinksByParent возвращает все дочерние заказы-услуги для родительского заказа
func (r *OrderRepo) ServiceLinksByParent(ctx context.Context, parentOrderID string) ([]models.OrderServiceLink, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			l.id::text,
			l.parent_order_id::text,
			l.child_order_id::text,
			l.service_type,
			l.amount,
			l.created_at,
			o.order_number,
			o.order_type,
			o.status,
			o.title,
			COALESCE(o.current_stage, '')
		FROM order_service_links l
		JOIN orders o ON o.id = l.child_order_id
		WHERE l.parent_order_id = $1
		ORDER BY l.created_at
	`, parentOrderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.OrderServiceLink
	for rows.Next() {
		var l models.OrderServiceLink
		if err := rows.Scan(
			&l.ID, &l.ParentOrderID, &l.ChildOrderID,
			&l.ServiceType, &l.Amount, &l.CreatedAt,
			&l.ChildOrderNumber, &l.ChildOrderType,
			&l.ChildStatus, &l.ChildTitle,
			&l.ChildCurrentStage,
		); err != nil {
			return nil, err
		}
		result = append(result, l)
	}
	return result, nil
}

// UpsertServiceLink создаёт дочерний заказ если нет, или обновляет сумму если есть.
// Этапы создаются в правильном порядке через StagesByType (слайс, не map).
func (r *OrderRepo) UpsertServiceLink(
	ctx context.Context,
	parentOrderID string,
	parentOrder *models.Order,
	serviceType string,
	orderType string,
	amount float64,
	createdBy string,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Проверяем существующий link
	var childOrderID string
	err = tx.QueryRowContext(ctx, `
		SELECT child_order_id::text
		FROM order_service_links
		WHERE parent_order_id = $1 AND service_type = $2
	`, parentOrderID, serviceType).Scan(&childOrderID)

	if err == sql.ErrNoRows {
		// Создаём дочерний заказ
		typeLabel := models.OrderTypeLabels[orderType]
		if typeLabel == "" {
			typeLabel = orderType
		}
		title := fmt.Sprintf("%s (из заказа №%d)", typeLabel, parentOrder.OrderNumber)

		var clientID interface{}
		if parentOrder.ClientID != "" {
			clientID = parentOrder.ClientID
		}

		err = tx.QueryRowContext(ctx, `
			INSERT INTO orders (
				order_type, title, client_id, client_name, client_phone,
				estimated_cost, status, priority, created_by, parent_order_id
			) VALUES ($1, $2, $3, $4, $5, $6, 'new', 'medium', NULLIF($7,'')::uuid, $8::uuid)
			RETURNING id::text
		`,
			orderType, title,
			clientID, parentOrder.ClientName, parentOrder.ClientPhone,
			amount, createdBy, parentOrderID,
		).Scan(&childOrderID)
		if err != nil {
			return fmt.Errorf("create child order: %w", err)
		}

		// Создаём этапы в правильном порядке через StagesByType (слайс).
		// Раньше использовался StageLabelsByType (map) — порядок был случайным.
		stages := models.StagesByType[orderType]
		for i, stage := range stages {
			tx.ExecContext(ctx, `
				INSERT INTO order_stages (order_id, stage, stage_order, status)
				VALUES ($1::uuid, $2, $3, 'pending')
				ON CONFLICT DO NOTHING
			`, childOrderID, stage, i+1)
		}

		// Первый этап сразу in_progress
		if len(stages) > 0 {
			tx.ExecContext(ctx, `
				UPDATE order_stages SET status = 'in_progress'
				WHERE order_id = $1::uuid AND stage = $2
			`, childOrderID, stages[0])
		}

		// Создаём link
		_, err = tx.ExecContext(ctx, `
			INSERT INTO order_service_links (parent_order_id, child_order_id, service_type, amount)
			VALUES ($1::uuid, $2::uuid, $3, $4)
		`, parentOrderID, childOrderID, serviceType, amount)
		if err != nil {
			return fmt.Errorf("insert link: %w", err)
		}

	} else if err == nil {
		// Обновляем amount в дочернем заказе и link
		_, err = tx.ExecContext(ctx,
			`UPDATE orders SET estimated_cost = $1 WHERE id = $2::uuid`,
			amount, childOrderID)
		if err != nil {
			return fmt.Errorf("update child order: %w", err)
		}
		_, err = tx.ExecContext(ctx, `
			UPDATE order_service_links SET amount = $1
			WHERE parent_order_id = $2::uuid AND service_type = $3
		`, amount, parentOrderID, serviceType)
		if err != nil {
			return fmt.Errorf("update link: %w", err)
		}
	} else {
		return err
	}

	return tx.Commit()
}

// EnsureExternalStagesFromEstimate — добавляет этапы sawing/edging/drilling
// в external-заказ динамически, если соответствующие группы услуг есть в смете.
// Вызывается после сохранения сметы (POST /detail-estimate).
//
// Порядок этапов для external фиксирован через ExternalDynamicStageOrder:
//   1-intake  2-design  3-production  4-sawing  5-edging  6-drilling  7-packing  8-handover
//
// Этапы packing и handover пересоздаются с правильным номером если
// динамические этапы сдвигают их вперёд.
func (r *OrderRepo) EnsureExternalStagesFromEstimate(ctx context.Context, orderID string) error {
	// Проверяем тип заказа
	var orderType string
	err := r.db.QueryRowContext(ctx,
		`SELECT order_type FROM orders WHERE id = $1::uuid`, orderID,
	).Scan(&orderType)
	if err != nil || orderType != "external" {
		return nil
	}

	// Определяем какие группы услуг есть в смете
	// Смотрим в order_estimate_services → estimate_catalog
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT c.group_name
		FROM order_estimate_services s
		JOIN estimate_catalog c ON c.id = s.catalog_id
		WHERE s.order_id = $1::uuid
		  AND c.group_name IN ('sawing', 'edging', 'drilling')
	`, orderID)
	if err != nil {
		// Таблица может называться иначе — пробуем запасной вариант
		// через order_detail_estimates (detail_name содержит название услуги)
		rows, err = r.db.QueryContext(ctx, `
			SELECT DISTINCT
				CASE
					WHEN detail_name ILIKE '%распил%'    THEN 'sawing'
					WHEN detail_name ILIKE '%кромк%'     THEN 'edging'
					WHEN detail_name ILIKE '%присадк%'   THEN 'drilling'
				END AS group_name
			FROM order_detail_estimates
			WHERE order_id = $1::uuid
			  AND service_type = 'cutting'
			  AND detail_name !~ '^\s*$'
		`, orderID)
		if err != nil {
			return nil
		}
	}
	defer rows.Close()

	hasGroup := map[string]bool{}
	for rows.Next() {
		var g string
		if rows.Scan(&g) == nil && g != "" {
			hasGroup[g] = true
		}
	}

	// Всегда обеспечиваем базовые этапы с правильными номерами
	// (используем ExternalDynamicStageOrder для всех этапов)
	allStages := []string{
		"intake", "design", "production",
		"sawing", "edging", "drilling",
		"packing", "handover",
	}

	for _, stage := range allStages {
		// Динамические этапы — только если есть в смете
		if stage == "sawing" || stage == "edging" || stage == "drilling" {
			if !hasGroup[stage] {
				continue
			}
		}

		stageOrder := models.ExternalDynamicStageOrder[stage]

		// INSERT OR IGNORE — не трогаем уже существующие этапы со статусом
		r.db.ExecContext(ctx, `
			INSERT INTO order_stages (order_id, stage, stage_order, status)
			VALUES ($1::uuid, $2, $3, 'pending')
			ON CONFLICT (order_id, stage) DO UPDATE SET
				stage_order = EXCLUDED.stage_order
		`, orderID, stage, stageOrder)
	}

	// Если нет ни одного этапа in_progress — активируем первый pending
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