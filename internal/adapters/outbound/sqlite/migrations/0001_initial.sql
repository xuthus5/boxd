CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE subscriptions (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    url TEXT NOT NULL,
    interval_seconds INTEGER NOT NULL DEFAULT 0,
    enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE nodes (
    id TEXT PRIMARY KEY,
    subscription_id TEXT REFERENCES subscriptions(id) ON DELETE SET NULL,
    tag TEXT NOT NULL,
    protocol TEXT NOT NULL,
    config_json TEXT NOT NULL,
    selected INTEGER NOT NULL DEFAULT 0 CHECK (selected IN (0, 1)),
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE test_results (
    node_id TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    latency_ms INTEGER,
    error TEXT,
    tested_at INTEGER NOT NULL,
    PRIMARY KEY (node_id, kind, tested_at)
);

CREATE TABLE config_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    revision TEXT NOT NULL UNIQUE,
    source TEXT NOT NULL,
    mode TEXT NOT NULL,
    summary TEXT NOT NULL DEFAULT '',
    format TEXT NOT NULL DEFAULT 'json',
    body BLOB NOT NULL,
    created_at INTEGER NOT NULL
);

CREATE TABLE config_metadata (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE rule_set_status (
    name TEXT PRIMARY KEY,
    source TEXT NOT NULL,
    etag TEXT NOT NULL DEFAULT '',
    sha256 TEXT NOT NULL DEFAULT '',
    updated_at INTEGER NOT NULL
);

CREATE TABLE audit_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    action TEXT NOT NULL,
    actor TEXT NOT NULL,
    source TEXT NOT NULL,
    detail_json TEXT NOT NULL DEFAULT '{}',
    created_at INTEGER NOT NULL
);

CREATE INDEX idx_nodes_subscription_id ON nodes(subscription_id);
CREATE INDEX idx_test_results_node_tested ON test_results(node_id, tested_at DESC);
CREATE INDEX idx_config_history_created ON config_history(created_at DESC);
CREATE INDEX idx_audit_events_created ON audit_events(created_at DESC);
CREATE INDEX idx_subscriptions_enabled_updated ON subscriptions(enabled, updated_at);
