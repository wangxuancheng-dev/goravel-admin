# MySQL 常用命令

## 连接数据库

```bash
# 查看版本
mysql --version
# 或
mysql -V

# 连接数据库
mysql -u root -p -h 127.0.0.1 -P 3306

# 连接指定数据库
mysql -u root -p -h 127.0.0.1 -P 3306 database_name

# 使用密码连接（不安全，仅用于脚本）
mysql -u root -p'password' -h 127.0.0.1 -P 3306

# 执行 SQL 文件
mysql -u root -p -h 127.0.0.1 -P 3306 database_name < query.sql

# 执行 SQL 命令
mysql -u root -p -h 127.0.0.1 -P 3306 -e "SHOW DATABASES;"

# 退出交互界面
exit
# 或
\q
# 或
quit
```

## 数据库管理

```sql
-- 列出所有数据库
SHOW DATABASES;

-- 创建数据库
CREATE DATABASE database_name;
CREATE DATABASE IF NOT EXISTS database_name;

-- 创建数据库并指定字符集
CREATE DATABASE database_name CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 删除数据库
DROP DATABASE database_name;
DROP DATABASE IF EXISTS database_name;

-- 切换数据库
USE database_name;

-- 查看当前数据库
SELECT DATABASE();

-- 查看数据库字符集
SHOW CREATE DATABASE database_name;

-- 修改数据库字符集
ALTER DATABASE database_name CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 查看数据库大小
SELECT 
    table_schema AS 'Database',
    ROUND(SUM(data_length + index_length) / 1024 / 1024, 2) AS 'Size (MB)'
FROM information_schema.tables
WHERE table_schema = 'database_name'
GROUP BY table_schema;

-- 查看所有数据库大小
SELECT 
    table_schema AS 'Database',
    ROUND(SUM(data_length + index_length) / 1024 / 1024, 2) AS 'Size (MB)'
FROM information_schema.tables
GROUP BY table_schema
ORDER BY SUM(data_length + index_length) DESC;
```

## 用户和权限管理

```sql
-- 列出所有用户
SELECT user, host FROM mysql.user;

-- 查看当前用户
SELECT USER();
SELECT CURRENT_USER();

-- 创建用户
CREATE USER 'username'@'localhost' IDENTIFIED BY 'password';
CREATE USER 'username'@'%' IDENTIFIED BY 'password';  -- 允许任意主机

-- 修改用户密码
ALTER USER 'username'@'localhost' IDENTIFIED BY 'new_password';
SET PASSWORD FOR 'username'@'localhost' = PASSWORD('new_password');  -- MySQL 5.7
SET PASSWORD FOR 'username'@'localhost' = 'new_password';  -- MySQL 8.0+

-- 删除用户
DROP USER 'username'@'localhost';
DROP USER IF EXISTS 'username'@'localhost';

-- 授予权限
GRANT ALL PRIVILEGES ON database_name.* TO 'username'@'localhost';
GRANT SELECT, INSERT, UPDATE, DELETE ON database_name.* TO 'username'@'localhost';
GRANT ALL PRIVILEGES ON *.* TO 'username'@'localhost' WITH GRANT OPTION;  -- 所有数据库

-- 授予特定表权限
GRANT SELECT, INSERT ON database_name.table_name TO 'username'@'localhost';

-- 撤销权限
REVOKE ALL PRIVILEGES ON database_name.* FROM 'username'@'localhost';

-- 查看用户权限
SHOW GRANTS FOR 'username'@'localhost';

-- 刷新权限（修改权限后必须执行）
FLUSH PRIVILEGES;
```

## 表管理

