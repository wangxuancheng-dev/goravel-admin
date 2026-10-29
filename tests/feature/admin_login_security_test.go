package feature_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/goravel/framework/facades"
	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"goravel/app/models"
	"goravel/app/services"
	"goravel/tests"
)

const loginSecurityPassword = "LoginSec123!"

// testHTTPPeerIP is the RemoteAddr of the in-process test HTTP client (not a trusted proxy).
const testHTTPPeerIP = "192.0.2.1"

type loginResult struct {
	Code      int    `json:"code"`
	ErrorCode string `json:"error_code"`
	Data      struct {
		Token       string `json:"token"`
		LoginTicket string `json:"login_ticket"`
	} `json:"data"`
}

func upsertLoginSecurityAdmin(t *testing.T, username, googleSecret string, status uint8) {
	t.Helper()
	hashed, err := facades.Hash().Make(loginSecurityPassword)
	require.NoError(t, err)

	var admin models.Admin
	require.NoError(t, facades.Orm().Query().Where("username", username).First(&admin))
	if admin.ID == 0 {
		admin = models.Admin{Username: username, Password: hashed, Nickname: username, Status: 1}
		require.NoError(t, facades.Orm().Query().Create(&admin))
	}
	// Explicit update: gorm skips zero values (status 0, empty secret) on create.
	_, err = facades.Orm().Query().Model(&models.Admin{}).Where("id", admin.ID).Update(map[string]any{
		"password":      hashed,
		"google_secret": googleSecret,
		"status":        status,
	})
	require.NoError(t, err)

	lockout := services.NewLoginLockoutService(context.Background())
	lockout.ClearFailures(testHTTPPeerIP, username)
	t.Cleanup(func() { lockout.ClearFailures(testHTTPPeerIP, username) })
}

func withAdminCaptcha(t *testing.T, enabled bool) {
	t.Helper()
	cfg := services.NewConfigService(context.Background())
	prev := map[string]string{"captcha_enabled": "0", "captcha_type": "image"}
	rows, err := cfg.GetByGroup("captcha")
	require.NoError(t, err)
	for _, row := range rows {
		if _, ok := prev[row.Key]; ok {
			prev[row.Key] = row.Value
		}
	}
	t.Cleanup(func() {
		_ = cfg.Save("captcha", map[string]any{
			"captcha_enabled": prev["captcha_enabled"],
			"captcha_type":    prev["captcha_type"],
		})
	})
	value := "0"
	if enabled {
		value = "1"
	}
	require.NoError(t, cfg.Save("captcha", map[string]any{"captcha_enabled": value, "captcha_type": "image"}))
}

func postAdminLogin(t *testing.T, body map[string]any, headers map[string]string) loginResult {
	t.Helper()
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	testCase := tests.TestCase{}
	req := testCase.Http(t).WithHeader("Content-Type", "application/json")
	for k, v := range headers {
		req = req.WithHeader(k, v)
	}
	resp, err := req.Post("/api/admin/login", strings.NewReader(string(raw)))
	require.NoError(t, err)
	content, err := resp.Content()
	require.NoError(t, err)
	var out loginResult
	require.NoError(t, json.Unmarshal([]byte(content), &out), content)
	return out
}

func solvedAdminCaptcha(t *testing.T) (string, string) {
	t.Helper()
	info := fetchAdminLoginCaptcha(t)
	id, _ := info["captcha_id"].(string)
	require.NotEmpty(t, id)
	return id, services.PeekCaptchaAnswer(id)
}

