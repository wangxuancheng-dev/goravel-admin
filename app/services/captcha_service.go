package services

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/goravel/framework/facades"
	"github.com/mojocn/base64Captcha"

	"goravel/app/tenancy"
	"goravel/app/utils"
)

type CaptchaService interface {
	Enabled() bool
	// Kind returns the active captcha type (image or slide).
	Kind() string
	// Generate issues a classic text captcha (id, data URL).
	Generate() (string, string, error)
	// GenerateChallenge issues a captcha of the active Kind.
	GenerateChallenge() (*CaptchaChallenge, error)
	Verify(id, answer string) (bool, string)
}

type CaptchaServiceImpl struct {
	ctx         context.Context
	driver      base64Captcha.Driver
	initialized bool
	kind        string
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

// NewPlatformCaptchaService builds the captcha service for the platform console.
// The type comes from env PLATFORM_CAPTCHA_TYPE (image|slide), see config/tenancy.go.
func NewPlatformCaptchaService(ctx context.Context) CaptchaService {
	return &CaptchaServiceImpl{
		ctx:         ctx,
		initialized: false,
		kind:        NormalizeCaptchaType(facades.Config().GetString("tenancy.platform_captcha_type", CaptchaTypeImage)),
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

// Kind returns the active captcha type (image or slide).
// Tenant/admin login: per-tenant DB config captcha.captcha_type.
// Platform login: fixed from env at construction time.
func (s *CaptchaServiceImpl) Kind() string {
	if s.kind == "" {
		s.kind = NormalizeCaptchaType(utils.GetConfigValue(s.ctx, "captcha", "captcha_type", CaptchaTypeImage))
	}
	return s.kind
}

// GenerateChallenge issues a captcha of the active Kind and registers its answer.
func (s *CaptchaServiceImpl) GenerateChallenge() (*CaptchaChallenge, error) {
	if s.Kind() != CaptchaTypeSlide {
		id, b64s, err := s.Generate()
		if err != nil {
			return nil, err
		}
		return &CaptchaChallenge{Type: CaptchaTypeImage, ID: id, Image: b64s}, nil
	}

	s.initDriver() // also initializes the shared store with the configured expiry
	challenge, answerX, err := generateSlide()
	if err != nil {
		return nil, err
	}
	id := base64Captcha.RandomId()
	key := tenancy.CacheKey(s.ctx, id)
	if err := sharedCaptchaStore(120).Set(key, slideStorePrefix+strconv.Itoa(answerX)); err != nil {
		return nil, err
	}
	challenge.ID = key
	return challenge, nil
}

func (s *CaptchaServiceImpl) Verify(id, answer string) (bool, string) {
	if id == "" || strings.TrimSpace(answer) == "" {
		return false, "captcha_required"
	}

	s.initDriver()
	// Always consume the entry (one attempt per captcha), pass or fail.
	expected := sharedCaptchaStore(120).Get(id, true)
	if expected == "" {
		return false, "captcha_expired"
	}

	isSlide := strings.HasPrefix(expected, slideStorePrefix)
	if isSlide != (s.Kind() == CaptchaTypeSlide) {
		return false, "captcha_invalid"
	}

	if isSlide {
		x, err := strconv.Atoi(strings.TrimPrefix(expected, slideStorePrefix))
		if err != nil || !verifySlideAnswer(x, answer) {
			return false, "captcha_invalid"
		}
		return true, ""
	}

	if !strings.EqualFold(expected, strings.TrimSpace(answer)) {
		return false, "captcha_invalid"
	}

	return true, ""
}

// PeekCaptchaAnswer returns the stored answer without consuming it (feature tests).
// For slide captchas this is the target x offset as a string.
func PeekCaptchaAnswer(id string) string {
	if strings.TrimSpace(id) == "" {
		return ""
	}
	return strings.TrimPrefix(sharedCaptchaStore(120).Get(id, false), slideStorePrefix)
}