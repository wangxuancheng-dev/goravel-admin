package feature_test

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	nethttp "net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/goravel/framework/facades"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "goravel/app/errors"
	"goravel/app/models"
	"goravel/tests"
)

const fakeOIDCClientID = "goravel-admin-test"

// fakeOIDCIdP is an in-process OpenID provider: discovery, JWKS and a token endpoint
// that returns an RS256 id_token whose claims the test registers per authorization code.
type fakeOIDCIdP struct {
	srv    *httptest.Server
	key    *rsa.PrivateKey
	mu     sync.Mutex
	seq    int
	claims map[string]map[string]any
}

func newFakeOIDCIdP(t *testing.T) *fakeOIDCIdP {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	idp := &fakeOIDCIdP{key: key, claims: map[string]map[string]any{}}

	mux := nethttp.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w nethttp.ResponseWriter, r *nethttp.Request) {
		base := idp.srv.URL
		writeJSON(w, map[string]any{
			"issuer":                                base,
			"authorization_endpoint":                base + "/authorize",
			"token_endpoint":                        base + "/token",
			"jwks_uri":                              base + "/jwks",
			"response_types_supported":              []string{"code"},
			"subject_types_supported":               []string{"public"},
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})
	mux.HandleFunc("/jwks", func(w nethttp.ResponseWriter, r *nethttp.Request) {
		pub := idp.key.PublicKey
		writeJSON(w, map[string]any{"keys": []map[string]any{{
			"kty": "RSA", "kid": "test", "alg": "RS256", "use": "sig",
			"n": b64url(pub.N.Bytes()),
			"e": b64url(big.NewInt(int64(pub.E)).Bytes()),
		}}})
	})
	mux.HandleFunc("/token", func(w nethttp.ResponseWriter, r *nethttp.Request) {
		_ = r.ParseForm()
		idp.mu.Lock()
		claims, ok := idp.claims[r.PostForm.Get("code")]
		idp.mu.Unlock()
		if !ok {
			w.WriteHeader(nethttp.StatusBadRequest)
			writeJSON(w, map[string]any{"error": "invalid_grant"})
			return
		}
		writeJSON(w, map[string]any{
			"access_token": "fake-access-token",
			"token_type":   "Bearer",
			"expires_in":   300,
			"id_token":     idp.sign(t, claims),
		})
	})
	idp.srv = httptest.NewServer(mux)
	t.Cleanup(idp.srv.Close)
	return idp
}

// issueCode registers the claims the token endpoint returns for a new code.
// iss/aud/exp/iat are filled in; nil values drop the claim (e.g. email_verified).
func (idp *fakeOIDCIdP) issueCode(claims map[string]any) string {
	now := time.Now()
	full := map[string]any{
		"iss": idp.srv.URL,
		"aud": fakeOIDCClientID,
		"iat": now.Unix(),
		"exp": now.Add(5 * time.Minute).Unix(),
	}
	for k, v := range claims {
		if v == nil {
			delete(full, k)
			continue
		}
		full[k] = v
	}
	idp.mu.Lock()
	idp.seq++
	code := fmt.Sprintf("code-%d", idp.seq)
	idp.claims[code] = full
	idp.mu.Unlock()
	return code
}

func (idp *fakeOIDCIdP) sign(t *testing.T, claims map[string]any) string {
	header, _ := json.Marshal(map[string]any{"alg": "RS256", "kid": "test", "typ": "JWT"})
	payload, _ := json.Marshal(claims)
	signingInput := b64url(header) + "." + b64url(payload)
	sum := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, idp.key, crypto.SHA256, sum[:])
	if err != nil {
		t.Errorf("sign id_token: %v", err)
	}
	return signingInput + "." + b64url(sig)
}

