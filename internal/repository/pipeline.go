package repository

import (
	"context"
	"database/sql"

	"jevon/internal/models"
)

type PipelineRepo struct {
	db *sql.DB
}

func NewPipelineRepo(db *sql.DB) *PipelineRepo {
	return &PipelineRepo{db: db}
}

// ── Operation Catalog ─────────────────────────────────────

func (r *PipelineRepo) CatalogList(ctx context.Context, category string) ([]models.OperationCatalog, error) {
	query := `SELECT id, name, COALESCE(description,''), category, is_active
	          FROM operation_catalog WHERE is_active = true`
	args := []interface{}{}

	if category != "" {
		query += " AND category = $1"
		args = append(args, category)
	}
	query += " ORDER BY category, name"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.OperationCatalog
	for rows.Next() {
		var o models.OperationCatalog
		rows.Scan(&o.ID, &o.Name, &o.Description, &o.Category, &o.IsActive)
		result = append(result, o)
	}
	return result, nil
}

// ── Project Stages ────────────────────────────────────────

func (r *PipelineRepo) StagesByProject(ctx context.Context, projectID string) ([]models.ProjectStage, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			ps.id, ps.project_id, ps.stage, ps.status,
			COALESCE(CAST(ps.assigned_to AS TEXT),''),
			COALESCE(u.full_name,''),
			ps.started_at, ps.finished_at,
			COALESCE(ps.notes,''), ps.created_at, ps.updated_at
		FROM project_stages ps
		LEFT JOIN users u ON u.id = ps.assigned_to
		WHERE ps.project_id = $1
		ORDER BY array_position(
			ARRAY['intake','design','cutting','production','warehouse','delivery','assembly','handover'],
			ps.stage
		)
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.ProjectStage
	for rows.Next() {
		var s models.ProjectStage
		rows.Scan(
			&s.ID, &s.ProjectID, &s.Stage, &s.Status,
			&s.AssignedTo, &s.AssigneeName,
			&s.StartedAt, &s.FinishedAt,
			&s.Notes, &s.CreatedAt, &s.UpdatedAt,
		)
		result = append(result, s)
	}
	return result, nil
}

func (r *PipelineRepo) StageByID(ctx context.Context, stageID string) (*models.ProjectStage, error) {
	var s models.ProjectStage
	err := r.db.QueryRowContext(ctx, `
		SELECT
			ps.id, ps.project_id, ps.stage, ps.status,
			COALESCE(CAST(ps.assigned_to AS TEXT),''),
			COALESCE(u.full_name,''),
			ps.started_at, ps.finished_at,
			COALESCE(ps.notes,''), ps.created_at, ps.updated_at
		FROM project_stages ps
		LEFT JOIN users u ON u.id = ps.assigned_to
		WHERE ps.id = $1
	`, stageID).Scan(
		&s.ID, &s.ProjectID, &s.Stage, &s.Status,
		&s.AssignedTo, &s.AssigneeName,
		&s.StartedAt, &s.FinishedAt,
		&s.Notes, &s.CreatedAt, &s.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &s, err
}

func (r *PipelineRepo) UpdateStage(ctx context.Context, stageID string, req models.UpdateStageRequest) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE project_stages SET
			status      = COALESCE($1, status),
			assigned_to = COALESCE(NULLIF($2,'')::uuid, assigned_to),
			notes       = COALESCE($3, notes),
			started_at  = CASE
				WHEN $1 = 'in_progress' AND started_at IS NULL THEN NOW()
				ELSE started_at
			END
		WHERE id = $4
	`, req.Status, req.AssignedTo, req.Notes, stageID)
	return err
}

func (r *PipelineRepo) CompleteStage(ctx context.Context, stageID string, req models.CompleteStageRequest) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE project_stages SET
			status    = 'done',
			notes     = COALESCE(NULLIF($1,''), notes)
		WHERE id = $2
	`, req.Notes, stageID)
	return err
}

// ── Stage Operations ──────────────────────────────────────

