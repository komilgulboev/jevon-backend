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

// UpsertServiceLink создаёт дочерний заказ если нет, или обновляет сумму если есть
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

		// Создаём этапы для дочернего заказа
		stages := models.StageLabelsByType[orderType]
		stageOrder := 0
		for stage := range stages {
			tx.ExecContext(ctx, `
				INSERT INTO order_stages (order_id, stage, stage_order, status)
				VALUES ($1::uuid, $2, $3, 'pending')
				ON CONFLICT DO NOTHING
			`, childOrderID, stage, stageOrder)
			stageOrder++
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