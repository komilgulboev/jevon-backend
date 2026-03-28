-- ================================================================
-- АВТОМАТИЧЕСКОЕ СОЗДАНИЕ ЭТАПОВ ПРИ СОЗДАНИИ ПРОЕКТА
-- При создании нового проекта автоматически создаются все 8 этапов
-- со статусом 'pending'. Первый этап (intake) сразу in_progress.
-- ================================================================

CREATE OR REPLACE FUNCTION create_project_stages()
RETURNS TRIGGER AS $$
DECLARE
    stages TEXT[] := ARRAY[
        'intake','design','cutting','production',
        'warehouse','delivery','assembly','handover'
    ];
    s TEXT;
BEGIN
    FOREACH s IN ARRAY stages LOOP
        INSERT INTO project_stages (project_id, stage, status)
        VALUES (
            NEW.id,
            s,
            CASE WHEN s = 'intake' THEN 'in_progress' ELSE 'pending' END
        );
    END LOOP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_create_project_stages
    AFTER INSERT ON projects
    FOR EACH ROW EXECUTE FUNCTION create_project_stages();

-- ================================================================
-- АВТОМАТИЧЕСКОЕ ОБНОВЛЕНИЕ current_stage ПРОЕКТА
-- При завершении этапа (status → done) автоматически
-- переключает проект на следующий этап
-- ================================================================

CREATE OR REPLACE FUNCTION advance_project_stage()
RETURNS TRIGGER AS $$
DECLARE
    next_stage TEXT;
    stage_order TEXT[] := ARRAY[
        'intake','design','cutting','production',
        'warehouse','delivery','assembly','handover'
    ];
    current_idx INT;
BEGIN
    -- Срабатывает только когда статус меняется на 'done'
    IF NEW.status = 'done' AND OLD.status != 'done' THEN

        -- Найти индекс текущего этапа
        SELECT array_position(stage_order, NEW.stage) INTO current_idx;

        -- Определить следующий этап
        IF current_idx < array_length(stage_order, 1) THEN
            next_stage := stage_order[current_idx + 1];

            -- Обновить current_stage проекта
            UPDATE projects
            SET current_stage = next_stage
            WHERE id = NEW.project_id;

            -- Переключить следующий этап в in_progress
            UPDATE project_stages
            SET status = 'in_progress', started_at = NOW()
            WHERE project_id = NEW.project_id AND stage = next_stage;

        ELSE
            -- Все этапы пройдены — проект завершён
            UPDATE projects
            SET current_stage = 'done', status = 'done'
            WHERE id = NEW.project_id;
        END IF;

        -- Записать в историю
        INSERT INTO project_history (project_id, from_stage, to_stage, changed_by)
        SELECT NEW.project_id, NEW.stage, COALESCE(next_stage, 'done'), NEW.assigned_to;

        -- Проставить время завершения этапа
        NEW.finished_at = NOW();
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_advance_project_stage
    BEFORE UPDATE ON project_stages
    FOR EACH ROW EXECUTE FUNCTION advance_project_stage();
