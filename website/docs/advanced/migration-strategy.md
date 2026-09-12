# 数据库迁移策略

## 问题背景

在容器化部署中，如果应用在数据库迁移完成之前启动，新版本的代码可能会尝试访问不存在的数据库字段或表，导致运行时错误。

## 解决方案

### 启动脚本自动迁移

我们使用 Docker 启动脚本 (`docker-entrypoint.sh`) 在应用启动前自动执行数据库迁移：

```
容器启动 → 执行迁移 → 迁移成功 → 启动应用
                ↓
            迁移失败 → 容器退出 → 部署脚本检测到并回滚
```

### 工作流程

1. **容器启动时**：`docker-entrypoint.sh` 脚本首先执行
2. **执行迁移**：运行 `artisan migrate` 命令
3. **迁移成功**：继续启动应用
4. **迁移失败**：容器退出（exit 1），部署脚本检测到并回滚

### 优势

- ✅ **避免字段不存在错误** - 迁移在应用启动前完成
- ✅ **自动执行** - 无需手动操作
- ✅ **失败自动回滚** - 迁移失败时容器退出，部署脚本自动回滚
- ✅ **幂等操作** - Goravel 的 migrate 是安全的，可以重复执行

### 配置选项

#### 跳过迁移（不推荐）

如果需要在某些情况下跳过迁移：

```bash
# 在 docker-compose.yml 中设置环境变量
environment:
  - SKIP_MIGRATE=true
```

**注意：** 跳过迁移可能导致应用启动后访问不存在的字段而报错。

### 部署脚本中的处理

部署脚本 (`docker-blue-green.sh`) 会：

1. 启动新版本容器
2. 等待容器启动（包括迁移完成）
3. 验证应用是否已启动（如果迁移失败，应用不会启动）
4. 如果容器退出，自动检测并回滚

### 健康检查

健康检查的 `start_period` 设置为 60 秒，给迁移足够的时间完成：

```yaml
healthcheck:
  start_period: 60s  # 等待迁移和应用启动
```

### 单独执行迁移

如果需要单独执行迁移（不重启容器）：

```bash
# 在运行中的容器中执行迁移
./scripts/deploy/migrate.sh
```

### 故障排查

如果迁移失败：

1. **查看容器日志**：
   ```bash
   docker logs goravel-admin-blue
   ```

2. **检查迁移状态**：
   ```bash
   docker exec goravel-admin-blue /www/main artisan migrate:status
   ```

3. **手动执行迁移**：
   ```bash
   docker exec goravel-admin-blue /www/main artisan migrate
   ```

### 最佳实践

1. **迁移应该是幂等的** - 确保迁移可以安全地重复执行
2. **测试迁移** - 在测试环境先测试迁移
3. **备份数据库** - 在生产环境执行迁移前备份数据库
4. **监控迁移** - 关注容器日志，确保迁移成功

### 相关文件

- `docker-entrypoint.sh` - 容器启动脚本
- `Dockerfile` - Docker 镜像构建文件
- `scripts/deploy/docker-blue-green.sh` - 部署脚本
- `scripts/deploy/migrate.sh` - 单独执行迁移脚本