func (r *PipelineRepo) OperationsByStage(ctx context.Context, stageID string) ([]models.StageOperation, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			so.id, so.project_id, so.stage_id,
			so.catalog_id,
			COALESCE(oc.name,''),
			COALESCE(so.custom_name,''),
			COALESCE(CAST(so.assigned_to AS TEXT),''),
			COALESCE(u.full_name,''),
			so.status,
			so.started_at, so.finished_at,
			COALESCE(so.notes,''), so.created_at
		FROM stage_operations so
		LEFT JOIN operation_catalog oc ON oc.id = so.catalog_id
		LEFT JOIN users u ON u.id = so.assigned_to
		WHERE so.stage_id = $1
		ORDER BY so.created_at ASC
	`, stageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.StageOperation
	for rows.Next() {
		var o models.StageOperation
		rows.Scan(
			&o.ID, &o.ProjectID, &o.StageID,
			&o.CatalogID, &o.CatalogName, &o.CustomName,
			&o.AssignedTo, &o.AssigneeName,
			&o.Status, &o.StartedAt, &o.FinishedAt,
			&o.Notes, &o.CreatedAt,
		)
		result = append(result, o)
	}
	return result, nil
}

func (r *PipelineRepo) OperationsByProject(ctx context.Context, projectID string) ([]models.StageOperation, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			so.id, so.project_id, so.stage_id,
			so.catalog_id,
			COALESCE(oc.name,''),
			COALESCE(so.custom_name,''),
			COALESCE(CAST(so.assigned_to AS TEXT),''),
			COALESCE(u.full_name,''),
			so.status,
			so.started_at, so.finished_at,
			COALESCE(so.notes,''), so.created_at
		FROM stage_operations so
		LEFT JOIN operation_catalog oc ON oc.id = so.catalog_id
		LEFT JOIN users u ON u.id = so.assigned_to
		WHERE so.project_id = $1
		ORDER BY so.created_at ASC
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.StageOperation
	for rows.Next() {
		var o models.StageOperation
		rows.Scan(
			&o.ID, &o.ProjectID, &o.StageID,
			&o.CatalogID, &o.CatalogName, &o.CustomName,
			&o.AssignedTo, &o.AssigneeName,
			&o.Status, &o.StartedAt, &o.FinishedAt,
			&o.Notes, &o.CreatedAt,
		)
		result = append(result, o)
	}
	return result, nil
}

func (r *PipelineRepo) CreateOperation(ctx context.Context, projectID string, req models.CreateOperationRequest, createdBy string) (string, error) {
	var id string
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO stage_operations
			(project_id, stage_id, catalog_id, custom_name, assigned_to, notes)
		VALUES ($1, $2, $3, NULLIF($4,''), NULLIF($5,'')::uuid, $6)
		RETURNING id
	`, projectID, req.StageID, req.CatalogID, req.CustomName,
		req.AssignedTo, req.Notes,
	).Scan(&id)
	return id, err
}

func (r *PipelineRepo) UpdateOperation(ctx context.Context, operationID string, req models.UpdateOperationRequest) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE stage_operations SET
			status      = COALESCE($1, status),
			assigned_to = COALESCE(NULLIF($2,'')::uuid, assigned_to),
			notes       = COALESCE($3, notes),
			started_at  = CASE
				WHEN $1 = 'in_progress' AND started_at IS NULL THEN NOW()
				ELSE started_at
			END,
			finished_at = CASE
				WHEN $1 = 'done' AND finished_at IS NULL THEN NOW()
				ELSE finished_at
			END
		WHERE id = $4
	`, req.Status, req.AssignedTo, req.Notes, operationID)
	return err
}

func (r *PipelineRepo) DeleteOperation(ctx context.Context, operationID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM stage_operations WHERE id = $1`, operationID)
	return err
}

// ── Operation Materials ───────────────────────────────────

func (r *PipelineRepo) MaterialsByOperation(ctx context.Context, operationID string) ([]models.OperationMaterial, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			id, operation_id, project_id, name,
			quantity, unit, unit_price, total_price,
			COALESCE(supplier,''), COALESCE(supplier_phone,''),
			CAST(purchased_at AS TEXT),
			COALESCE(notes,''),
			COALESCE(CAST(created_by AS TEXT),''),
			created_at
		FROM operation_materials
		WHERE operation_id = $1
		ORDER BY created_at ASC
	`, operationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.OperationMaterial
	for rows.Next() {
		var m models.OperationMaterial
		var purchasedAt sql.NullString
		rows.Scan(
			&m.ID, &m.OperationID, &m.ProjectID, &m.Name,
			&m.Quantity, &m.Unit, &m.UnitPrice, &m.TotalPrice,
			&m.Supplier, &m.SupplierPhone,
			&purchasedAt, &m.Notes, &m.CreatedBy, &m.CreatedAt,
		)
		if purchasedAt.Valid {
			m.PurchasedAt = &purchasedAt.String
		}
		result = append(result, m)
	}
	return result, nil
}

func (r *PipelineRepo) MaterialsByProject(ctx context.Context, projectID string) ([]models.OperationMaterial, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			m.id, m.operation_id, m.project_id, m.name,
			m.quantity, m.unit, m.unit_price, m.total_price,
			COALESCE(m.supplier,''), COALESCE(m.supplier_phone,''),
			CAST(m.purchased_at AS TEXT),
			COALESCE(m.notes,''),
			COALESCE(CAST(m.created_by AS TEXT),''),
			m.created_at
		FROM operation_materials m
		WHERE m.project_id = $1
		ORDER BY m.created_at ASC
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.OperationMaterial
	for rows.Next() {
		var m models.OperationMaterial
		var purchasedAt sql.NullString
		rows.Scan(
			&m.ID, &m.OperationID, &m.ProjectID, &m.Name,
			&m.Quantity, &m.Unit, &m.UnitPrice, &m.TotalPrice,
			&m.Supplier, &m.SupplierPhone,
			&purchasedAt, &m.Notes, &m.CreatedBy, &m.CreatedAt,
		)
		if purchasedAt.Valid {
			m.PurchasedAt = &purchasedAt.String
		}
		result = append(result, m)
	}
	return result, nil
}

func (r *PipelineRepo) CreateMaterial(ctx context.Context, operationID, projectID, createdBy string, req models.CreateMaterialRequest) (string, error) {
	if req.Unit == "" {
		req.Unit = "шт"
	}
	var id string
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO operation_materials
			(operation_id, project_id, name, quantity, unit, unit_price,
			 supplier, supplier_phone, purchased_at, notes, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NULLIF($9,'')::date,$10,NULLIF($11,'')::uuid)
		RETURNING id
	`, operationID, projectID, req.Name, req.Quantity, req.Unit, req.UnitPrice,
		req.Supplier, req.SupplierPhone, req.PurchasedAt, req.Notes, createdBy,
	).Scan(&id)
	return id, err
}

