package migrations

// M20261009000001DropFulltextIndexFromOperationLogsRequest removes leftover ft_request
// created by older builds. Request filtering uses LIKE and does not depend on fulltext.
type M20261009000001DropFulltextIndexFromOperationLogsRequest struct{}

// Signature The unique signature for the migration.
func (r *M20261009000001DropFulltextIndexFromOperationLogsRequest) Signature() string {
	return "20261009000001_drop_fulltext_index_from_operation_logs_request"
}

// Up Run the migrations.
func (r *M20261009000001DropFulltextIndexFromOperationLogsRequest) Up() error {
	return dropTextIndexIfExists("operation_logs", "ft_request")
}

// Down Reverse the migrations (intentionally no-op; fulltext is not recreated).
func (r *M20261009000001DropFulltextIndexFromOperationLogsRequest) Down() error {
	return nil
}
