PRAGMA foreign_keys = ON;

-- Self-paced learning. A course is a long-running container (Udemy, bootcamp,
-- YouTube series, etc.); progress_percent is entered manually since providers
-- already report it and deriving from sparse blocks is unreliable. Nothing here
-- requires fixed lecture times.
CREATE TABLE IF NOT EXISTS courses (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT NOT NULL UNIQUE DEFAULT (lower(hex(randomblob(16)))),
    name TEXT NOT NULL,
    provider TEXT NOT NULL DEFAULT '',
    instructor TEXT NOT NULL DEFAULT '',
    url TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused', 'completed', 'archived')),
    progress_percent INTEGER NOT NULL DEFAULT 0,
    target_date TEXT,
    notes TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    archived_at TEXT,
    sync_version INTEGER NOT NULL DEFAULT 1,
    last_modified_device_id TEXT
);

CREATE INDEX IF NOT EXISTS idx_courses_status ON courses(status);

-- A learning block is any unit of a course: a lesson/module to watch, a study
-- or review block, an assignment/exam deadline, or an optional live session.
-- due_date is the day it lands on the calendar (optional for unscheduled
-- self-paced lessons); start_time/end_time are optional clock times on that day
-- for the rare scheduled block. is_catchup marks a block spawned by postponing.
CREATE TABLE IF NOT EXISTS learning_blocks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT NOT NULL UNIQUE DEFAULT (lower(hex(randomblob(16)))),
    course_id INTEGER NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    block_type TEXT NOT NULL DEFAULT 'lesson' CHECK (block_type IN ('lesson', 'module', 'study', 'assignment', 'review', 'exam', 'live_session')),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'completed', 'skipped', 'postponed')),
    start_time TEXT,
    end_time TEXT,
    due_date TEXT,
    url TEXT NOT NULL DEFAULT '',
    repeat_type TEXT NOT NULL DEFAULT 'none' CHECK (repeat_type IN ('none', 'daily', 'weekly', 'monthly')),
    notes TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_catchup INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TEXT,
    archived_at TEXT,
    sync_version INTEGER NOT NULL DEFAULT 1,
    last_modified_device_id TEXT
);

CREATE INDEX IF NOT EXISTS idx_learning_blocks_course ON learning_blocks(course_id);
CREATE INDEX IF NOT EXISTS idx_learning_blocks_due ON learning_blocks(due_date);
CREATE INDEX IF NOT EXISTS idx_learning_blocks_status ON learning_blocks(status);
