package routes

import (
	"github.com/goravel/framework/contracts/route"
	httpmiddleware "github.com/goravel/framework/http/middleware"

	"goravel/app/facades"
	"goravel/app/http/controllers/admin"
	"goravel/app/http/controllers/api"
	"goravel/app/http/middleware"
)

func Api() {
	authController := api.NewAuthController()
	orderSearchController := api.NewOrderController()
	queueTestController := api.NewQueueTestController()
	paymentNotifyController := api.NewPaymentNotifyController()
	attachmentController := admin.NewAttachmentController()
	publicConfigController := api.NewPublicConfigController()

	// 公开附件（C 端文章/站点可直接引用，无需 admin 前缀）
	facades.Route().Prefix("api").Middleware(middleware.Lang(), middleware.Tenant(), middleware.Blacklist()).Group(func(router route.Router) {
		router.Get("public/files/{id}", attachmentController.PublicPreview)
		router.Get("public/customer-service", publicConfigController.CustomerService)
	})

	// 支付回调：通用 {type}；租户从路径末段绑定。新渠道 RegisterPaymentGateway 后无需加路由。
	facades.Route().Prefix("api").Middleware(middleware.Lang(), middleware.Blacklist()).Group(func(router route.Router) {
		router.Post("payment/notify/{type}/{tenant}", paymentNotifyController.Notify)
		router.Post("payment/notify/{type}", paymentNotifyController.NotifyLegacy)
	})

	// C端用户路由组：统一前缀
	facades.Route().Prefix("api/user").Middleware(middleware.Tenant(), middleware.Blacklist()).Group(func(router route.Router) {

		// 登录注册相关（不需要认证，但需要限流）
		router.Middleware(middleware.Lang(), httpmiddleware.Throttle("login")).Group(func(router route.Router) {
			router.Post("register", authController.Register)
			router.Post("login", authController.Login)
		})

		// 需要认证的路由
		router.Middleware(middleware.UserJwt()).Group(func(router route.Router) {
			// 用户信息
			router.Get("info", authController.Info)
			// 登出
			router.Post("logout", authController.Logout)
			// 关键词搜「我的订单」（需 SEARCH_ENABLED=true，索引需已有数据）
			router.Get("orders/search", orderSearchController.SearchMyOrders)
		})

	})

	// 队列驱动测试路由（开发调试使用，仅允许开发环境）
	facades.Route().Prefix("api/queue-test").Middleware(middleware.DevelopmentOnly()).Group(func(router route.Router) {
		// 以下高噪音/异常链路测试接口先注释，避免开发环境持续写失败日志与 failed_jobs。
		// router.Post("all-in-one", queueTestController.AllInOne)
		// router.Post("all-special", queueTestController.AllSpecial)
		// router.Post("reclaim", queueTestController.Reclaim)
		// 默认队列立即投递。
		router.Post("dispatch", queueTestController.Dispatch)
		// 默认队列延迟投递（rabbitmq 需 delayed-message 插件才能严格延迟）。
		router.Post("delay", queueTestController.Delay)
		// 投递到 long-running 逻辑队列。
		router.Post("long-running", queueTestController.LongRunning)
		// router.Post("fail", queueTestController.Fail)
		// router.Post("backoff", queueTestController.Backoff)
		// 唯一队列测试：窗口期内同 key 只投递一次（默认 30s）。
		router.Post("unique", queueTestController.Unique)
		// 唯一队列状态：查看某个 key 是否在窗口期内。
		router.Get("unique/status", queueTestController.UniqueStatus)
		// 查看各测试 Job 的执行结果缓存。
		router.Get("result", queueTestController.Result)
		// 清空测试结果缓存，便于下一轮测试对比。
		router.Post("reset", queueTestController.Reset)
	})
}
