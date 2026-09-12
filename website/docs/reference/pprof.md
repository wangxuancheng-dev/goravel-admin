# 生产环境使用 pprof 性能分析

## 概述

pprof 是 Go 语言内置的性能分析工具，可以帮助你诊断生产环境中的性能问题。本指南介绍如何在生产环境中安全地使用 pprof。

## 安全配置

### 1. 启用 pprof

在 `.env` 文件中添加以下配置：

```env
# 启用 pprof（生产环境需要显式设置）
PPROF_ENABLED=true

# IP 白名单（推荐使用）
# 支持单个 IP 和 CIDR 格式，多个 IP 用逗号分隔
PPROF_ALLOWED_IPS=127.0.0.1,192.168.1.100,10.0.0.0/8

# 访问 token（可选，但强烈推荐）
# 如果设置，需要在请求头或查询参数中提供
PPROF_TOKEN=your-secret-token-here
```

### 2. 安全建议

**强烈推荐同时使用 IP 白名单和 token：**
- IP 白名单：限制只有特定 IP 可以访问
- Token 认证：即使 IP 泄露，也需要 token 才能访问

**不推荐在生产环境不设置任何限制：**
- 如果 `PPROF_ALLOWED_IPS` 为空且 `PPROF_TOKEN` 为空，任何人都可以访问性能数据

## 使用方法

### 方法一：使用 go tool pprof（推荐）

即使生产服务器没有 Go 环境，你也可以在本地机器上使用 `go tool pprof` 连接到远程服务器。

#### 1. 安装 Go 工具链

在你的本地机器上安装 Go（如果还没有安装）：
- 下载地址：https://golang.org/dl/
- 或者使用包管理器：`brew install go` (macOS) / `apt-get install golang-go` (Linux)

#### 2. 连接到远程服务器

```bash
# CPU 性能分析（30秒采样）
go tool pprof http://your-server:3008/debug/pprof/profile?token=your-secret-token-here

# 堆内存分析
go tool pprof http://your-server:3008/debug/pprof/heap?token=your-secret-token-here

# 协程分析
go tool pprof http://your-server:3008/debug/pprof/goroutine?token=your-secret-token-here

# 线程创建分析
go tool pprof http://your-server:3008/debug/pprof/threadcreate?token=your-secret-token-here
```

#### 3. 使用请求头传递 token

```bash
# 使用 curl 设置请求头
curl -H "X-Pprof-Token: your-secret-token-here" \
  http://your-server:3008/debug/pprof/heap > heap.prof

# 然后使用 go tool pprof 分析
go tool pprof heap.prof
```

### 方法二：导出数据文件

如果无法直接连接，可以先导出数据文件，然后在有 Go 环境的机器上分析。

#### 1. 导出 pprof 数据

```bash
# 使用 curl 导出数据（需要 token）
curl -H "X-Pprof-Token: your-secret-token-here" \
  http://your-server:3008/debug/pprof/heap > heap.prof

curl -H "X-Pprof-Token: your-secret-token-here" \
  http://your-server:3008/debug/pprof/profile?seconds=30 > cpu.prof
```

#### 2. 传输到本地机器

```bash
# 使用 scp 传输文件
scp user@your-server:/path/to/heap.prof ./
```

#### 3. 在本地分析

```bash
# 分析堆内存
go tool pprof heap.prof

# 分析 CPU
go tool pprof cpu.prof
```

### 方法三：使用 Web 界面

直接在浏览器中访问（需要 token）：

```
http://your-server:3008/debug/pprof/?token=your-secret-token-here
```

或者使用请求头：

```bash
# 使用 curl 访问主页
curl -H "X-Pprof-Token: your-secret-token-here" \
  http://your-server:3008/debug/pprof/
```

## 常用分析命令

### CPU 性能分析

```bash
# 30秒 CPU 采样
go tool pprof http://your-server:3008/debug/pprof/profile?token=your-token

# 交互式命令
(pprof) top10          # 查看占用 CPU 最多的 10 个函数
(pprof) list 函数名      # 查看函数详细代码
(pprof) web            # 生成 SVG 图表（需要安装 graphviz）
(pprof) png            # 生成 PNG 图表
```

### 内存分析

```bash
# 堆内存分析
go tool pprof http://your-server:3008/debug/pprof/heap?token=your-token

# 交互式命令
(pprof) top10          # 查看占用内存最多的 10 个函数
(pprof) alloc_space    # 查看累计分配的内存
(pprof) inuse_space    # 查看当前使用的内存
```

### 协程分析

```bash
# 协程分析（排查 goroutine 泄漏）
go tool pprof http://your-server:3008/debug/pprof/goroutine?token=your-token

# 交互式命令
(pprof) top10          # 查看协程数量最多的 10 个函数
```

## 常见问题排查

### 1. CPU 使用率过高

```bash
# 分析 CPU profile
go tool pprof http://your-server:3008/debug/pprof/profile?token=your-token

# 查看 top 函数
(pprof) top20
```

### 2. 内存泄漏

```bash
# 分析堆内存
go tool pprof http://your-server:3008/debug/pprof/heap?token=your-token

# 查看内存分配
(pprof) top20 -cum
```

### 3. 协程泄漏

```bash
# 分析协程
go tool pprof http://your-server:3008/debug/pprof/goroutine?token=your-token

# 查看协程堆栈
(pprof) top20
```

### 4. 阻塞问题

```bash
# 分析阻塞
go tool pprof http://your-server:3008/debug/pprof/block?token=your-token
```

## 注意事项

1. **性能影响**：pprof 采样会对性能产生一定影响，建议在排查问题时临时启用
2. **数据安全**：pprof 数据可能包含敏感信息，务必使用 IP 白名单和 token 保护
3. **网络访问**：确保防火墙规则允许访问 pprof 端口
4. **定期清理**：排查完成后，建议关闭 pprof 或加强安全限制

## 故障排查

### 404 错误

- 检查 `PPROF_ENABLED=true` 是否设置
- 检查应用是否重启以加载新配置

### 403 错误（IP 不允许）

- 检查 `PPROF_ALLOWED_IPS` 是否包含你的 IP
- 如果使用代理，检查真实 IP 获取是否正确

### 401 错误（Token 无效）

- 检查 `PPROF_TOKEN` 是否设置
- 检查请求头或查询参数中的 token 是否正确

## 参考资源

- [Go pprof 官方文档](https://pkg.go.dev/net/http/pprof)
- [Go 性能分析指南](https://github.com/golang/go/wiki/Performance)

