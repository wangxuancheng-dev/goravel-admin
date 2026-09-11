package jobs

import (
	"time"

	"github.com/goravel/framework/contracts/mail"
	"github.com/goravel/framework/facades"

	"goravel/app/errors"
	"goravel/app/utils"
)

// SendEmailArgs 发送邮件任务的参数结构体
type SendEmailArgs struct {
	To      string `validate:"required,email"`
	Subject string `validate:"required"`
	Content string
}

// SendEmail 发送邮件任务（支持递增延迟重试）
type SendEmail struct {
}

func (r *SendEmail) Signature() string {
	return "send_email"
}

// Handle 处理发送邮件任务
//
// 参数:
//   - args[0]: SendEmailArgs 结构体或 map[string]any
//
// 返回:
//   - error: 错误信息
func (r *SendEmail) Handle(args ...any) error {
	if len(args) < 1 {
		return errors.ErrInvalidArgument.WithMessage("missing email arguments")
	}

	// 解析参数
	var emailArgs SendEmailArgs
	switch v := args[0].(type) {
	case SendEmailArgs:
		emailArgs = v
	case map[string]any:
		// 从 map 转换（使用泛型辅助函数）
		if to, ok := utils.GetString(v, "to"); ok {
			emailArgs.To = to
		}
		if subject, ok := utils.GetString(v, "subject"); ok {
			emailArgs.Subject = subject
		}
		if content, ok := utils.GetString(v, "content"); ok {
			emailArgs.Content = content
		}
	default:
		// 兼容旧版本：尝试按位置解析
		if len(args) >= 2 {
			to, ok := args[0].(string)
			if !ok || to == "" {
				return errors.ErrInvalidArgument.WithMessage("invalid or empty email address")
			}
			emailArgs.To = to

			subject, ok := args[1].(string)
			if !ok || subject == "" {
				return errors.ErrInvalidArgument.WithMessage("invalid or empty subject")
			}
			emailArgs.Subject = subject

			if len(args) >= 3 {
				if content, ok := args[2].(string); ok {
					emailArgs.Content = content
				}
			}
		} else {
			return errors.ErrInvalidArgument.WithMessage("insufficient arguments")
		}
	}

	// 基本验证
	if emailArgs.To == "" {
		return errors.ErrInvalidArgument.WithMessage("email address is required")
	}
	if emailArgs.Subject == "" {
		return errors.ErrInvalidArgument.WithMessage("subject is required")
	}

	// HTML 转义邮件内容，防止 XSS 攻击
	// 即使邮件客户端支持 HTML，也应该转义用户输入的内容
	escapedContent := utils.EscapeString(emailArgs.Content)
	escapedSubject := utils.EscapeString(emailArgs.Subject)

	// 实际场景中这里会调用邮件服务发送邮件
	err := sendEmail(emailArgs.To, escapedSubject, escapedContent)
	if err != nil {
		facades.Log().Errorf("📧 [Job] 发送邮件失败 - 收件人: %s, 主题: %s, 错误: %v", emailArgs.To, emailArgs.Subject, err)
		return err // 返回错误，触发重试
	}

	facades.Log().Infof("📧 [Job] 发送邮件成功 - 收件人: %s, 主题: %s", emailArgs.To, emailArgs.Subject)
	return nil
}

// ShouldRetry 自定义重试逻辑：递增延迟重试
//
// 参数:
//   - err: 错误信息
//   - attempt: 当前重试次数（从1开始，第1次重试时attempt=1，第2次重试时attempt=2）
//
// 返回:
//   - retryable: 是否重试
//   - delay: 延迟时间
func (r *SendEmail) ShouldRetry(err error, attempt int) (retryable bool, delay time.Duration) {

	maxRetries := 3 // 最大重试次数

	if attempt > maxRetries {
		facades.Log().Errorf("📧 [Job] 已达到最大重试次数 %d，不再重试", maxRetries)
		return false, 0 // 不再重试
	}

	// 递增延迟重试：3秒、10秒、20秒
	delays := []time.Duration{3 * time.Second, 10 * time.Second, 20 * time.Second}

	// 获取当前重试的延迟时间（attempt从1开始，所以减1作为索引）
	delayIndex := max(attempt-1, 0)
	if delayIndex < len(delays) {
		delay = delays[delayIndex]
	} else {
		// 如果超过配置的延迟数组，使用最后一个延迟时间
		if len(delays) > 0 {
			delay = delays[len(delays)-1]
		} else {
			delay = 3 * time.Second // 默认延迟
		}
	}

	facades.Log().Infof("📧 [Job] 第 %d 次重试，将在 %v 后执行", attempt, delay)
	return true, delay
}

// sendEmail 发送邮件函数
//
// 参数:
//   - to: 收件人邮箱
//   - subject: 邮件主题
//   - content: 邮件内容
//
// 返回:
//   - error: 错误信息
func sendEmail(to, subject, content string) error {
	return facades.Mail().
		To([]string{to}).
		Subject(subject).
		Content(mail.Content{Html: content}).
		Send()
}
