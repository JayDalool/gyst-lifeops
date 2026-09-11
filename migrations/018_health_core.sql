-- Health Core is additive. No provider records or example health data are seeded.
CREATE TABLE health_cycles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL DEFAULT '',
    start_date TEXT NOT NULL DEFAULT '',
    end_date TEXT NOT NULL DEFAULT '',
    primary_goal TEXT NOT NULL DEFAULT '',
    daily_calorie_target REAL NOT NULL DEFAULT 0 CHECK (daily_calorie_target >= 0),
    protein_target_g REAL NOT NULL DEFAULT 0 CHECK (protein_target_g >= 0),
    carbs_target_g REAL NOT NULL DEFAULT 0 CHECK (carbs_target_g >= 0),
    fat_target_g REAL NOT NULL DEFAULT 0 CHECK (fat_target_g >= 0),
    fiber_target_g REAL NOT NULL DEFAULT 0 CHECK (fiber_target_g >= 0),
    target_deficit_kcal REAL NOT NULL DEFAULT 0 CHECK (target_deficit_kcal >= 0),
    strength_sessions_per_week INTEGER NOT NULL DEFAULT 0 CHECK (strength_sessions_per_week >= 0),
    cardio_sessions_per_week INTEGER NOT NULL DEFAULT 0 CHECK (cardio_sessions_per_week >= 0),
    step_target INTEGER NOT NULL DEFAULT 0 CHECK (step_target >= 0),
    notes TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (end_date > start_date)
);
CREATE INDEX idx_health_cycles_dates ON health_cycles(start_date, end_date);

CREATE TABLE daily_health_snapshots (
    date TEXT PRIMARY KEY NOT NULL,
    dietary_calories REAL CHECK (dietary_calories >= 0),
    protein_g REAL CHECK (protein_g >= 0),
    carbs_g REAL CHECK (carbs_g >= 0),
    fat_g REAL CHECK (fat_g >= 0),
    fiber_g REAL CHECK (fiber_g >= 0),
    steps_watch INTEGER CHECK (steps_watch >= 0),
    steps_phone INTEGER CHECK (steps_phone >= 0),
    active_energy_kcal REAL CHECK (active_energy_kcal >= 0),
    resting_energy_kcal REAL CHECK (resting_energy_kcal >= 0),
    exercise_minutes REAL CHECK (exercise_minutes >= 0),
    last_synced_at TEXT NOT NULL
);

CREATE TABLE health_weight_entries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    measured_at TEXT NOT NULL,
    weight_kg REAL NOT NULL CHECK (weight_kg > 0),
    source TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_health_weight_measured ON health_weight_entries(measured_at, id);

CREATE TABLE health_sync_states (
    source TEXT PRIMARY KEY NOT NULL,
    last_synced_at TEXT NOT NULL,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
