package services

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"

	"github.com/dromara/carbon/v2"
	"github.com/spf13/cast"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/models"
)

type ConfigService interface {
	GetByGroup(group string) ([]models.Config, error)
	Save(group string, configsMap map[string]any) error
	TestEmail(params TestEmailParams, recipientEmail string) error
}

type TestEmailParams struct {
	Host       string
	Port       int
	Username   string
	Password   string
	From       string
	FromName   string
	Encryption string
}

type ConfigServiceImpl struct {
	ctx context.Context
}

func NewConfigService(ctx context.Context) ConfigService {
	return &ConfigServiceImpl{ctx: ctx}
}

// sensitiveConfigKeys maps config group → keys that must be blanked on read and
// skipped on save when the incoming value is empty (preserve stored secret).
var sensitiveConfigKeys = map[string]map[string]struct{}{
	"email": {
		"email_password": {},
	},
	"payment": {
		"api_key":      {},
		"api_secret":   {},
		"private_key":  {},
		"secret":       {},
		"secret_key":   {},
		"mch_secret":   {},
		"app_secret":   {},
		"notify_secret": {},
	},
}

func isSensitiveConfigKey(group, key string) bool {
	keys, ok := sensitiveConfigKeys[group]
	if !ok {
		return false
	}
	_, ok = keys[key]
	return ok
}

// GetByGroup 根据分组获取配置（敏感字段读出时置空）
func (s *ConfigServiceImpl) GetByGroup(group string) ([]models.Config, error) {
	if group == "" {
		return nil, apperrors.ErrConfigGroupRequired
	}

	var configs []models.Config
	// 查询配置，即使没有数据也返回空数组，不返回错误
	_ = appfacades.OrmQuery(s.ctx).Where("group", group).Order("sort asc, id asc").Get(&configs)

	for i := range configs {
		if isSensitiveConfigKey(group, configs[i].Key) {
			configs[i].Value = ""
		}
	}

	return configs, nil
}

// Save 按分组批量保存配置
func (s *ConfigServiceImpl) Save(group string, configsMap map[string]any) error {
	if group == "" {
		return apperrors.ErrConfigGroupRequired
	}
	if len(configsMap) == 0 {
		return apperrors.ErrConfigsRequired
	}

	if group == "captcha" {
		if raw, ok := configsMap["captcha_expire"]; ok {
			expire := cast.ToInt(raw)
			if expire < 30 {
				return apperrors.ErrCaptchaExpireMin.WithParams(map[string]any{
					"seconds": 30,
				})
			}
		}
	}

	var existingConfigs []models.Config
	_ = appfacades.OrmQuery(s.ctx).Where("group", group).Get(&existingConfigs)

	configMap := make(map[string]*models.Config)
	for i := range existingConfigs {
		configMap[existingConfigs[i].Key] = &existingConfigs[i]
	}

	now := carbon.Now()

	// storage 分组仅允许保存驱动选择字段（白名单）
	if group == "storage" {
		allowedKeys := map[string]bool{
			"file_disk":     true,
			"storage_disk":  true, // 向后兼容，保留但不推荐使用
			"export_disk":   true, // 向后兼容
			"export_format": true,
		}
		filteredConfigs := make(map[string]any)
		for key, value := range configsMap {
			if allowedKeys[key] {
				filteredConfigs[key] = value
			}
		}
		configsMap = filteredConfigs
	}

	for key, value := range configsMap {
		var valueStr string
		switch v := value.(type) {
		case bool:
			if v {
				valueStr = "1"
			} else {
				valueStr = "0"
			}
		case nil:
			valueStr = ""
		default:
			valueStr = cast.ToString(value)
		}

		// 敏感字段为空且已存在时跳过更新，保留原值
		if isSensitiveConfigKey(group, key) && valueStr == "" {
			if _, exists := configMap[key]; exists {
				continue
			}
		}

		if config, exists := configMap[key]; exists {
			config.Value = valueStr
			if err := appfacades.OrmQuery(s.ctx).Save(config); err != nil {
				return err
			}
		} else {
			configData := map[string]any{
				"group":      group,
				"key":        key,
				"value":      valueStr,
				"type":       "input",
				"sort":       0,
				"created_at": now,
				"updated_at": now,
			}
			if err := appfacades.OrmQuery(s.ctx).Table("configs").Create(configData); err != nil {
				return err
			}
		}
	}

	return nil
}

// TestEmail 使用给定 SMTP 参数向收件人发送测试邮件
func (s *ConfigServiceImpl) TestEmail(params TestEmailParams, recipientEmail string) error {
	fromName := params.FromName
	if fromName == "" {
		fromName = params.From
	}
	encryption := params.Encryption
	if encryption == "" {
		encryption = "tls"
	}

	subject := "测试邮件"
	body := fmt.Sprintf(`<h2>这是一封测试邮件</h2>
<p>如果您收到这封邮件，说明邮件配置正确。</p>
<p>发送时间：%s</p>
<p>SMTP服务器：%s:%d</p>
<p>加密方式：%s</p>`, carbon.Now().ToDateTimeString(), params.Host, params.Port, encryption)

	message := fmt.Sprintf("From: %s <%s>\r\n", fromName, params.From)
	message += fmt.Sprintf("To: %s\r\n", recipientEmail)
	message += fmt.Sprintf("Subject: %s\r\n", subject)
	message += "MIME-Version: 1.0\r\n"
	message += "Content-Type: text/html; charset=UTF-8\r\n"
	message += "\r\n" + body

	addr := fmt.Sprintf("%s:%d", params.Host, params.Port)
	auth := smtp.PlainAuth("", params.Username, params.Password, params.Host)

	var err error
	if encryption == "ssl" {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: false,
			ServerName:         params.Host,
		}
		conn, connErr := tls.Dial("tcp", addr, tlsConfig)
		if connErr != nil {
			return connErr
		}
		defer conn.Close()

		client, clientErr := smtp.NewClient(conn, params.Host)
		if clientErr != nil {
			return clientErr
		}
		defer client.Close()

		if err = client.Auth(auth); err != nil {
			return err
		}
		if err = client.Mail(params.From); err != nil {
			return err
		}
		if err = client.Rcpt(recipientEmail); err != nil {
			return err
		}

		writer, writerErr := client.Data()
		if writerErr != nil {
			return writerErr
		}
		if _, err = writer.Write([]byte(message)); err != nil {
			writer.Close()
			return err
		}
		return writer.Close()
	}

	if encryption == "tls" {
		err = smtp.SendMail(addr, auth, params.From, []string{recipientEmail}, []byte(message))
		if err == nil {
			return nil
		}

		// 直接 SendMail 失败时尝试手动 TLS
		tlsConfig := &tls.Config{
			InsecureSkipVerify: false,
			ServerName:         params.Host,
		}
		conn, connErr := smtp.Dial(addr)
		if connErr != nil {
			return connErr
		}
		defer conn.Close()

		if err = conn.StartTLS(tlsConfig); err != nil {
			return err
		}
		if err = conn.Auth(auth); err != nil {
			return err
		}
		if err = conn.Mail(params.From); err != nil {
			return err
		}
		if err = conn.Rcpt(recipientEmail); err != nil {
			return err
		}

		writer, writerErr := conn.Data()
		if writerErr != nil {
			return writerErr
		}
		if _, err = writer.Write([]byte(message)); err != nil {
			writer.Close()
			return err
		}
		return writer.Close()
	}

	// 普通连接（无加密）
	return smtp.SendMail(addr, auth, params.From, []string{recipientEmail}, []byte(message))
}
