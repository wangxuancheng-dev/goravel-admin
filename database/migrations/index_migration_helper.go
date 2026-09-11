package migrations

import (
	"fmt"
	"strings"

	"github.com/goravel/framework/contracts/database/orm"
	"github.com/goravel/framework/facades"
)

const (
	dbTypePostgres = "postgres"
	dbTypeMariaDB  = "mariadb"
	dbTypeMySQL    = "mysql"
	dbTypeUnknown  = "unknown"
)

func hasIndex(tableName, indexName string) (bool, error) {
	indexes, err := facades.Schema().GetIndexes(tableName)
	if err != nil {
		return false, err
	}

	for _, index := range indexes {
		if index.Name == indexName {
			return true, nil
		}
	}

	// Schema().GetIndexes may omit FULLTEXT indexes on some MySQL/MariaDB drivers.
	if ok, err := hasIndexInInformationSchema(tableName, indexName); err != nil {
		return false, err
	} else if ok {
		return true, nil
	}

	return false, nil
}

func hasIndexInInformationSchema(tableName, indexName string) (bool, error) {
	query := facades.Orm().Query()
	isDM := facades.Config().GetString("database.default") == "dm"
	if isDM {
		return false, nil
	}

	dbType, err := detectDBType(query)
	if err != nil || dbType == dbTypePostgres || dbType == dbTypeUnknown {
		return false, nil
	}

	var count int64
	sql := `
SELECT COUNT(1)
FROM information_schema.statistics
WHERE table_schema = DATABASE()
  AND table_name = ?
  AND index_name = ?`
	if err := query.Raw(sql, tableName, indexName).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func isDuplicateIndexError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate key name") ||
		strings.Contains(msg, "already exists") ||
		strings.Contains(msg, "1061")
}

func detectDBType(query orm.Query) (string, error) {
	var version string
	if err := query.Raw("SELECT VERSION()").Scan(&version); err != nil {
		return dbTypeUnknown, err
	}

	switch {
	case strings.Contains(version, "PostgreSQL"):
		return dbTypePostgres, nil
	case strings.Contains(version, "MariaDB"):
		return dbTypeMariaDB, nil
	default:
		return dbTypeMySQL, nil
	}
}

func createCompatibleTextIndex(tableName, columnName, indexName string) error {
	query := facades.Orm().Query()

	isDM := facades.Config().GetString("database.default") == "dm"
	if isDM {
		attempts := []string{
			// DM: 优先尝试带中文词法器的全文索引（不同版本/配置语法存在差异）
			fmt.Sprintf("CREATE CONTEXT INDEX %s ON %s(%s) LEXER chinese_lexer", indexName, tableName, columnName),
			fmt.Sprintf("CREATE CONTEXT INDEX %s ON %s(%s) LEXER CHINESE_LEXER", indexName, tableName, columnName),
			fmt.Sprintf("CREATE CONTEXT INDEX %s ON %s(%s) LEXER 'chinese_lexer'", indexName, tableName, columnName),
			fmt.Sprintf("CREATE CONTEXT INDEX %s ON %s(%s)", indexName, tableName, columnName),
			fmt.Sprintf("CREATE FULLTEXT INDEX %s ON %s(%s)", indexName, tableName, columnName),
			fmt.Sprintf("CREATE INDEX %s ON %s(%s)", indexName, tableName, columnName),
		}
		for _, sql := range attempts {
			if _, err := query.Exec(sql); err == nil {
				return nil
			}
		}
		facades.Log().Warningf("skip dm text index migration: table=%s index=%s all create-index attempts failed", tableName, indexName)
		return nil
	}

	dbType, err := detectDBType(query)
	if err != nil {
		return err
	}

	switch dbType {
	case dbTypePostgres:
		if _, err := query.Exec("CREATE EXTENSION IF NOT EXISTS pg_trgm"); err != nil {
			facades.Log().Warningf("skip pg_trgm index migration: table=%s index=%s cannot create extension, err=%v", tableName, indexName, err)
			return nil
		}

		sql := fmt.Sprintf("CREATE INDEX %s ON %s USING GIN(%s gin_trgm_ops)", indexName, tableName, columnName)
		if _, err := query.Exec(sql); err != nil {
			if isDuplicateIndexError(err) {
				return nil
			}
			facades.Log().Warningf("skip pg_trgm index migration: table=%s index=%s cannot create index, err=%v", tableName, indexName, err)
		}
		return nil
	case dbTypeMariaDB:
		sql := fmt.Sprintf("CREATE FULLTEXT INDEX %s ON %s(%s)", indexName, tableName, columnName)
		if _, err := query.Exec(sql); err != nil {
			if isDuplicateIndexError(err) {
				return nil
			}
			facades.Log().Warningf("skip mariadb fulltext index migration: table=%s index=%s cannot create index, err=%v", tableName, indexName, err)
		}
		return nil
	default:
		sql := fmt.Sprintf("CREATE FULLTEXT INDEX %s ON %s(%s) WITH PARSER ngram", indexName, tableName, columnName)
		if _, err := query.Exec(sql); err != nil {
			if isDuplicateIndexError(err) {
				return nil
			}
			errStr := strings.ToLower(err.Error())
			if strings.Contains(errStr, "ngram") && (strings.Contains(errStr, "not defined") || strings.Contains(errStr, "not supported")) {
				fallbackSQL := fmt.Sprintf("CREATE FULLTEXT INDEX %s ON %s(%s)", indexName, tableName, columnName)
				if _, fallbackErr := query.Exec(fallbackSQL); fallbackErr != nil {
					if isDuplicateIndexError(fallbackErr) {
						return nil
					}
					facades.Log().Warningf("skip mysql fulltext index migration: table=%s index=%s fallback create index failed, err=%v", tableName, indexName, fallbackErr)
				}
				return nil
			}

			facades.Log().Warningf("skip mysql fulltext index migration: table=%s index=%s create index failed, err=%v", tableName, indexName, err)
		}
		return nil
	}
}

func dropIndexCompatible(tableName, indexName string) error {
	query := facades.Orm().Query()

	isDM := facades.Config().GetString("database.default") == "dm"
	if isDM {
		dropAttempts := []string{
			fmt.Sprintf("DROP INDEX %s", indexName),
			fmt.Sprintf("DROP INDEX %s ON %s", indexName, tableName),
		}
		for _, sql := range dropAttempts {
			if _, err := query.Exec(sql); err == nil {
				return nil
			}
		}
		facades.Log().Warningf("skip drop dm index: table=%s index=%s all drop-index attempts failed", tableName, indexName)
		return nil
	}

	dbType, err := detectDBType(query)
	if err != nil {
		return err
	}

	if dbType == dbTypePostgres {
		sql := fmt.Sprintf("DROP INDEX IF EXISTS %s", indexName)
		if _, err := query.Exec(sql); err != nil {
			facades.Log().Warningf("skip drop pg index: table=%s index=%s err=%v", tableName, indexName, err)
		}
		return nil
	}

	sql := fmt.Sprintf("DROP INDEX %s ON %s", indexName, tableName)
	if _, err := query.Exec(sql); err != nil {
		facades.Log().Warningf("skip drop mysql index: table=%s index=%s err=%v", tableName, indexName, err)
	}
	return nil
}
