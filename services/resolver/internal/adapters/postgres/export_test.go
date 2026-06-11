package postgres

import "context"

// Test-only helpers (compiled only with the test binary): seed and clean a
// tenant so the integration test is self-contained against a migrated dev DB.

// SeedForTest inserts the organization (if missing) and one gold row.
func SeedForTest(ctx context.Context, g *GoldIndex, orgID, filePath, language, content, vectorLiteral string) error {
	if _, err := g.pool.Exec(ctx,
		`INSERT INTO enterprise_organization (org_id, company_name)
		 VALUES ($1, 'integration-test') ON CONFLICT (org_id) DO NOTHING`, orgID); err != nil {
		return err
	}
	_, err := g.pool.Exec(ctx,
		`INSERT INTO gold_codebase_index
		   (org_id, file_path, language_tag, raw_code_content, architectural_context, embedding)
		 VALUES ($1, $2, $3, $4, 'test context', $5::vector)`,
		orgID, filePath, language, content, vectorLiteral)
	return err
}

// CleanupForTest removes the tenant; gold rows cascade.
func CleanupForTest(ctx context.Context, g *GoldIndex, orgID string) error {
	_, err := g.pool.Exec(ctx, `DELETE FROM enterprise_organization WHERE org_id = $1`, orgID)
	return err
}
