package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/xuthus5/boxd/control"
)

// TestResultRepository persists latency history for nodes.
type TestResultRepository struct {
	db *sql.DB
}

var _ control.TestResultRepository = (*TestResultRepository)(nil)

func (r *TestResultRepository) Save(ctx context.Context, result control.NodeTestResult) error {
	testedAt := result.TestedAt
	if testedAt.IsZero() {
		testedAt = time.Now().UTC()
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO test_results(node_id, kind, latency_ms, error, tested_at, url)
		VALUES(?, ?, ?, ?, ?, ?)`,
		result.NodeID, result.Kind, result.Latency.Milliseconds(), result.Error, testedAt.UTC().UnixMilli(), result.URL,
	)
	return err
}

func (r *TestResultRepository) List(ctx context.Context, nodeID string, limit int) ([]control.NodeTestResult, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT node_id, kind, latency_ms, error, tested_at, url
		FROM test_results WHERE node_id = ?
		ORDER BY tested_at DESC LIMIT ?`, nodeID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]control.NodeTestResult, 0)
	for rows.Next() {
		var item control.NodeTestResult
		var latency sql.NullInt64
		var testedAt int64
		if err := rows.Scan(&item.NodeID, &item.Kind, &latency, &item.Error, &testedAt, &item.URL); err != nil {
			return nil, err
		}
		if latency.Valid {
			item.Latency = time.Duration(latency.Int64) * time.Millisecond
		}
		item.TestedAt = time.UnixMilli(testedAt).UTC()
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
