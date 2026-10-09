package migrations

type M20251227063517AddFulltextIndexToOperationLogsRequest struct{}

// Signature The unique signature for the migration.
func (r *M20251227063517AddFulltextIndexToOperationLogsRequest) Signature() string {
	return "20251227063517_add_fulltext_index_to_operation_logs_request"
}

// Up is a no-op: operation log request search uses LIKE by default and does not require a fulltext index.
func (r *M20251227063517AddFulltextIndexToOperationLogsRequest) Up() error {
	return nil
}

// Down Reverse the migrations (drop leftover ft_request if present from older builds).
func (r *M20251227063517AddFulltextIndexToOperationLogsRequest) Down() error {
	const tableName = "operation_logs"
	const indexName = "ft_request"

	return dropTextIndexIfExists(tableName, indexName)
}
