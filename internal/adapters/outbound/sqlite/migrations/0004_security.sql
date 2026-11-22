CREATE TABLE security_credential (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    salt BLOB NOT NULL,
    hash BLOB NOT NULL,
    iterations INTEGER NOT NULL CHECK (iterations > 0),
    updated_at INTEGER NOT NULL
);
