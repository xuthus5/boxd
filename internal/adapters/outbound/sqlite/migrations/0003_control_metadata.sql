CREATE TABLE test_results_v2 (
    node_id TEXT NOT NULL,
    kind TEXT NOT NULL,
    latency_ms INTEGER,
    error TEXT,
    tested_at INTEGER NOT NULL,
    url TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (node_id, kind, tested_at)
);

INSERT INTO test_results_v2(node_id, kind, latency_ms, error, tested_at, url)
SELECT node_id, kind, latency_ms, error, tested_at, '' FROM test_results;

DROP TABLE test_results;

ALTER TABLE test_results_v2 RENAME TO test_results;

CREATE INDEX idx_test_results_node_tested ON test_results(node_id, tested_at DESC);

ALTER TABLE rule_set_status ADD COLUMN last_modified TEXT NOT NULL DEFAULT '';

ALTER TABLE rule_set_status ADD COLUMN error TEXT NOT NULL DEFAULT '';