```sql
-- 列出所有表
SHOW TABLES;
SHOW TABLES FROM database_name;

-- 查看表结构
DESCRIBE table_name;
DESC table_name;
SHOW COLUMNS FROM table_name;
SHOW CREATE TABLE table_name;  -- 查看建表语句

-- 查看表状态
SHOW TABLE STATUS LIKE 'table_name';
SHOW TABLE STATUS FROM database_name;

-- 查看表大小
SELECT 
    table_name AS 'Table',
    ROUND(((data_length + index_length) / 1024 / 1024), 2) AS 'Size (MB)',
    table_rows AS 'Rows'
FROM information_schema.tables
WHERE table_schema = 'database_name'
  AND table_name = 'table_name';

-- 查看所有表大小
SELECT 
    table_schema AS 'Database',
    table_name AS 'Table',
    ROUND(((data_length + index_length) / 1024 / 1024), 2) AS 'Size (MB)',
    table_rows AS 'Rows'
FROM information_schema.tables
WHERE table_schema = 'database_name'
ORDER BY (data_length + index_length) DESC;

-- 查看表行数
SELECT COUNT(*) FROM table_name;

-- 查看表引擎
SHOW TABLE STATUS WHERE Name = 'table_name';

-- 修改表引擎
ALTER TABLE table_name ENGINE = InnoDB;

-- 修改表字符集
ALTER TABLE table_name CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 重命名表
RENAME TABLE old_table_name TO new_table_name;
ALTER TABLE old_table_name RENAME TO new_table_name;

-- 清空表数据
TRUNCATE TABLE table_name;

-- 删除表
DROP TABLE table_name;
DROP TABLE IF EXISTS table_name;
```

## 索引管理

```sql
-- 列出所有索引
SHOW INDEX FROM table_name;
SHOW INDEXES FROM table_name;
SHOW KEYS FROM table_name;

-- 查看索引详细信息
SELECT 
    TABLE_NAME,
    INDEX_NAME,
    COLUMN_NAME,
    SEQ_IN_INDEX,
    INDEX_TYPE
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = 'database_name'
  AND TABLE_NAME = 'table_name';

-- 创建索引
CREATE INDEX idx_name ON table_name(column_name);

-- 创建唯一索引
CREATE UNIQUE INDEX idx_name ON table_name(column_name);

-- 创建复合索引
CREATE INDEX idx_name ON table_name(column1, column2);

-- 创建全文索引（MyISAM 或 InnoDB 5.6+）
CREATE FULLTEXT INDEX idx_name ON table_name(column_name);

-- 创建前缀索引
CREATE INDEX idx_name ON table_name(column_name(10));

-- 删除索引
DROP INDEX idx_name ON table_name;
ALTER TABLE table_name DROP INDEX idx_name;

-- 查看索引使用情况（需要启用 performance_schema）
SELECT 
    object_schema,
    object_name,
    index_name,
    count_star,
    sum_timer_wait / 1000000000000 AS sum_timer_wait_sec
FROM performance_schema.table_io_waits_summary_by_index_usage
WHERE object_schema = 'database_name'
ORDER BY sum_timer_wait DESC;
```

## 创建全文索引 ngram
```sql
CREATE TABLE table_name(
    id INT UNSIGNED AUTO_INCREMENT NOT NULL PRIMARY KEY,
    title VARCHAR (200),
    content TEXT,
    -- # 建立全文索引，同时使用ngram全文分析器
    FULLTEXT (title) WITH PARSER ngram
) ENGINE = INNODB;

-- # 通过 alter table 的方式来添加
ALTER TABLE table_name ADD FULLTEXT INDEX full_index_title(title) WITH PARSER ngram;

-- # 为title创建全文索引并且使用ngram全文解析器进行分词
CREATE FULLTEXT INDEX full_index_title ON table_name(title) WITH PARSER `ngram`;

-- # 使用双引号把要搜素的词括起来，效果类似于like '%高性能%'
select * ,(MATCH ( `title`  ) AGAINST ( '"高性能"' )  )AS score 
from table_name 
where MATCH(title) AGAINST('"高性能"' IN BOOLEAN MODE) order by score Desc;

SELECT * FROM table_name WHERE MATCH(`title`) AGAINST('高性能' IN BOOLEAN MODE);

-- # 优化全文索引，减少索引碎片
REPAIR TABLE table_name
-- 或更轻量的优化（不锁表）
OPTIMIZE TABLE table_name;

-- 若需匹配单字（如 “微”“信”）
-- 临时设置（会话级）
SET SESSION ngram_token_size = 1;
-- 永久设置（修改 my.cnf 后重启 MySQL）
-- ngram_token_size = 1
```

