package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"goravel/app/clients"
	"goravel/app/facades"
	"goravel/app/websocket/notifications"
	"goravel/bootstrap"
)

// @title           Goravel Admin API
// @version         1.0
// @description     这是一个基于 Goravel 框架的后台管理系统 API 文档
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:3000
// @BasePath  /api/admin

// @securityDefinitions.apikey BearerAuth
// @in                         header
// @name                       Authorization
// @description                JWT 认证，格式：Bearer {token}

// shutdownTimeout 优雅关闭最长等待时间。
// 监控 SSE / Dashboard SSE 等长连接可能拖住 HTTP Shutdown，超时后强制退出。
const shutdownTimeout = 10 * time.Second

func main() {
	// Bootstrap the application
	app := bootstrap.Boot()

	// 须在 Start 之前注册信号：Start() 会阻塞直到所有 runner 退出，
	// 若放在 Start 之后则 Ctrl+C 永远无法触发自定义的强制退出逻辑。
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		app.Start()
	}()

	sig := <-quit
	facades.Log().Infof("Received signal %v, shutting down...", sig)
	shutdownApplication(app)
}

// shutdownApplication 优雅关闭应用程序（框架会依次关闭所有 runners：Route、Queue 等）
func shutdownApplication(app interface{ Shutdown() error }) {
	done := make(chan struct{})
	go func() {
		defer close(done)

		// 停止 NotificationHub（关闭所有 WebSocket 连接）
		notifications.Hub().Stop()
		facades.Log().Info("NotificationHub stopped")

		// Stop the WS Redis bridge before closing Redis clients to avoid a spurious warning.
		notifications.StopRedisBridge()

		// 关闭所有 Redis 客户端
		if err := clients.CloseAllRedisClients(); err != nil {
			facades.Log().Errorf("Close Redis clients error: %v", err)
		} else {
			facades.Log().Info("Redis clients closed")
		}

		// 关闭应用程序（框架会依次关闭所有 runners：Route、Queue 等）
		if err := app.Shutdown(); err != nil {
			facades.Log().Errorf("Application Shutdown error: %v", err)
			os.Exit(1)
		}
	}()

	select {
	case <-done:
		facades.Log().Info("Application stopped")
	case <-time.After(shutdownTimeout):
		facades.Log().Warningf("Graceful shutdown timed out after %s, forcing exit", shutdownTimeout)
		os.Exit(1)
	}
}
