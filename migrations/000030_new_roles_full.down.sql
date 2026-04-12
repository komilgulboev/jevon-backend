DELETE FROM roles WHERE name IN (
    'director', 'accountant', 'assistant_manager', 'project_manager',
    'workshop_manager', 'design_lead', 'cutting_lead', 'painting_lead',
    'soft_furniture_lead', 'procurement', 'seller', 'constructor',
    'carpenter', 'installer', 'cnc_operator', 'soft_furniture_master',
    'sander', 'painter_worker', 'cutter_worker', 'driller', 'edger', 'cook'
);