// Bound 2FA no longer skips the captcha: the password step is captcha-gated, and the
// Google-code step rides on a one-time ticket instead of a second captcha.
func TestAdminLogin2FAStillRequiresCaptchaAndTicket(t *testing.T) {
	withTenancyDriver(t, "off")
	withAdminCaptcha(t, true)

	key, err := totp.Generate(totp.GenerateOpts{Issuer: "test", AccountName: "sec_2fa_admin"})
	require.NoError(t, err)
	const username = "sec_2fa_admin"
	upsertLoginSecurityAdmin(t, username, key.Secret(), 1)

	// No captcha: rejected before the password is looked at (no right/wrong-password oracle).
	res := postAdminLogin(t, map[string]any{"username": username, "password": "wrong-password", "google_code": "000000"}, nil)
	assert.Equal(t, "captcha_required", res.ErrorCode)
	res = postAdminLogin(t, map[string]any{"username": username, "password": loginSecurityPassword, "google_code": "000000"}, nil)
	assert.Equal(t, "captcha_required", res.ErrorCode)

	// Step 1: captcha + password → ticket.
	id, answer := solvedAdminCaptcha(t)
	res = postAdminLogin(t, map[string]any{"username": username, "password": loginSecurityPassword, "captcha_id": id, "captcha_answer": answer}, nil)
	require.Equal(t, "google_code_required", res.ErrorCode)
	ticket := res.Data.LoginTicket
	require.NotEmpty(t, ticket)

	// Forged ticket is rejected.
	res = postAdminLogin(t, map[string]any{"username": username, "password": loginSecurityPassword, "login_ticket": "forged", "google_code": "000000"}, nil)
	assert.Equal(t, "login_ticket_expired", res.ErrorCode)

	// Step 2: wrong code keeps the ticket usable.
	res = postAdminLogin(t, map[string]any{"username": username, "password": loginSecurityPassword, "login_ticket": ticket, "google_code": "000000"}, nil)
	assert.Equal(t, "google_code_invalid", res.ErrorCode)

	code, err := totp.GenerateCode(key.Secret(), time.Now())
	require.NoError(t, err)
	res = postAdminLogin(t, map[string]any{"username": username, "password": loginSecurityPassword, "login_ticket": ticket, "google_code": code}, nil)
	require.Equal(t, 200, res.Code, res.ErrorCode)
	require.NotEmpty(t, res.Data.Token)

	// Ticket is single use.
	res = postAdminLogin(t, map[string]any{"username": username, "password": loginSecurityPassword, "login_ticket": ticket, "google_code": code}, nil)
	assert.Equal(t, "login_ticket_expired", res.ErrorCode)

	// The same TOTP code cannot be replayed within its validity window.
	id, answer = solvedAdminCaptcha(t)
	res = postAdminLogin(t, map[string]any{"username": username, "password": loginSecurityPassword, "captcha_id": id, "captcha_answer": answer, "google_code": code}, nil)
	assert.Equal(t, "google_code_invalid", res.ErrorCode)
}

// A disabled account is only revealed to someone who knows the password.
func TestAdminLoginDisabledAccountNeedsPasswordFirst(t *testing.T) {
	withTenancyDriver(t, "off")
	withAdminCaptcha(t, false)
	const username = "sec_disabled_admin"
	upsertLoginSecurityAdmin(t, username, "", 0)

	res := postAdminLogin(t, map[string]any{"username": username, "password": "wrong-password"}, nil)
	assert.Equal(t, "username_or_password_error", res.ErrorCode)
	res = postAdminLogin(t, map[string]any{"username": username, "password": loginSecurityPassword}, nil)
	assert.Equal(t, "account_disabled", res.ErrorCode)
}

// Proxy headers from an untrusted peer are ignored, so rotating them does not reset per-IP counters.
func TestAdminLoginSpoofedIPHeadersDoNotResetFailures(t *testing.T) {
	withTenancyDriver(t, "off")
	withAdminCaptcha(t, false)
	const username = "sec_spoof_admin"
	upsertLoginSecurityAdmin(t, username, "", 1)

	for i := 0; i < 3; i++ {
		fake := fmt.Sprintf("203.0.113.%d", 10+i)
		res := postAdminLogin(t, map[string]any{"username": username, "password": "wrong-password"}, map[string]string{
			"X-Forwarded-For":  fake,
			"X-Real-IP":        fake,
			"CF-Connecting-IP": fake,
		})
		if i < 2 {
			assert.Equal(t, "username_or_password_error", res.ErrorCode, "attempt %d", i)
		} else {
			// Third attempt from "another IP" is still the same peer → adaptive captcha kicks in.
			assert.Equal(t, "captcha_required", res.ErrorCode, "attempt %d", i)
		}
	}
}

