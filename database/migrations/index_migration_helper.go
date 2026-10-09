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

func dropTextIndexIfExists(tableName, indexName string) error {
	if !facades.Schema().HasTable(tableName) {
		return nil
	}

	exists, err := hasIndex(tableName, indexName)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	return dropIndexCompatible(tableName, indexName)
}

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

func dropIndexCompatible(tableName, indexName string) error {
	query := facades.Orm().Query()

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
