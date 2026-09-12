# PostgreSQL 常用命令

## 连接数据库

```bash
# 查看版本
psql -V

# 连接数据库
psql -U postgres -h 127.0.0.1 -p 5432

# 连接指定数据库
psql -U postgres -d database_name -h 127.0.0.1 -p 5432

# 使用密码文件连接
psql -U postgres -h 127.0.0.1 -p 5432 -d database_name -f query.sql

# 退出交互界面
\q
```

## 宝塔 aapanel 安装扩展
```bash
ss -tlnp | grep postgres

cd /usr/local/pgsql
make -C contrib -j"$(nproc)" PG_CONFIG=/www/server/pgsql/bin/pg_config
make -C contrib install PG_CONFIG=/www/server/pgsql/bin/pg_config

/www/server/pgsql/bin/psql -U postgres -d postgres -c "SELECT name FROM pg_available_extensions ORDER BY name;"

sudo su - postgres
/www/server/pgsql/bin/pg_ctl restart -D /www/server/pgsql/data

# 重新创建扩展并验证
/www/server/pgsql/bin/psql -U postgres -d database_name -p 5432 -c "CREATE EXTENSION IF NOT EXISTS citext;"
/www/server/pgsql/bin/psql -U postgres -d database_name -p 5432 -c "\dx"

# 可选（最常用的扩展）
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS unaccent;
CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS hstore;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;
```

## 数据库管理

```sql
-- 列出所有数据库
\l
-- 或
SELECT datname FROM pg_database;

-- 创建数据库
CREATE DATABASE database_name;

-- 删除数据库
DROP DATABASE database_name;

-- 切换数据库
\c database_name

-- 查看当前数据库
SELECT current_database();

-- 查看数据库大小
SELECT pg_size_pretty(pg_database_size('database_name'));

-- 查看所有数据库大小
SELECT datname, pg_size_pretty(pg_database_size(datname)) AS size 
FROM pg_database 
ORDER BY pg_database_size(datname) DESC;
```

## 用户和权限管理

```sql
-- 列出所有用户
\du
-- 或
SELECT usename FROM pg_user;

-- 创建用户
CREATE USER username WITH PASSWORD 'password';

-- 创建超级用户
CREATE USER username WITH SUPERUSER PASSWORD 'password';

-- 修改用户密码
ALTER USER username WITH PASSWORD 'new_password';

-- 删除用户
DROP USER username;

-- 授予权限
GRANT ALL PRIVILEGES ON DATABASE database_name TO username;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO username;

-- 撤销权限
REVOKE ALL PRIVILEGES ON DATABASE database_name FROM username;

-- 查看用户权限
\du username
```

## 表管理

```sql
-- 列出所有表
\dt
-- 或
SELECT tablename FROM pg_tables WHERE schemaname = 'public';

-- 列出所有表（包括系统表）
\dt+

-- 查看表结构
\d table_name
\d+ table_name  -- 详细信息

-- 查看表大小
SELECT pg_size_pretty(pg_total_relation_size('table_name')) AS total_size,
       pg_size_pretty(pg_relation_size('table_name')) AS table_size,
       pg_size_pretty(pg_indexes_size('table_name')) AS indexes_size;

-- 查看所有表大小
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- 查看表行数
SELECT COUNT(*) FROM table_name;

-- 查看表统计信息
SELECT * FROM pg_stat_user_tables WHERE relname = 'table_name';
```

## 索引管理

```sql
-- 列出所有索引
\di
\di table_name*
-- 或
SELECT indexname FROM pg_indexes WHERE tablename = 'table_name';

-- 查看索引详细信息
\d+ index_name

-- 创建索引
CREATE INDEX idx_name ON table_name(column_name);

-- 创建唯一索引
CREATE UNIQUE INDEX idx_name ON table_name(column_name);

-- 创建复合索引
CREATE INDEX idx_name ON table_name(column1, column2);

-- 创建 GIN 索引（用于全文搜索、数组等）
CREATE INDEX idx_name ON table_name USING GIN(column_name gin_trgm_ops);

-- 创建 GIST 索引（用于几何数据、范围等）
CREATE INDEX idx_name ON table_name USING GIST(column_name);

-- 删除索引
DROP INDEX index_name;

-- 重建索引（解决索引碎片）
REINDEX INDEX index_name;
REINDEX TABLE table_name;
REINDEX DATABASE database_name;

-- 查看索引使用情况
SELECT 
    schemaname,
    tablename,
    indexname,
    idx_scan,
    idx_tup_read,
    idx_tup_fetch
FROM pg_stat_user_indexes
ORDER BY idx_scan DESC;
```

