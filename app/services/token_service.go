package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"

	"github.com/goravel/framework/contracts/database/orm"
	"github.com/goravel/framework/facades"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/utils/errorlog"
	"goravel/app/utils/traceid"
)

type TokenService interface {
	CreateToken(tokenableType string, tokenableID uint, name string, expiresAt *time.Time, browser, ip, os, sessionID string) (string, *models.PersonalAccessToken, error)
	FindToken(token string) (*models.PersonalAccessToken, error)
	DeleteToken(token string) error
	DeleteTokensByUser(tokenableType string, tokenableID uint) error
	GetTokensByUser(tokenableType string, tokenableID uint) ([]models.PersonalAccessToken, error)
	UpdateLastUsedAt(token string) error
}

type TokenServiceImpl struct {
	ctx      context.Context
	platform bool
}

var (
	personalAccessTokensColumnsOnce sync.Once
	hasPATBrowserColumn             bool
	hasPATIPColumn                  bool
	hasPATOSColumn                  bool
	hasPATSessionIDColumn           bool
)

func NewTokenServiceImpl(ctx context.Context) *TokenServiceImpl {
	return &TokenServiceImpl{ctx: ctx}
}

// NewPlatformTokenService stores/looks up tokens on the platform (default) connection.
func NewPlatformTokenService(ctx context.Context) *TokenServiceImpl {
	if ctx == nil {
		ctx = context.Background()
	}
	return &TokenServiceImpl{ctx: ctx, platform: true}
}

func (s *TokenServiceImpl) query() orm.Query {
	if s.platform {
		return appfacades.PlatformOrmQuery(s.ctx)
	}
	return appfacades.OrmQuery(s.ctx)
}

func loadPersonalAccessTokenColumns() {
	personalAccessTokensColumnsOnce.Do(func() {
		if !facades.Schema().HasTable("personal_access_tokens") {
			return
		}
		hasPATBrowserColumn = facades.Schema().HasColumn("personal_access_tokens", "browser")
		hasPATIPColumn = facades.Schema().HasColumn("personal_access_tokens", "ip")
		hasPATOSColumn = facades.Schema().HasColumn("personal_access_tokens", "os")
		hasPATSessionIDColumn = facades.Schema().HasColumn("personal_access_tokens", "session_id")
	})
}

func (s *TokenServiceImpl) CreateToken(tokenableType string, tokenableID uint, name string, expiresAt *time.Time, browser, ip, os, sessionID string) (string, *models.PersonalAccessToken, error) {
	plainToken := s.generateRandomToken()
	tokenHash := s.hashToken(plainToken)

	if sessionID == "" {
		if len(tokenHash) >= 16 {
			sessionID = tokenHash[:16]
		} else {
			sessionID = tokenHash
		}
	}

	now := time.Now()
	loadPersonalAccessTokenColumns()
	payload := map[string]any{
		"tokenable_type": tokenableType,
		"tokenable_id":   tokenableID,
		"name":           name,
		"token":          tokenHash,
		"expires_at":     expiresAt,
		"last_used_at":   &now,
	}
	if hasPATBrowserColumn {
		payload["browser"] = browser
	}
	if hasPATIPColumn {
		payload["ip"] = ip
	}
	if hasPATOSColumn {
		payload["os"] = os
	}
	if hasPATSessionIDColumn {
		payload["session_id"] = sessionID
	}

	if err := s.query().Table("personal_access_tokens").Create(payload); err != nil {
		return "", nil, err
	}

	var accessToken models.PersonalAccessToken
	if err := s.query().Where("token", tokenHash).First(&accessToken); err != nil {
		return plainToken, nil, nil
	}

	return plainToken, &accessToken, nil
}

func (s *TokenServiceImpl) FindToken(token string) (*models.PersonalAccessToken, error) {
	if token == "" {
		return nil, apperrors.ErrInvalidArgument.WithMessage("token is empty")
	}

	tokenHash := s.hashToken(token)
	var accessToken models.PersonalAccessToken
	if err := s.query().Where("token", tokenHash).FirstOrFail(&accessToken); err != nil {
		return nil, err
	}

	if accessToken.ExpiresAt != nil && accessToken.ExpiresAt.Before(time.Now()) {
		if _, err := s.query().Delete(&accessToken); err != nil {
			ctx, _ := traceid.EnsureContext(context.Background())
			errorlog.Record(ctx, "token", "Failed to delete expired token", map[string]any{
				"token_id":     accessToken.ID,
				"tokenable_id": accessToken.TokenableID,
				"expires_at":   accessToken.ExpiresAt,
				"error":        err.Error(),
			}, "Failed to delete expired token (ID: %d): %v", accessToken.ID, err)
		}
		return nil, apperrors.ErrInvalidArgument.WithMessage("token expired")
	}

	return &accessToken, nil
}

func (s *TokenServiceImpl) DeleteToken(token string) error {
	tokenHash := s.hashToken(token)
	_, err := s.query().Where("token", tokenHash).Delete(&models.PersonalAccessToken{})
	return err
}

func (s *TokenServiceImpl) DeleteTokensByUser(tokenableType string, tokenableID uint) error {
	_, err := s.query().
		Where("tokenable_type", tokenableType).
		Where("tokenable_id", tokenableID).
		Delete(&models.PersonalAccessToken{})
	return err
}

func (s *TokenServiceImpl) GetTokensByUser(tokenableType string, tokenableID uint) ([]models.PersonalAccessToken, error) {
	var tokens []models.PersonalAccessToken
	err := s.query().
		Where("tokenable_type", tokenableType).
		Where("tokenable_id", tokenableID).
		Order("created_at desc").
		Find(&tokens)
	return tokens, err
}

func (s *TokenServiceImpl) UpdateLastUsedAt(token string) error {
	tokenHash := s.hashToken(token)
	now := time.Now()
	_, err := s.query().
		Model(&models.PersonalAccessToken{}).
		Where("token", tokenHash).
		Update("last_used_at", now)
	return err
}

func (s *TokenServiceImpl) generateRandomToken() string {
	b := make([]byte, 40)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)[:40]
}

func (s *TokenServiceImpl) hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
