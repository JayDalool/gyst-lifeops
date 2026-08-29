ALTER TABLE projects ADD COLUMN color TEXT NOT NULL DEFAULT 'cyan';

UPDATE projects
SET color = CASE (id - 1) % 8
    WHEN 0 THEN 'cyan'
    WHEN 1 THEN 'violet'
    WHEN 2 THEN 'emerald'
    WHEN 3 THEN 'amber'
    WHEN 4 THEN 'rose'
    WHEN 5 THEN 'blue'
    WHEN 6 THEN 'orange'
    ELSE 'magenta'
END;