## 查询和分析

```sql
-- 查看执行计划
EXPLAIN SELECT * FROM table_name WHERE condition;
EXPLAIN FORMAT=JSON SELECT * FROM table_name WHERE condition;
EXPLAIN FORMAT=TREE SELECT * FROM table_name WHERE condition;  -- MySQL 8.0+

-- 分析表（更新统计信息）
ANALYZE TABLE table_name;

-- 查看慢查询日志（需要启用）
SHOW VARIABLES LIKE 'slow_query_log';
SHOW VARIABLES LIKE 'long_query_time';

-- 查看当前正在执行的查询
SHOW PROCESSLIST;
SHOW FULL PROCESSLIST;  -- 显示完整 SQL

-- 查看等待锁的查询
SELECT 
    r.trx_id waiting_trx_id,
    r.trx_mysql_thread_id waiting_thread,
    r.trx_query waiting_query,
    b.trx_id blocking_trx_id,
    b.trx_mysql_thread_id blocking_thread,
    b.trx_query blocking_query
FROM information_schema.innodb_lock_waits w
INNER JOIN information_schema.innodb_trx b ON b.trx_id = w.blocking_trx_id
INNER JOIN information_schema.innodb_trx r ON r.trx_id = w.requesting_trx_id;

-- 终止查询
KILL process_id;
KILL QUERY process_id;  -- 只终止查询，不断开连接
```

## 备份和恢复

```bash
# 备份数据库（SQL格式）
mysqldump -u root -p -h 127.0.0.1 -P 3306 database_name > backup.sql

# 备份所有数据库
mysqldump -u root -p -h 127.0.0.1 -P 3306 --all-databases > all_databases.sql

# 备份单个表
mysqldump -u root -p -h 127.0.0.1 -P 3306 database_name table_name > table_backup.sql

# 仅备份结构（不包含数据）
mysqldump -u root -p -h 127.0.0.1 -P 3306 --no-data database_name > schema.sql

# 仅备份数据（不包含结构）
mysqldump -u root -p -h 127.0.0.1 -P 3306 --no-create-info database_name > data.sql

# 备份并压缩
mysqldump -u root -p -h 127.0.0.1 -P 3306 database_name | gzip > backup.sql.gz

# 恢复数据库（从SQL文件）
mysql -u root -p -h 127.0.0.1 -P 3306 database_name < backup.sql

# 恢复压缩的备份
gunzip < backup.sql.gz | mysql -u root -p -h 127.0.0.1 -P 3306 database_name

# 备份时排除某些表
mysqldump -u root -p -h 127.0.0.1 -P 3306 database_name --ignore-table=database_name.table1 --ignore-table=database_name.table2 > backup.sql

# 备份时添加锁表选项
mysqldump -u root -p -h 127.0.0.1 -P 3306 --single-transaction --routines --triggers database_name > backup.sql
```

## 性能监控

```sql
-- 查看数据库连接数
SHOW STATUS LIKE 'Threads_connected';
SHOW STATUS LIKE 'Max_used_connections';

-- 查看最大连接数
SHOW VARIABLES LIKE 'max_connections';

-- 查看当前配置
SHOW VARIABLES;

-- 查看服务器状态
SHOW STATUS;

-- 查看 InnoDB 状态
SHOW ENGINE INNODB STATUS;

-- 查看表统计信息
SELECT 
    table_schema,
    table_name,
    table_rows,
    ROUND(((data_length + index_length) / 1024 / 1024), 2) AS size_mb
FROM information_schema.tables
WHERE table_schema = 'database_name'
ORDER BY table_rows DESC;

-- 查看缓存命中率
SHOW STATUS LIKE 'Qcache%';
SHOW VARIABLES LIKE 'query_cache%';

-- 查看慢查询统计
SHOW STATUS LIKE 'Slow_queries';

-- 查看当前连接信息
SHOW STATUS LIKE 'Threads%';

-- 查看 InnoDB 缓冲池状态
SHOW STATUS LIKE 'Innodb_buffer_pool%';

-- 查看锁等待情况
SELECT 
    r.trx_id waiting_trx_id,
    r.trx_mysql_thread_id waiting_thread,
    TIMESTAMPDIFF(SECOND, r.trx_wait_started, NOW()) wait_time,
    r.trx_query waiting_query,
    l.lock_table,
    l.lock_mode,
    l.lock_type,
    b.trx_id blocking_trx_id,
    b.trx_mysql_thread_id blocking_thread,
    b.trx_query blocking_query
FROM information_schema.innodb_lock_waits w
INNER JOIN information_schema.innodb_trx b ON b.trx_id = w.blocking_trx_id
INNER JOIN information_schema.innodb_trx r ON r.trx_id = w.requesting_trx_id
INNER JOIN information_schema.innodb_locks l ON l.lock_id = w.requested_lock_id;
```