func (r *PipelineRepo) DeleteMaterial(ctx context.Context, materialID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM operation_materials WHERE id = $1`, materialID)
	return err
}

// ── Stage Files ───────────────────────────────────────────

func (r *PipelineRepo) FilesByStage(ctx context.Context, stageID string) ([]models.StageFile, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, project_id, stage_id, file_name, file_url,
		       COALESCE(file_type,''), COALESCE(file_size,0),
		       COALESCE(CAST(uploaded_by AS TEXT),''),
		       COALESCE(category,'other'),
		       created_at
		FROM stage_files
		WHERE stage_id = $1
		ORDER BY category, created_at DESC
	`, stageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.StageFile
	for rows.Next() {
		var f models.StageFile
		rows.Scan(&f.ID, &f.ProjectID, &f.StageID, &f.FileName, &f.FileURL,
			&f.FileType, &f.FileSize, &f.UploadedBy, &f.Category, &f.CreatedAt)
		result = append(result, f)
	}
	return result, nil
}

func (r *PipelineRepo) CreateFile(ctx context.Context, projectID, stageID, uploadedBy string, req models.CreateFileRequest) (string, error) {
	category := req.Category
	if category == "" {
		category = "other"
	}
	var id string
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO stage_files (project_id, stage_id, file_name, file_url, file_type, file_size, uploaded_by, category)
		VALUES (NULLIF($1,'')::uuid, $2, $3, $4, $5, $6, NULLIF($7,'')::uuid, $8)
		RETURNING id
	`, projectID, stageID, req.FileName, req.FileURL, req.FileType, req.FileSize, uploadedBy, category,
	).Scan(&id)
	return id, err
}

func (r *PipelineRepo) DeleteFile(ctx context.Context, fileID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM stage_files WHERE id = $1`, fileID)
	return err
}

// ── Project History ───────────────────────────────────────

func (r *PipelineRepo) History(ctx context.Context, projectID string) ([]models.ProjectHistory, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			ph.id, ph.project_id,
			COALESCE(ph.from_stage,''), COALESCE(ph.to_stage,''),
			COALESCE(CAST(ph.changed_by AS TEXT),''),
			COALESCE(u.full_name,''),
			COALESCE(ph.comment,''), ph.created_at
		FROM project_history ph
		LEFT JOIN users u ON u.id = ph.changed_by
		WHERE ph.project_id = $1
		ORDER BY ph.created_at DESC
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.ProjectHistory
	for rows.Next() {
		var h models.ProjectHistory
		rows.Scan(&h.ID, &h.ProjectID, &h.FromStage, &h.ToStage,
			&h.ChangedBy, &h.ChangerName, &h.Comment, &h.CreatedAt)
		result = append(result, h)
	}
	return result, nil
}

// ── Project total cost ────────────────────────────────────

type ProjectCost struct {
	ProjectID string  `json:"project_id"`
	TotalCost float64 `json:"total_cost"`
}

func (r *PipelineRepo) TotalCost(ctx context.Context, projectID string) (float64, error) {
	var total float64
	err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(total_price), 0)
		FROM operation_materials
		WHERE project_id = $1
	`, projectID).Scan(&total)
	return total, err
}