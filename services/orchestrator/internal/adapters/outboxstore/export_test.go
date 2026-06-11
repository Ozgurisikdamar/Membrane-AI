package outboxstore

import "context"

// Test-only helpers for the integration test.

// CountForTest returns (audit rows, pending outbox rows) for one tenant.
func CountForTest(ctx context.Context, s *Store, orgID string) (audits, pending int, err error) {
	if err = s.pool.QueryRow(ctx,
		`SELECT count(*) FROM verdict_audit WHERE org_id = $1`, orgID).Scan(&audits); err != nil {
		return 0, 0, err
	}
	err = s.pool.QueryRow(ctx,
		`SELECT count(*) FROM outbox WHERE partition_key = $1 AND published_at IS NULL`, orgID).Scan(&pending)
	return audits, pending, err
}

// CleanupForTest removes the tenant's audit and outbox rows.
func CleanupForTest(ctx context.Context, s *Store, orgID string) error {
	if _, err := s.pool.Exec(ctx, `DELETE FROM verdict_audit WHERE org_id = $1`, orgID); err != nil {
		return err
	}
	_, err := s.pool.Exec(ctx, `DELETE FROM outbox WHERE partition_key = $1`, orgID)
	return err
}