## 维护操作

```sql
-- 分析表（更新统计信息）
ANALYZE TABLE table_name;

-- 优化表（整理碎片，回收空间）
OPTIMIZE TABLE table_name;

-- 检查表
CHECK TABLE table_name;

-- 修复表（仅 MyISAM）
REPAIR TABLE table_name;

-- 查看表碎片情况
SELECT 
    table_schema,
    table_name,
    ROUND(data_free / 1024 / 1024, 2) AS data_free_mb,
    ROUND((data_length + index_length) / 1024 / 1024, 2) AS total_size_mb,
    ROUND(data_free / (data_length + index_length) * 100, 2) AS fragmentation_percent
FROM information_schema.tables
WHERE table_schema = 'database_name'
  AND data_free > 0
ORDER BY data_free DESC;

-- 刷新表缓存
FLUSH TABLES;

-- 刷新日志
FLUSH LOGS;
```

## 实用命令

```sql
-- 查看当前时间
SELECT NOW();
SELECT CURRENT_TIMESTAMP();

-- 查看时区
SELECT @@global.time_zone, @@session.time_zone;
SHOW VARIABLES LIKE '%time_zone%';

-- 设置时区
SET time_zone = '+08:00';
SET GLOBAL time_zone = '+08:00';

-- 查看当前用户
SELECT USER();
SELECT CURRENT_USER();

-- 查看当前数据库
SELECT DATABASE();

-- 查看 MySQL 版本
SELECT VERSION();
SHOW VARIABLES LIKE 'version%';

-- 查看服务器配置
SHOW VARIABLES LIKE 'datadir';  -- 数据目录
SHOW VARIABLES LIKE 'basedir';  -- 安装目录

-- 复制表结构
CREATE TABLE new_table LIKE old_table;

-- 复制表数据
INSERT INTO new_table SELECT * FROM old_table;

-- 复制表结构和数据
CREATE TABLE new_table AS SELECT * FROM old_table;

-- 查看自增 ID 当前值
SHOW TABLE STATUS WHERE Name = 'table_name';
SELECT AUTO_INCREMENT FROM information_schema.tables WHERE table_schema = 'database_name' AND table_name = 'table_name';

-- 重置自增 ID
ALTER TABLE table_name AUTO_INCREMENT = 1;

-- 查看字符集和排序规则
SHOW VARIABLES LIKE 'character_set%';
SHOW VARIABLES LIKE 'collation%';
```

## 事务管理

```sql
-- 开启事务
START TRANSACTION;
BEGIN;

-- 提交事务
COMMIT;

-- 回滚事务
ROLLBACK;

-- 设置自动提交
SET autocommit = 0;  -- 关闭自动提交
SET autocommit = 1;  -- 开启自动提交

-- 查看事务隔离级别
SELECT @@transaction_isolation;  -- MySQL 8.0+
SELECT @@tx_isolation;  -- MySQL 5.7

-- 设置事务隔离级别
SET SESSION TRANSACTION ISOLATION LEVEL READ UNCOMMITTED;
SET SESSION TRANSACTION ISOLATION LEVEL READ COMMITTED;
SET SESSION TRANSACTION ISOLATION LEVEL REPEATABLE READ;
SET SESSION TRANSACTION ISOLATION LEVEL SERIALIZABLE;

-- 查看当前事务状态
SELECT * FROM information_schema.innodb_trx;
```

