package admin

import (
	"encoding/json"
	"fmt"
	appfacades "goravel/app/facades"
	nethttp "net/http"
	"time"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"

	"goravel/app/http/response"
	"goravel/app/services"
)

type DashboardController struct{}

func NewDashboardController() *DashboardController {
	return &DashboardController{}
}

func (r *DashboardController) dashboardService(ctx http.Context) services.DashboardService {
	return services.NewDashboardService(ctx)
}

// GetCount 获取统计数据
func (r *DashboardController) GetCount(ctx http.Context) http.Response {
	return response.Success(ctx, "get_success", r.dashboardService(ctx).GetCountSnapshot())
}

// GetUserAccessSource 获取用户访问来源数据（根据 UserAgent 判断设备类型）
func (r *DashboardController) GetUserAccessSource(ctx http.Context) http.Response {
	return response.Success(ctx, "get_success", r.dashboardService(ctx).GetUserAccessSource())
}

// GetWeeklyUserActivity 获取每周用户活跃量（从操作日志统计）
func (r *DashboardController) GetWeeklyUserActivity(ctx http.Context) http.Response {
	return response.Success(ctx, "get_success", r.dashboardService(ctx).GetWeeklyUserActivity())
}

// GetMonthlySales 获取每月操作统计（替换销售额数据）
func (r *DashboardController) GetMonthlySales(ctx http.Context) http.Response {
	return response.Success(ctx, "get_success", r.dashboardService(ctx).GetMonthlyOperations())
}

// GetRecentActivities 获取最近活动
func (r *DashboardController) GetRecentActivities(ctx http.Context) http.Response {
	return response.Success(ctx, "get_success", r.dashboardService(ctx).GetRecentActivities())
}

// StreamDashboardData SSE 实时推送 Dashboard 数据
func (r *DashboardController) StreamDashboardData(ctx http.Context) http.Response {
	interval := 5
	if intervalStr := ctx.Request().Query("interval", ""); intervalStr != "" {
		if parsed, err := time.ParseDuration(intervalStr + "s"); err == nil {
			interval = int(parsed.Seconds())
			if interval < 2 {
				interval = 2
			}
			if interval > 60 {
				interval = 60
			}
		}
	}

	writer := ctx.Response().Writer()
	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("Connection", "keep-alive")
	writer.Header().Set("X-Accel-Buffering", "no")

	initMsg := map[string]any{
		"type":     "connected",
		"message":  "SSE连接已建立，开始推送 Dashboard 数据",
		"interval": interval,
	}
	initData, _ := json.Marshal(initMsg)
	fmt.Fprintf(writer, "data: %s\n\n", string(initData))
	if flusher, ok := writer.(nethttp.Flusher); ok {
		flusher.Flush()
	}

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	clientGone := ctx.Request().Origin().Context().Done()
	appDone := appfacades.App().Context().Done()
	svc := r.dashboardService(ctx)

	for {
		select {
		case <-clientGone:
			return nil
		case <-appDone:
			return nil
		case <-ticker.C:
			message := map[string]any{
				"type":      "dashboard_data",
				"data":      svc.CollectDashboardData(),
				"timestamp": time.Now().Format(time.RFC3339),
			}

			messageData, err := json.Marshal(message)
			if err != nil {
				facades.Log().Errorf("Dashboard SSE: failed to marshal data: %v", err)
				continue
			}

			fmt.Fprintf(writer, "data: %s\n\n", string(messageData))
			if flusher, ok := writer.(nethttp.Flusher); ok {
				flusher.Flush()
			}
		}
	}
}