func b64url(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func writeJSON(w nethttp.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// withOIDCConfig points process OIDC config at the fake IdP and blanks any tenant
// overrides (configs group "oidc") on the dev DB, restoring both on cleanup.
func withOIDCConfig(t *testing.T, idp *fakeOIDCIdP, autoProvision, allowUnverified bool) {
	t.Helper()
	values := map[string]any{
		"oidc.enabled":                true,
		"oidc.issuer":                 idp.srv.URL,
		"oidc.client_id":              fakeOIDCClientID,
		"oidc.client_secret":          "secret",
		"oidc.redirect_url":           "http://localhost/api/admin/auth/oidc/callback",
		"oidc.auto_provision":         autoProvision,
		"oidc.allow_unverified_email": allowUnverified,
	}
	for key, v := range values {
		prev := facades.Config().Get(key)
		facades.Config().Add(key, v)
		t.Cleanup(func() { facades.Config().Add(key, prev) })
	}

	var rows []models.Config
	require.NoError(t, facades.Orm().Query().Where("group", "oidc").Get(&rows))
	for _, row := range rows {
		if row.Value == "" {
			continue
		}
		id, prev := row.ID, row.Value
		_, err := facades.Orm().Query().Model(&models.Config{}).Where("id", id).Update("value", "")
		require.NoError(t, err)
		t.Cleanup(func() {
			_, _ = facades.Orm().Query().Model(&models.Config{}).Where("id", id).Update("value", prev)
		})
	}
}

// createOIDCAdmin creates an admin with a unique email and removes it, its tokens
// and its login logs on cleanup.
func createOIDCAdmin(t *testing.T, status uint8) models.Admin {
	t.Helper()
	suffix := time.Now().UnixNano()
	hashed, err := facades.Hash().Make(loginSecurityPassword)
	require.NoError(t, err)
	admin := models.Admin{
		Username: fmt.Sprintf("oidc_e2e_%d", suffix),
		Password: hashed,
		Nickname: "oidc e2e",
		Email:    fmt.Sprintf("oidc-e2e-%d@example.test", suffix),
		Status:   1,
	}
	require.NoError(t, facades.Orm().Query().Create(&admin))
	if status != 1 {
		_, err = facades.Orm().Query().Model(&models.Admin{}).Where("id", admin.ID).Update("status", status)
		require.NoError(t, err)
		admin.Status = status
	}
	cleanupOIDCAdmin(t, admin.Email)
	return admin
}

func cleanupOIDCAdmin(t *testing.T, email string) {
	t.Cleanup(func() {
		var admins []models.Admin
		_ = facades.Orm().Query().Where("email", email).Get(&admins)
		for _, a := range admins {
			_, _ = facades.Orm().Query().Model(&models.PersonalAccessToken{}).
				Where("tokenable_type", "admin").Where("tokenable_id", a.ID).Delete()
			_, _ = facades.Orm().Query().Model(&models.LoginLog{}).Where("admin_id", a.ID).Delete()
			_, _ = facades.Orm().Query().Model(&models.Admin{}).Where("id", a.ID).ForceDelete()
		}
	})
}

func oidcTokenCount(t *testing.T, adminID uint) int64 {
	t.Helper()
	n, err := facades.Orm().Query().Model(&models.PersonalAccessToken{}).
		Where("tokenable_type", "admin").Where("tokenable_id", adminID).Count()
	require.NoError(t, err)
	return n
}

func oidcRedirectLocation(t *testing.T, path string) *url.URL {
	t.Helper()
	resp, err := (&tests.TestCase{}).Http(t).Get(path)
	require.NoError(t, err)
	content, _ := resp.Content()
	resp.AssertStatus(nethttp.StatusFound)
	loc := resp.Headers().Get("Location")
	require.NotEmpty(t, loc, content)
	u, err := url.Parse(loc)
	require.NoError(t, err)
	return u
}

// startOIDCLogin hits the redirect endpoint and returns the state sent to the IdP.
func startOIDCLogin(t *testing.T, idp *fakeOIDCIdP) string {
	t.Helper()
	u := oidcRedirectLocation(t, "/api/admin/auth/oidc/redirect")
	require.Equal(t, idp.srv.URL+"/authorize", u.Scheme+"://"+u.Host+u.Path)
	assert.Equal(t, fakeOIDCClientID, u.Query().Get("client_id"))
	state := u.Query().Get("state")
	require.NotEmpty(t, state)
	return state
}

// finishOIDCLogin hits the callback and returns the token and error code handed to the SPA.
func finishOIDCLogin(t *testing.T, code, state string) (token, errCode string) {
	t.Helper()
	q := url.Values{"code": {code}, "state": {state}}
	u := oidcRedirectLocation(t, "/api/admin/auth/oidc/callback?"+q.Encode())
	return u.Query().Get("token"), u.Query().Get("error")
}

func verifiedClaims(admin models.Admin) map[string]any {
	return map[string]any{"sub": "sub-" + admin.Username, "email": admin.Email, "email_verified": true}
}

func TestOIDCLoginVerifiedEmailIssuesToken(t *testing.T) {
	idp := newFakeOIDCIdP(t)
	withOIDCConfig(t, idp, false, false)
	admin := createOIDCAdmin(t, 1)

	state := startOIDCLogin(t, idp)
	token, errCode := finishOIDCLogin(t, idp.issueCode(verifiedClaims(admin)), state)

	assert.Empty(t, errCode)
	assert.NotEmpty(t, token)
	assert.Equal(t, int64(1), oidcTokenCount(t, admin.ID))
}

func TestOIDCLoginEmailMatchIsCaseInsensitive(t *testing.T) {
	idp := newFakeOIDCIdP(t)
	withOIDCConfig(t, idp, false, false)
	admin := createOIDCAdmin(t, 1)

	claims := verifiedClaims(admin)
	claims["email"] = "  " + upperASCII(admin.Email) + " "
	token, errCode := finishOIDCLogin(t, idp.issueCode(claims), startOIDCLogin(t, idp))

	assert.Empty(t, errCode)
	assert.NotEmpty(t, token)
}

func TestOIDCLoginRejectsUnverifiedEmail(t *testing.T) {
	idp := newFakeOIDCIdP(t)
	withOIDCConfig(t, idp, false, false)
	admin := createOIDCAdmin(t, 1)

	for name, verified := range map[string]any{
		"false":        false,
		"string false": "false",
		"missing":      nil,
	} {
		t.Run(name, func(t *testing.T) {
			claims := verifiedClaims(admin)
			claims["email_verified"] = verified
			token, errCode := finishOIDCLogin(t, idp.issueCode(claims), startOIDCLogin(t, idp))
			assert.Empty(t, token)
			assert.Equal(t, "oidc_email_unverified", errCode)
		})
	}
	assert.Equal(t, int64(0), oidcTokenCount(t, admin.ID), "no token may be issued for an unverified email")
}

func TestOIDCLoginAllowUnverifiedSwitch(t *testing.T) {
	idp := newFakeOIDCIdP(t)
	withOIDCConfig(t, idp, false, true)
	admin := createOIDCAdmin(t, 1)

	claims := verifiedClaims(admin)
	claims["email_verified"] = nil
	token, errCode := finishOIDCLogin(t, idp.issueCode(claims), startOIDCLogin(t, idp))

	assert.Empty(t, errCode)
	assert.NotEmpty(t, token)
}

func TestOIDCLoginStateIsSingleUse(t *testing.T) {
	idp := newFakeOIDCIdP(t)
	withOIDCConfig(t, idp, false, false)
	admin := createOIDCAdmin(t, 1)

	state := startOIDCLogin(t, idp)
	token, errCode := finishOIDCLogin(t, idp.issueCode(verifiedClaims(admin)), state)
	require.Empty(t, errCode)
	require.NotEmpty(t, token)

	token, errCode = finishOIDCLogin(t, idp.issueCode(verifiedClaims(admin)), state)
	assert.Empty(t, token)
	assert.Equal(t, "oidc_invalid_state", errCode)
	assert.Equal(t, int64(1), oidcTokenCount(t, admin.ID))
}

// Pull is Get+Forget, so two concurrent callbacks can both read the state; the
// set-if-absent "used" marker is what lets only one through. That race is too narrow
// to hit reliably over HTTP, so simulate the loser: the state is still readable but
// another callback has already claimed it.
func TestOIDCLoginRejectsAlreadyClaimedState(t *testing.T) {
	idp := newFakeOIDCIdP(t)
	withOIDCConfig(t, idp, false, false)
	admin := createOIDCAdmin(t, 1)

	state := startOIDCLogin(t, idp)
	usedKey := "oidc:state-claimed:" + state
	require.True(t, facades.Cache().Add(usedKey, 1, time.Minute))
	t.Cleanup(func() { facades.Cache().Forget(usedKey) })

	token, errCode := finishOIDCLogin(t, idp.issueCode(verifiedClaims(admin)), state)
	assert.Empty(t, token)
	assert.Equal(t, "oidc_invalid_state", errCode)
	assert.Equal(t, int64(0), oidcTokenCount(t, admin.ID))
}

// The "already claimed" marker must not be redeemable as a state itself: passing
// state=used:<spent state> once read the marker ("1" on Redis), which parsed as a
// valid legacy state and let a spent state log in again.
func TestOIDCLoginRejectsClaimMarkerAsState(t *testing.T) {
	idp := newFakeOIDCIdP(t)
	withOIDCConfig(t, idp, false, false)
	admin := createOIDCAdmin(t, 1)

	state := startOIDCLogin(t, idp)
	token, errCode := finishOIDCLogin(t, idp.issueCode(verifiedClaims(admin)), state)
	require.Empty(t, errCode)
	require.NotEmpty(t, token)

	token, errCode = finishOIDCLogin(t, idp.issueCode(verifiedClaims(admin)), "used:"+state)
	assert.Empty(t, token)
	assert.Equal(t, "oidc_invalid_state", errCode)
	assert.Equal(t, int64(1), oidcTokenCount(t, admin.ID))
}

func TestOIDCLoginRejectsUnknownState(t *testing.T) {
	idp := newFakeOIDCIdP(t)
	withOIDCConfig(t, idp, false, false)
	admin := createOIDCAdmin(t, 1)

	token, errCode := finishOIDCLogin(t, idp.issueCode(verifiedClaims(admin)), "never-issued-state")
	assert.Empty(t, token)
	assert.Equal(t, "oidc_invalid_state", errCode)
}

func TestOIDCLoginRejectsTokenForOtherClient(t *testing.T) {
	idp := newFakeOIDCIdP(t)
	withOIDCConfig(t, idp, false, false)
	admin := createOIDCAdmin(t, 1)

	claims := verifiedClaims(admin)
	claims["aud"] = "some-other-app"
	token, errCode := finishOIDCLogin(t, idp.issueCode(claims), startOIDCLogin(t, idp))
	assert.Empty(t, token)
	assert.Equal(t, "oidc_id_token_invalid", errCode)
}

func TestOIDCLoginRejectsDisabledAdmin(t *testing.T) {
	idp := newFakeOIDCIdP(t)
	withOIDCConfig(t, idp, false, false)
	admin := createOIDCAdmin(t, 0)

	token, errCode := finishOIDCLogin(t, idp.issueCode(verifiedClaims(admin)), startOIDCLogin(t, idp))
	assert.Empty(t, token)
	assert.Equal(t, apperrors.ErrAccountDisabled.Code, errCode)
	assert.Equal(t, int64(0), oidcTokenCount(t, admin.ID))
}

func TestOIDCLoginUnknownEmail(t *testing.T) {
	email := fmt.Sprintf("oidc-new-%d@example.test", time.Now().UnixNano())
	claims := map[string]any{"sub": "new-sub", "email": email, "email_verified": true, "preferred_username": "oidc new"}
	cleanupOIDCAdmin(t, email)

	t.Run("not linked without auto provision", func(t *testing.T) {
		idp := newFakeOIDCIdP(t)
		withOIDCConfig(t, idp, false, false)
		token, errCode := finishOIDCLogin(t, idp.issueCode(claims), startOIDCLogin(t, idp))
		assert.Empty(t, token)
		assert.Equal(t, "oidc_user_not_linked", errCode)
		n, err := facades.Orm().Query().Model(&models.Admin{}).Where("email", email).Count()
		require.NoError(t, err)
		assert.Zero(t, n)
	})

	t.Run("auto provision creates a disabled admin", func(t *testing.T) {
		idp := newFakeOIDCIdP(t)
		withOIDCConfig(t, idp, true, false)
		token, errCode := finishOIDCLogin(t, idp.issueCode(claims), startOIDCLogin(t, idp))
		assert.Empty(t, token)
		assert.Equal(t, apperrors.ErrAccountDisabled.Code, errCode)

		var created models.Admin
		require.NoError(t, facades.Orm().Query().Where("email", email).First(&created))
		require.NotZero(t, created.ID)
		assert.Equal(t, uint8(0), created.Status)
		assert.True(t, strings.HasPrefix(created.Username, "oidcnew"), created.Username)
	})
}

func upperASCII(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'a' && c <= 'z' {
			b[i] = c - 32
		}
	}
	return string(b)
}