## 常用 mysql 命令

```bash
# 在 mysql 交互界面中
SHOW DATABASES;     # 列出所有数据库
USE dbname;         # 切换到数据库
SHOW TABLES;        # 列出当前数据库的所有表
DESC table;         # 显示表结构
SHOW CREATE TABLE table;  # 显示建表语句
SHOW INDEX FROM table;    # 显示索引
SHOW PROCESSLIST;   # 显示当前连接
SHOW STATUS;        # 显示服务器状态
SHOW VARIABLES;     # 显示系统变量
SOURCE file.sql;    # 执行 SQL 文件
\. file.sql         # 执行 SQL 文件（另一种方式）
exit                # 退出
quit                # 退出
\q                  # 退出
\c                  # 清除当前输入
\G                  # 垂直显示结果
\g                  # 执行命令（等同于 ;）
\s                  # 显示状态信息
\T                  # 开始记录日志
\t                  # 停止记录日志
\W                  # 显示警告
\w                  # 不显示警告
```

## 导入/导出

```sql
-- 导出查询结果到文件（需要 FILE 权限）
SELECT * FROM table_name 
INTO OUTFILE '/path/to/file.csv'
FIELDS TERMINATED BY ',' 
ENCLOSED BY '"'
LINES TERMINATED BY '\n';

-- 从文件导入数据（需要 FILE 权限）
LOAD DATA INFILE '/path/to/file.csv'
INTO TABLE table_name
FIELDS TERMINATED BY ','
ENCLOSED BY '"'
LINES TERMINATED BY '\n'
IGNORE 1 ROWS;  -- 忽略第一行（标题行）
```

```bash
# 使用 mysql 命令导出 CSV
mysql -u root -p -h 127.0.0.1 -P 3306 database_name -e "SELECT * FROM table_name" > output.csv

# 使用 mysql 命令导出 CSV（带表头）
mysql -u root -p -h 127.0.0.1 -P 3306 database_name -e "SELECT * FROM table_name" | sed 's/\t/,/g' > output.csv

# 使用 mysql 命令导入 CSV
mysql -u root -p -h 127.0.0.1 -P 3306 database_name -e "LOAD DATA LOCAL INFILE '/path/to/file.csv' INTO TABLE table_name FIELDS TERMINATED BY ',' LINES TERMINATED BY '\n' IGNORE 1 ROWS"
```

## 日志管理

```sql
-- 查看日志相关配置
SHOW VARIABLES LIKE 'log%';

-- 查看二进制日志
SHOW BINARY LOGS;
SHOW MASTER STATUS;

-- 查看错误日志位置
SHOW VARIABLES LIKE 'log_error';

-- 查看慢查询日志配置
SHOW VARIABLES LIKE 'slow_query%';
SHOW VARIABLES LIKE 'long_query_time';

-- 启用慢查询日志
SET GLOBAL slow_query_log = 'ON';
SET GLOBAL long_query_time = 2;  -- 设置慢查询阈值（秒）

-- 查看通用查询日志
SHOW VARIABLES LIKE 'general_log%';
SET GLOBAL general_log = 'ON';
```

## 主从复制

```sql
-- 查看主从状态
SHOW MASTER STATUS;
SHOW SLAVE STATUS;  -- MySQL 8.0+ 使用 SHOW REPLICA STATUS

-- 查看从库状态
SHOW REPLICA STATUS;  -- MySQL 8.0+
SHOW SLAVE STATUS\G;  -- MySQL 5.7

-- 停止从库复制
STOP SLAVE;  -- MySQL 5.7
STOP REPLICA;  -- MySQL 8.0+

-- 启动从库复制
START SLAVE;  -- MySQL 5.7
START REPLICA;  -- MySQL 8.0+

-- 跳过错误（谨慎使用）
SET GLOBAL sql_slave_skip_counter = 1;  -- MySQL 5.7
SET GLOBAL sql_replica_skip_counter = 1;  -- MySQL 8.0+
```
