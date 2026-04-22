// internal/repository/estimate_section_stages.go
package repository

import (
	"context"
	"database/sql"
	"fmt"
)

// Этапы по типу услуги
var SectionStages = map[string][]string{
	"cutting": {
		"material", "sawing", "edging", "drilling", "packing", "shipment",
	},
	"painting": {
		"sanding", "priming", "painting", "delivery",
	},
	"cnc": {
		"calculate", "cnc_work", "delivery",
	},
	"soft": {
		"calculate", "work", "delivery",
	},
}

// Названия этапов
var SectionStageLabels = map[string]string{
	"material":  "Приём материала",
	"sawing":    "Распил",
	"edging":    "Кромкование",
	"drilling":  "Присадка",
	"packing":   "Упаковка",
	"shipment":  "Отгрузка",
	"sanding":   "Шлифовка",
	"priming":   "Грунтовка",
	"painting":  "Покраска",
	"delivery":  "Выдача",
	"calculate": "Расчёт",
	"cnc_work":  "Фрезеровка",
	"work":      "Работа",
}

type EstimateSectionStage struct {
	ID          string  `json:"id"`
	OrderID     string  `json:"order_id"`
	ServiceType string  `json:"service_type"`
	Stage       string  `json:"stage"`
	StageOrder  int     `json:"stage_order"`
	Status      string  `json:"status"`
	AssignedTo  string  `json:"assigned_to"`
	AssigneeName string `json:"assignee_name"`
	Notes       string  `json:"notes"`
	StartedAt   *string `json:"started_at"`
	FinishedAt  *string `json:"finished_at"`
}

type EstimateSectionStagesRepo struct {
	db *sql.DB
}

func NewEstimateSectionStagesRepo(db *sql.DB) *EstimateSectionStagesRepo {
	return &EstimateSectionStagesRepo{db: db}
}

// EnsureStagesForSection — создаёт этапы для секции если их ещё нет
func (r *EstimateSectionStagesRepo) EnsureStagesForSection(ctx context.Context, orderID, serviceType string) error {
	stages, ok := SectionStages[serviceType]
	if !ok {
		return nil // неизвестный тип — ничего не делаем
	}

	for i, stage := range stages {
		_, err := r.db.ExecContext(ctx, `
			INSERT INTO estimate_section_stages
				(order_id, service_type, stage, stage_order, status)
			VALUES ($1::uuid, $2, $3, $4, 'pending')
			ON CONFLICT (order_id, service_type, stage) DO NOTHING
		`, orderID, serviceType, stage, i+1)
		if err != nil {
			return fmt.Errorf("ensure stage %s/%s: %w", serviceType, stage, err)
		}
	}

	// Первый этап ставим in_progress если все pending
	var countInProgress int
	r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM estimate_section_stages
		WHERE order_id = $1 AND service_type = $2 AND status = 'in_progress'
	`, orderID, serviceType).Scan(&countInProgress)

	if countInProgress == 0 {
		r.db.ExecContext(ctx, `
			UPDATE estimate_section_stages
			SET status = 'in_progress'
			WHERE order_id = $1 AND service_type = $2
			  AND stage_order = (
			  	SELECT MIN(stage_order) FROM estimate_section_stages
			  	WHERE order_id = $1 AND service_type = $2 AND status = 'pending'
			  )
		`, orderID, serviceType)
	}

	return nil
}

// GetByOrder — все этапы заказа сгруппированные по service_type
func (r *EstimateSectionStagesRepo) GetByOrder(ctx context.Context, orderID string) ([]EstimateSectionStage, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			s.id, s.order_id, s.service_type, s.stage, s.stage_order, s.status,
			COALESCE(CAST(s.assigned_to AS TEXT), ''),
			COALESCE(u.full_name, ''),
			COALESCE(s.notes, ''),
			CAST(s.started_at AS TEXT),
			CAST(s.finished_at AS TEXT)
		FROM estimate_section_stages s
		LEFT JOIN users u ON u.id = s.assigned_to
		WHERE s.order_id = $1
		ORDER BY s.service_type, s.stage_order
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []EstimateSectionStage
	for rows.Next() {
		var s EstimateSectionStage
		var startedAt, finishedAt sql.NullString
		rows.Scan(
			&s.ID, &s.OrderID, &s.ServiceType, &s.Stage, &s.StageOrder, &s.Status,
			&s.AssignedTo, &s.AssigneeName, &s.Notes,
			&startedAt, &finishedAt,
		)
		if startedAt.Valid  { s.StartedAt  = &startedAt.String  }
		if finishedAt.Valid { s.FinishedAt = &finishedAt.String }
		result = append(result, s)
	}
	return result, nil
}

// UpdateStage — обновить статус этапа
func (r *EstimateSectionStagesRepo) UpdateStage(ctx context.Context, stageID, status, notes string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE estimate_section_stages SET
			status      = COALESCE(NULLIF($1,''), status),
			notes       = COALESCE(NULLIF($2,''), notes),
			started_at  = CASE WHEN $1 = 'in_progress' AND started_at IS NULL THEN NOW() ELSE started_at END,
			finished_at = CASE WHEN $1 = 'done' THEN NOW() ELSE finished_at END
		WHERE id = $3
	`, status, notes, stageID)
	return err
}

// CompleteStage — завершить этап и активировать следующий
func (r *EstimateSectionStagesRepo) CompleteStage(ctx context.Context, stageID string) error {
	// Получить info о текущем этапе
	var orderID, serviceType string
	var stageOrder int
	err := r.db.QueryRowContext(ctx, `
		SELECT order_id, service_type, stage_order
		FROM estimate_section_stages WHERE id = $1
	`, stageID).Scan(&orderID, &serviceType, &stageOrder)
	if err != nil {
		return err
	}

	// Завершить текущий
	r.db.ExecContext(ctx, `
		UPDATE estimate_section_stages
		SET status = 'done', finished_at = NOW()
		WHERE id = $1
	`, stageID)

	// Активировать следующий
	r.db.ExecContext(ctx, `
		UPDATE estimate_section_stages
		SET status = 'in_progress', started_at = NOW()
		WHERE order_id = $1 AND service_type = $2
		  AND stage_order = $3 AND status = 'pending'
	`, orderID, serviceType, stageOrder+1)

	return nil
}

// DeleteBySection — удалить этапы секции (при удалении секции сметы)
func (r *EstimateSectionStagesRepo) DeleteBySection(ctx context.Context, orderID, serviceType string) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM estimate_section_stages
		WHERE order_id = $1 AND service_type = $2
	`, orderID, serviceType)
	return err
}
