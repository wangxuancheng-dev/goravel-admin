package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/goravel/framework/facades"

	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/utils"
)

// LoginAnomalyAlert 异地登录告警内容（通知已创建；邮件由调用方可选发送）。
type LoginAnomalyAlert struct {
	Title   string
	Content string
	Email   string
}

// LoginAnomalyService 登录异常（换 IP）检测与告警。
type LoginAnomalyService interface {
	// CheckAndAlert 在记录本次成功登录前调用：若上次成功登录 IP 与当前不同则站内通知，并返回邮件告警信息。
	CheckAndAlert(admin models.Admin, currentIP string) (*LoginAnomalyAlert, error)
}

type LoginAnomalyServiceImpl struct {
	ctx context.Context
}

func NewLoginAnomalyService(ctx context.Context) LoginAnomalyService {
	return &LoginAnomalyServiceImpl{ctx: ctx}
}

func (s *LoginAnomalyServiceImpl) CheckAndAlert(admin models.Admin, currentIP string) (*LoginAnomalyAlert, error) {
	enabled := utils.GetConfigValueBool(s.ctx, "login_security", "anomaly_alert_enabled",
		facades.Config().GetBool("login_security.anomaly_alert_enabled", true))
	if !enabled {
		return nil, nil
	}
	currentIP = strings.TrimSpace(currentIP)
	if currentIP == "" || admin.ID == 0 {
		return nil, nil
	}

	var last models.LoginLog
	err := appfacades.OrmQuery(s.ctx).
		Model(&models.LoginLog{}).
		Where("admin_id", admin.ID).
		Where("status", 1).
		Where("message", "login_success").
		OrderBy("id", "desc").
		First(&last)
	if err != nil {
		// 无历史成功登录：不告警
		return nil, nil
	}

	lastIP := strings.TrimSpace(last.IP)
	if lastIP == "" || lastIP == currentIP {
		return nil, nil
	}

	title := "登录异常提醒"
	content := fmt.Sprintf("检测到账号 %s 从新 IP 登录。上次 IP：%s，本次 IP：%s。若非本人操作，请立即修改密码并检查安全设置。",
		admin.Username, lastIP, currentIP)

	receiverID := admin.ID
	notifSvc := NewNotificationServiceImpl(s.ctx)
	if _, err := notifSvc.Create(title, content, "login_anomaly", nil, &receiverID); err != nil {
		facades.Log().Errorf("login anomaly notification failed: admin_id=%d err=%v", admin.ID, err)
	}

	alert := &LoginAnomalyAlert{
		Title:   title,
		Content: content,
		Email:   strings.TrimSpace(admin.Email),
	}
	return alert, nil
}
