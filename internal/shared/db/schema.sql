-- Categories for organizing habits
CREATE TABLE IF NOT EXISTS categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    color TEXT NOT NULL DEFAULT '#FFFFFF',
    emoji TEXT DEFAULT '📁',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Habits definition
CREATE TABLE IF NOT EXISTS habits (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    description TEXT DEFAULT '',
    emoji TEXT DEFAULT '',
    category_id INTEGER REFERENCES categories(id) ON DELETE SET NULL,
    frequency_type TEXT NOT NULL DEFAULT 'daily',
    frequency_value INTEGER DEFAULT 1,
    target_per_day INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    archived_at DATETIME
);

-- Completion records
CREATE TABLE IF NOT EXISTS completions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    habit_id INTEGER NOT NULL REFERENCES habits(id) ON DELETE CASCADE,
    completed_at DATE NOT NULL,
    notes TEXT DEFAULT ''
);

-- App settings
CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_habits_category ON habits(category_id);
CREATE INDEX IF NOT EXISTS idx_habits_archived ON habits(archived_at);
CREATE INDEX IF NOT EXISTS idx_completions_habit ON completions(habit_id);
CREATE INDEX IF NOT EXISTS idx_completions_date ON completions(completed_at);