## 查询和分析

```sql
-- 查看执行计划
EXPLAIN SELECT * FROM table_name WHERE condition;
EXPLAIN ANALYZE SELECT * FROM table_name WHERE condition;
EXPLAIN (ANALYZE, BUFFERS, VERBOSE) SELECT * FROM table_name WHERE condition;

-- 查看慢查询（需要启用 pg_stat_statements）
SELECT 
    query,
    calls,
    total_exec_time,
    mean_exec_time,
    max_exec_time
FROM pg_stat_statements
ORDER BY total_exec_time DESC
LIMIT 10;

-- 查看当前正在执行的查询
SELECT pid, usename, application_name, state, query, query_start 
FROM pg_stat_activity 
WHERE state = 'active';

-- 查看等待锁的查询
SELECT 
    blocked_locks.pid AS blocked_pid,
    blocking_locks.pid AS blocking_pid,
    blocked_activity.usename AS blocked_user,
    blocking_activity.usename AS blocking_user,
    blocked_activity.query AS blocked_statement,
    blocking_activity.query AS blocking_statement
FROM pg_catalog.pg_locks blocked_locks
JOIN pg_catalog.pg_stat_activity blocked_activity ON blocked_activity.pid = blocked_locks.pid
JOIN pg_catalog.pg_locks blocking_locks ON blocking_locks.locktype = blocked_locks.locktype
JOIN pg_catalog.pg_stat_activity blocking_activity ON blocking_activity.pid = blocking_locks.pid
WHERE NOT blocked_locks.granted AND blocking_locks.granted;

-- 终止查询
SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE pid = <pid>;
```

## 扩展管理

```sql
-- 列出所有已安装的扩展
\dx
-- 或
SELECT extname, extversion FROM pg_extension;

-- 查看可用扩展
SELECT * FROM pg_available_extensions;

-- 安装扩展
CREATE EXTENSION extension_name;

-- 安装扩展（如果不存在）
CREATE EXTENSION IF NOT EXISTS extension_name;

-- 卸载扩展
DROP EXTENSION extension_name;

-- pg_trgm 扩展（用于相似度搜索）
-- 查看已安装的扩展，确认包含 pg_trgm
SELECT extname, extversion FROM pg_extension WHERE extname = 'pg_trgm';

-- 启用 pg_trgm 扩展（数据库级生效，超级用户权限）
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- 降低阈值（默认0.3），匹配更多相似结果（测试用）
SET pg_trgm.similarity_threshold = 0.2;

-- 执行相似匹配查询
SELECT * FROM test_trgm WHERE similarity(content, '三四五') > 0.2;

-- 重建GIN索引（定期执行，比如每月1次） 解决索引碎片
REINDEX INDEX idx_test_trgm_content;
```

## 备份和恢复

```bash
# 备份数据库（自定义格式，支持压缩）
pg_dump -U postgres -h 127.0.0.1 -d database_name -F c -f backup.dump

# 备份数据库（SQL格式）
pg_dump -U postgres -h 127.0.0.1 -d database_name -f backup.sql

# 备份单个表
pg_dump -U postgres -h 127.0.0.1 -d database_name -t table_name -f table_backup.sql

# 仅备份结构（不包含数据）
pg_dump -U postgres -h 127.0.0.1 -d database_name -s -f schema.sql

# 仅备份数据（不包含结构）
pg_dump -U postgres -h 127.0.0.1 -d database_name -a -f data.sql

# 恢复数据库（从自定义格式）
pg_restore -U postgres -h 127.0.0.1 -d database_name backup.dump

# 恢复数据库（从SQL文件）
psql -U postgres -h 127.0.0.1 -d database_name -f backup.sql

# 备份所有数据库
pg_dumpall -U postgres -h 127.0.0.1 -f all_databases.sql

# 恢复所有数据库
psql -U postgres -h 127.0.0.1 -f all_databases.sql
```

