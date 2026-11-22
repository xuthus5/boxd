CREATE TABLE control_profile (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    revision TEXT NOT NULL,
    state_json TEXT NOT NULL,
    updated_at INTEGER NOT NULL
);
