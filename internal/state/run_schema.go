package state

import (
	"context"
	"database/sql"
)

// Keep this list aligned with openStore's idempotent schema initialization.
// The schema inventory test rejects unlisted objects when the schema grows.
const runSchemaObjects = `WITH required(type,name) AS (VALUES
 ('table','generations'),('table','active'),('table','leases'),
 ('table','lease_identity'),('table','lease_completion'),('table','operations'),
 ('table','operation_tree_holds'),('table','operation_identity'),('table','operation_tracked'),
 ('table','operation_children'),('table','operation_child_completion'),('table','operation_tree_completed'),
 ('table','generation_python'),('table','generation_evidence'),('table','deleting_generations'),('table','deleting_operations'),
 ('index','leases_generation_page'),('index','operation_children_owner'),
 ('trigger','protect_deleting_preparation'),('trigger','protect_deleting_operation_update'),
 ('trigger','protect_deleting_active'),('trigger','protect_deleting_lease'),('trigger','protect_deleting_operation')
) `

func hasRunSchema(ctx context.Context, db *sql.DB) (bool, error) {
	var complete bool
	err := db.QueryRowContext(ctx, runSchemaObjects+`SELECT NOT EXISTS (
 SELECT 1 FROM required r WHERE NOT EXISTS (
	 SELECT 1 FROM sqlite_master s WHERE s.type=r.type AND s.name=r.name))`).Scan(&complete)
	if err == nil && complete {
		err = db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM active WHERE singleton=1)`).Scan(&complete)
	}
	return complete, err
}
