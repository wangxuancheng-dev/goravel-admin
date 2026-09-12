package services

import (
	"context"
	"strings"

	"github.com/goravel/framework/facades"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/models"
)

// SensitiveConfirmService 敏感操作二次确认（绑定 2FA 时校验 TOTP，否则校验当前密码）。
type SensitiveConfirmService interface {
	VerifySensitiveConfirm(adminID uint, confirmCode string) error
}

type SensitiveConfirmServiceImpl struct {
	ctx context.Context
}

func NewSensitiveConfirmService(ctx context.Context) SensitiveConfirmService {
	return &SensitiveConfirmServiceImpl{ctx: ctx}
}

func (s *SensitiveConfirmServiceImpl) VerifySensitiveConfirm(adminID uint, confirmCode string) error {
	confirmCode = strings.TrimSpace(confirmCode)
	if confirmCode == "" {
		return apperrors.ErrSensitiveConfirmRequired
	}

	var admin models.Admin
	if err := appfacades.OrmQuery(s.ctx).Where("id", adminID).FirstOrFail(&admin); err != nil {
		return apperrors.ErrAdminNotFound.WithError(err)
	}

	ga := NewGoogleAuthenticatorServiceImpl(s.ctx)
	bound, err := ga.IsBound(adminID)
	if err != nil {
		return err
	}

	return MatchSensitiveConfirm(SensitiveConfirmInput{
		Has2FA:       bound,
		ConfirmCode:  confirmCode,
		GoogleSecret: admin.GoogleSecret,
		PasswordHash: admin.Password,
		VerifyTOTP: func(secret, code string) bool {
			return ga.Verify(secret, code)
		},
		CheckPassword: func(plain, hash string) bool {
			return facades.Hash().Check(plain, hash)
		},
	})
}

// SensitiveConfirmInput 二次确认纯逻辑入参（便于单测）。
type SensitiveConfirmInput struct {
	Has2FA        bool
	ConfirmCode   string
	GoogleSecret  string
	PasswordHash  string
	VerifyTOTP    func(secret, code string) bool
	CheckPassword func(plain, hash string) bool
}

// MatchSensitiveConfirm 二次确认纯逻辑。
func MatchSensitiveConfirm(in SensitiveConfirmInput) error {
	code := strings.TrimSpace(in.ConfirmCode)
	if code == "" {
		return apperrors.ErrSensitiveConfirmRequired
	}
	if in.Has2FA {
		if in.VerifyTOTP == nil || !in.VerifyTOTP(in.GoogleSecret, code) {
			return apperrors.ErrSensitiveConfirmInvalid
		}
		return nil
	}
	if in.CheckPassword == nil || !in.CheckPassword(code, in.PasswordHash) {
		return apperrors.ErrSensitiveConfirmInvalid
	}
	return nil
}

// VerifySensitiveConfirm 便捷方法：使用当前请求上下文校验二次确认。
func VerifySensitiveConfirm(ctx context.Context, adminID uint, confirmCode string) error {
	return NewSensitiveConfirmService(ctx).VerifySensitiveConfirm(adminID, confirmCode)
}
