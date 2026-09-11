package services

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/mojocn/base64Captcha"

	"goravel/app/tenancy"
	"goravel/app/utils"
)

type CaptchaService interface {
	Enabled() bool
	Generate() (string, string, error)
	Verify(id, answer string) (bool, string)
}

type CaptchaServiceImpl struct {
	ctx         context.Context
	driver      base64Captcha.Driver
	initialized bool
}

// Shared in-process store so Generate and Verify across HTTP requests see the same captcha.
// Per-request NewMemoryStore made every login return captcha_expired.
var (
	captchaStoreOnce sync.Once
	captchaStore     base64Captcha.Store
)

func sharedCaptchaStore(expireSeconds int) base64Captcha.Store {
	captchaStoreOnce.Do(func() {
		if expireSeconds < 30 {
			expireSeconds = 120
		}
		captchaStore = base64Captcha.NewMemoryStore(1024, time.Duration(expireSeconds)*time.Second)
	})
	return captchaStore
}

func NewCaptchaServiceImpl(ctx context.Context) CaptchaService {
	// Delay driver init so constructing the service does not hit the database.
	return &CaptchaServiceImpl{
		ctx:         ctx,
		initialized: false,
	}
}

func (s *CaptchaServiceImpl) initDriver() {
	if s.initialized {
		return
	}

	expireSeconds := utils.GetConfigValueInt(s.ctx, "captcha", "captcha_expire", 120)
	if expireSeconds < 30 {
		expireSeconds = 120
	}

	s.driver = base64Captcha.NewDriverString(
		50,  // height
		180, // width
		5,   // noise count
		base64Captcha.OptionShowHollowLine|base64Captcha.OptionShowSineLine,
		5, // length
		"2345678ABCDEFGHJKLMNPQRSTUVWXYZabcdefghjkmnpqrstuvwxyz",
		nil,
		nil,
		nil,
	)
	_ = sharedCaptchaStore(expireSeconds)
	s.initialized = true
}

func (s *CaptchaServiceImpl) Enabled() bool {
	return utils.GetConfigValueBool(s.ctx, "captcha", "captcha_enabled", false)
}

func (s *CaptchaServiceImpl) Generate() (string, string, error) {
	s.initDriver()
	store := sharedCaptchaStore(120)
	c := base64Captcha.NewCaptcha(s.driver, store)
	id, b64s, _, err := c.Generate()
	if err != nil {
		return "", "", err
	}
	// Tenant-prefix store key so captchas don't collide across tenants on one process.
	key := tenancy.CacheKey(s.ctx, id)
	if key != id {
		val := store.Get(id, true)
		if val != "" {
			_ = store.Set(key, val)
		}
		id = key
	}
	return id, b64s, nil
}

func (s *CaptchaServiceImpl) Verify(id, answer string) (bool, string) {
	if id == "" || strings.TrimSpace(answer) == "" {
		return false, "captcha_required"
	}

	s.initDriver()
	expected := sharedCaptchaStore(120).Get(id, true)
	if expected == "" {
		return false, "captcha_expired"
	}

	if !strings.EqualFold(expected, strings.TrimSpace(answer)) {
		return false, "captcha_invalid"
	}

	return true, ""
}