// Account-wide counters catch attackers rotating real IPs; manual unlock does not open a lockout-free window.
func TestLoginLockoutAccountDimensionAndUnlock(t *testing.T) {
	withTenancyDriver(t, "off")
	lockout := services.NewLoginLockoutService(context.Background())
	username := fmt.Sprintf("sec_account_%d", time.Now().UnixNano())
	t.Cleanup(func() { lockout.UnlockUsername(username) })

	captchaAfter := facades.Config().GetInt("login_security.account_captcha_after", 5)
	maxAttempts := facades.Config().GetInt("login_security.account_max_attempts", 20)

	for i := 0; i < captchaAfter; i++ {
		lockout.RecordFailure(fmt.Sprintf("198.51.100.%d", i), username)
	}
	assert.True(t, lockout.NeedsCaptcha("198.51.100.250", username), "fresh IP must still need captcha")

	for i := captchaAfter; i < maxAttempts; i++ {
		lockout.RecordFailure(fmt.Sprintf("198.51.%d.%d", i/250, i%250), username)
	}
	locked, _ := lockout.IsLocked("198.51.100.251", username)
	assert.True(t, locked, "account lock applies to every IP")

	lockout.UnlockUsername(username)
	locked, _ = lockout.IsLocked("198.51.100.251", username)
	assert.False(t, locked)

	// New failures after the unlock lock again (previously the unlock window disabled lockout entirely).
	time.Sleep(time.Millisecond)
	maxPerIP := facades.Config().GetInt("login_security.max_attempts", 5)
	for i := 0; i < maxPerIP; i++ {
		lockout.RecordFailure("198.51.100.252", username)
	}
	locked, _ = lockout.IsLocked("198.51.100.252", username)
	assert.True(t, locked)
}

// Case / trailing-space variants hit the same MySQL row, so they must share lockout counters.
func TestLoginLockoutUsernameVariantsShareCounters(t *testing.T) {
	withTenancyDriver(t, "off")
	lockout := services.NewLoginLockoutService(context.Background())
	base := fmt.Sprintf("sec_variant_%d", time.Now().UnixNano())
	t.Cleanup(func() { lockout.UnlockUsername(base) })

	maxPerIP := facades.Config().GetInt("login_security.max_attempts", 5)
	variants := []string{base, strings.ToUpper(base), base + " ", " " + strings.ToUpper(base[:1]) + base[1:]}
	for i := 0; i < maxPerIP; i++ {
		lockout.RecordFailure("198.51.100.30", variants[i%len(variants)])
	}
	locked, _ := lockout.IsLocked("198.51.100.30", base)
	assert.True(t, locked)
}

func TestLoginTwoFactorTicketUsernameWithSeparator(t *testing.T) {
	withTenancyDriver(t, "off")
	ctx := context.Background()
	ticket, err := services.IssueLoginTwoFactorTicket(ctx, "admin", 42, "a|b", "198.51.100.40")
	require.NoError(t, err)
	t.Cleanup(func() { services.ConsumeLoginTwoFactorTicket(ctx, ticket) })

	assert.Equal(t, uint(42), services.LoginTwoFactorTicketSubject(ctx, ticket, "admin", "A|B ", "198.51.100.40"))
	assert.Zero(t, services.LoginTwoFactorTicketSubject(ctx, ticket, "admin", "a|b", "198.51.100.41"))
	assert.Zero(t, services.LoginTwoFactorTicketSubject(ctx, ticket, "platform", "a|b", "198.51.100.40"))
}