## 性能监控

```sql
-- 查看数据库连接数
SELECT count(*) FROM pg_stat_activity;

-- 查看最大连接数
SHOW max_connections;

-- 查看当前配置
SHOW ALL;

-- 查看共享缓冲区使用情况
SELECT * FROM pg_stat_bgwriter;

-- 查看表统计信息
SELECT 
    schemaname,
    tablename,
    n_tup_ins AS inserts,
    n_tup_upd AS updates,
    n_tup_del AS deletes,
    n_live_tup AS live_rows,
    n_dead_tup AS dead_rows,
    last_vacuum,
    last_autovacuum,
    last_analyze,
    last_autoanalyze
FROM pg_stat_user_tables
ORDER BY n_live_tup DESC;

-- 查看缓存命中率
SELECT 
    sum(heap_blks_read) as heap_read,
    sum(heap_blks_hit)  as heap_hit,
    (sum(heap_blks_hit) - sum(heap_blks_read)) / sum(heap_blks_hit) as ratio
FROM pg_statio_user_tables;
```

## 维护操作

```sql
-- 分析表（更新统计信息）
ANALYZE table_name;
ANALYZE;  -- 分析所有表

-- 清理表（VACUUM）
VACUUM table_name;
VACUUM FULL table_name;  -- 完全清理，会锁表
VACUUM ANALYZE table_name;  -- 清理并分析

-- 自动清理（后台进程）
-- 查看自动清理配置
SHOW autovacuum;

-- 查看需要清理的表
SELECT 
    schemaname,
    tablename,
    n_dead_tup,
    n_live_tup,
    round(n_dead_tup * 100.0 / NULLIF(n_live_tup, 0), 2) AS dead_ratio
FROM pg_stat_user_tables
WHERE n_dead_tup > 0
ORDER BY dead_ratio DESC;
```

## 实用命令

```sql
-- 查看当前时间
SELECT NOW();

-- 查看时区
SHOW timezone;
SELECT current_setting('timezone');

-- 设置时区
SET timezone = 'Asia/Shanghai';

-- 查看当前用户
SELECT current_user;

-- 查看当前数据库
SELECT current_database();

-- 查看 PostgreSQL 版本
SELECT version();

-- 查看服务器配置
SHOW config_file;  -- 配置文件路径
SHOW data_directory;  -- 数据目录

-- 复制表结构
CREATE TABLE new_table (LIKE old_table INCLUDING ALL);

-- 复制表数据
INSERT INTO new_table SELECT * FROM old_table;

-- 查看序列
\ds
SELECT sequence_name FROM information_schema.sequences;

-- 重置序列
ALTER SEQUENCE sequence_name RESTART WITH 1;
```

## 安装扩展（Ubuntu/Debian）

```bash
# 替换版本号（如 14, 15, 16）为你的 PostgreSQL 版本
apt-get update
apt-get install -y postgresql-contrib-16

# 安装其他常用扩展
apt-get install -y postgresql-16-pg-stat-statements
```

## 常用 psql 命令

```bash
# 在 psql 交互界面中
\l          # 列出所有数据库
\c dbname   # 连接到数据库
\dt         # 列出当前数据库的所有表
\d table    # 显示表结构
\di         # 列出索引
\du         # 列出用户
\dx         # 列出扩展
\df         # 列出函数
\dn         # 列出模式
\q          # 退出
\?          # 帮助
\h          # SQL 命令帮助
\timing     # 显示查询执行时间
\x          # 扩展显示模式（列转行）
\copy       # 导入/导出 CSV
```

## 导入/导出 CSV

```sql
-- 导出表到 CSV
\copy table_name TO '/path/to/file.csv' WITH CSV HEADER;

-- 从 CSV 导入
\copy table_name FROM '/path/to/file.csv' WITH CSV HEADER;

-- 使用 COPY 命令（需要超级用户权限）
COPY table_name TO '/path/to/file.csv' WITH CSV HEADER;
COPY table_name FROM '/path/to/file.csv' WITH CSV HEADER;
```