// Weak-password errors list every missing character class, translated for the request language.
func TestRegisterWeakPasswordMessageListsMissingChars(t *testing.T) {
	withTenancyDriver(t, "off")
	username := fmt.Sprintf("sec_reg_%d", time.Now().UnixNano())

	for lang, want := range map[string][]string{
		"zh-CN": {"密码还需包含：", "数字、特殊符号"},
		"en-US": {"Password must also contain:", "a number, a special character"},
	} {
		testCase := tests.TestCase{}
		resp, err := testCase.Http(t).
			WithHeader("Content-Type", "application/json").
			WithHeader("Accept-Language", lang).
			Post("/api/user/register", strings.NewReader(fmt.Sprintf(`{"username":%q,"password":"kestrelhawk"}`, username)))
		require.NoError(t, err)
		content, err := resp.Content()
		require.NoError(t, err)

		var payload struct {
			ErrorCode string `json:"error_code"`
			Message   string `json:"message"`
		}
		require.NoError(t, json.Unmarshal([]byte(content), &payload), content)
		assert.Equal(t, "password_missing_chars", payload.ErrorCode, content)
		for _, w := range want {
			assert.Contains(t, payload.Message, w, lang)
		}
	}
}

// /info exposes the effective password policy so forms can validate before submitting.
func TestAdminInfoExposesPasswordPolicy(t *testing.T) {
	withTenancyDriver(t, "off")
	token := loginSmokeAdmin(t)

	testCase := tests.TestCase{}
	resp, err := testCase.Http(t).
		WithHeader("Authorization", "Bearer "+token).
		Get("/api/admin/info")
	require.NoError(t, err)
	content, err := resp.Content()
	require.NoError(t, err)

	var payload struct {
		Data struct {
			Config struct {
				PasswordPolicy map[string]any `json:"password_policy"`
			} `json:"config"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(content), &payload), content)
	policy := payload.Data.Config.PasswordPolicy
	require.NotNil(t, policy, content)
	for _, key := range []string{"min_length", "require_letter", "require_number", "require_special"} {
		assert.Contains(t, policy, key)
	}
}

// A rejected weak-password reset is recorded as a failed operation with the error message.
func TestResetAdminWeakPasswordLoggedAsFailed(t *testing.T) {
	withTenancyDriver(t, "off")
	const operatorPassword = "OpLogReset#2024"
	token := loginAdminWithPermission(t, fmt.Sprintf("sec_oplog_%d", time.Now().UnixNano()), operatorPassword,
		"admin.password", "PUT", "/api/admin/admins/*/password")
	target := ensureNamedAdmin(t, fmt.Sprintf("sec_target_%d", time.Now().UnixNano()), loginSecurityPassword)

	testCase := tests.TestCase{}
	resp, err := testCase.Http(t).
		WithHeader("Authorization", "Bearer "+token).
		WithHeader("Content-Type", "application/json").
		Put(fmt.Sprintf("/api/admin/admins/%d/password", target.ID),
			strings.NewReader(fmt.Sprintf(`{"password":"123456","confirm_code":%q}`, operatorPassword)))
	require.NoError(t, err)
	content, err := resp.Content()
	require.NoError(t, err)
	assert.Contains(t, content, "password_requirements_unmet")

	path := fmt.Sprintf("/api/admin/admins/%d/password", target.ID)
	var log models.OperationLog
	require.Eventually(t, func() bool {
		log = models.OperationLog{}
		_ = facades.Orm().Query().Where("path", path).OrderBy("id", "desc").First(&log)
		return log.ID > 0 && log.Status == 0
	}, 5*time.Second, 50*time.Millisecond, "operation log should be marked failed")
	assert.Contains(t, log.ErrorMsg, "password_requirements_unmet")
}
