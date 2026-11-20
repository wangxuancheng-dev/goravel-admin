package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/goravel/framework/facades"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/models"
)

// Platform "sign in as tenant admin" (support impersonation).
//
// Flow: a platform owner asks for a one-time ticket (IssueImpersonationTicket), the
// console opens the tenant admin SPA at /login/impersonate?ticket=..., and that page
// redeems the ticket (RedeemImpersonationTicket) for a normal admin token whose name
// marks it as an impersonation session. The token itself never appears in a URL.
const (
	// ImpersonationTokenPrefix starts personal_access_tokens.name for impersonation
	// sessions: "impersonate:<platform_admin_id>:<platform_username>".
	ImpersonationTokenPrefix = "impersonate:"

	impersonationTicketTTL     = 60 * time.Second
	impersonationTicketPrefix  = "platform:impersonate:ticket:"
	impersonationTicketUsedKey = "platform:impersonate:used:"
	impersonationReasonMaxLen  = 200
	defaultImpersonationTTLMin = 30
)

// ImpersonationTicket is what the platform hands the tenant admin SPA.
type ImpersonationTicket struct {
	Ticket     string `json:"ticket"`
	TenantCode string `json:"tenant_code"`
	ExpiresIn  int64  `json:"expires_in"`
	// URL opens the tenant admin SPA landing page (absolute for custom/sub domains,
	// otherwise a same-origin path).
	URL string `json:"url"`
	// Host is the tenant's own host (custom domain or subdomain), empty when the tenant
	// is picked by tenant_code on the console's origin. Path is the landing page with
	// its query. The console joins them with its own scheme and port, so the link also
	// works over http and on dev ports (URL always says https).
	Host string `json:"host"`
	Path string `json:"path"`
}

type impersonationTicketPayload struct {
	TenantID         uint   `json:"tenant_id"`
	TenantCode       string `json:"tenant_code"`
	AdminID          uint   `json:"admin_id"`
	PlatformAdminID  uint   `json:"platform_admin_id"`
	PlatformUsername string `json:"platform_username"`
	Reason           string `json:"reason"`
}

// ImpersonationInfo describes an active impersonation session (from the token name).
type ImpersonationInfo struct {
	PlatformAdminID  uint       `json:"platform_admin_id"`
	PlatformUsername string     `json:"platform_username"`
	ExpiresAt        *time.Time `json:"expires_at"`
}

// ImpersonationTTL is the hard session length for impersonation tokens.
func ImpersonationTTL() time.Duration {
	minutes := facades.Config().GetInt("tenancy.impersonate_ttl_minutes", defaultImpersonationTTLMin)
	if minutes <= 0 {
		minutes = defaultImpersonationTTLMin
	}
	if minutes > 240 {
		minutes = 240
	}
	return time.Duration(minutes) * time.Minute
}

// IsImpersonationTokenName reports whether a token name marks an impersonation session.
func IsImpersonationTokenName(name string) bool {
	return strings.HasPrefix(name, ImpersonationTokenPrefix)
}

// ImpersonationTokenName builds the token name stored for an impersonation session.
func ImpersonationTokenName(platformAdminID uint, platformUsername string) string {
	name := fmt.Sprintf("%s%d:%s", ImpersonationTokenPrefix, platformAdminID, strings.TrimSpace(platformUsername))
	if len(name) > 255 {
		name = name[:255]
	}
	return name
}

// ParseImpersonationToken returns session info when the token is an impersonation token.
func ParseImpersonationToken(token *models.PersonalAccessToken) *ImpersonationInfo {
	if token == nil || !IsImpersonationTokenName(token.Name) {
		return nil
	}
	rest := strings.TrimPrefix(token.Name, ImpersonationTokenPrefix)
	idPart, username, _ := strings.Cut(rest, ":")
	id, _ := strconv.ParseUint(idPart, 10, 64)
	return &ImpersonationInfo{
		PlatformAdminID:  uint(id),
		PlatformUsername: username,
		ExpiresAt:        token.ExpiresAt,
	}
}

// IssueImpersonationTicket validates the target and stores a one-time ticket.
//
//tenantlint:allow cache: ticket is issued on the platform and redeemed in a tenant request; payload carries and is checked against tenant_id
func IssueImpersonationTicket(tenant *models.Tenant, adminID, platformAdminID uint, platformUsername, reason string) (*ImpersonationTicket, *models.Admin, error) {
	if tenant == nil || adminID == 0 || platformAdminID == 0 {
		return nil, nil, apperrors.ErrInvalidArgument
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, nil, apperrors.ErrInvalidArgument.WithMessage("reason required")
	}
	if len([]rune(reason)) > impersonationReasonMaxLen {
		reason = string([]rune(reason)[:impersonationReasonMaxLen])
	}
	if tenant.Status != models.TenantStatusActive {
		return nil, nil, apperrors.ErrTenantDisabled
	}
	if !tenant.IsProvisionReady() {
		return nil, nil, apperrors.ErrTenantConnectionFailed.WithMessage("tenant not provisioned")
	}
	// The Tenant middleware rejects every admin request in maintenance, so the ticket
	// could never be redeemed; say so here instead of in the new tab.
	if tenant.Maintenance {
		return nil, nil, apperrors.ErrTenantMaintenance
	}

	ctx, err := bindTenantSupportCtx(tenant)
	if err != nil {
		return nil, nil, err
	}
	var target models.Admin
	if err := appfacades.OrmQuery(ctx).Where("id", adminID).First(&target); err != nil {
		return nil, nil, err
	}
	if target.ID == 0 {
		return nil, nil, apperrors.ErrAdminNotFound
	}
	if target.Status != 1 {
		return nil, nil, apperrors.ErrAccountDisabled
	}

	ticket, err := randomImpersonationTicket()
	if err != nil {
		return nil, nil, err
	}
	raw, err := json.Marshal(impersonationTicketPayload{
		TenantID:         tenant.ID,
		TenantCode:       tenant.Code,
		AdminID:          target.ID,
		PlatformAdminID:  platformAdminID,
		PlatformUsername: strings.TrimSpace(platformUsername),
		Reason:           reason,
	})
	if err != nil {
		return nil, nil, err
	}
	if err := facades.Cache().Put(impersonationTicketPrefix+hashImpersonationTicket(ticket), string(raw), impersonationTicketTTL); err != nil {
		return nil, nil, err
	}
	links := BuildTenantLoginLinks(tenant)
	host, path := ImpersonationTarget(links, ticket)
	return &ImpersonationTicket{
		Ticket:     ticket,
		TenantCode: tenant.Code,
		ExpiresIn:  int64(impersonationTicketTTL / time.Second),
		URL:        BuildImpersonationURL(links, ticket),
		Host:       host,
		Path:       path,
	}, &target, nil
}

// RedeemImpersonationTicket consumes a ticket for the tenant bound to this request.
//
//tenantlint:allow cache: ticket keys are global by design; tenant_id in the payload must match the request tenant
func RedeemImpersonationTicket(ticket string, requestTenantID uint) (*impersonationTicketPayload, error) {
	ticket = strings.TrimSpace(ticket)
	if ticket == "" || requestTenantID == 0 {
		return nil, apperrors.ErrImpersonationTicketInvalid
	}
	hash := hashImpersonationTicket(ticket)
	// Add is set-if-absent: a second redeem of the same ticket (replay or a racing
	// double submit) fails here even if the cache delete below is not atomic.
	if !facades.Cache().Add(impersonationTicketUsedKey+hash, "1", impersonationTicketTTL) {
		return nil, apperrors.ErrImpersonationTicketInvalid
	}
	raw, _ := facades.Cache().Pull(impersonationTicketPrefix+hash, "").(string)
	if raw == "" {
		return nil, apperrors.ErrImpersonationTicketInvalid
	}
	var payload impersonationTicketPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, apperrors.ErrImpersonationTicketInvalid
	}
	if payload.TenantID != requestTenantID || payload.AdminID == 0 || payload.PlatformAdminID == 0 {
		return nil, apperrors.ErrImpersonationTicketInvalid
	}
	return &payload, nil
}

// BuildImpersonationURL points the tenant admin login link at the impersonation landing page.
func BuildImpersonationURL(links TenantLoginLinks, ticket string) string {
	host, path := ImpersonationTarget(links, ticket)
	if host == "" {
		return path
	}
	return "https://" + host + path
}

// ImpersonationTarget picks where the landing page opens. Custom domains resolve the
// tenant in any resolver mode; a subdomain only when the resolver uses it. Otherwise
// host is empty: stay on the console's origin and let tenant_code pick the tenant
// (header mode, or subdomain mode without TENANCY_BASE_DOMAIN). tenant_code is always
// sent; on a tenant host it names the same tenant, so it never conflicts.
func ImpersonationTarget(links TenantLoginLinks, ticket string) (host, path string) {
	q := url.Values{}
	q.Set("ticket", ticket)
	if links.TenantCode != "" {
		q.Set("tenant_code", links.TenantCode)
	}
	path = "/login/impersonate?" + q.Encode()

	base := strings.TrimSpace(links.PrimaryCustomURL)
	if base == "" && links.Resolver == "subdomain" {
		base = strings.TrimSpace(links.SubdomainURL)
	}
	if base == "" {
		return "", path
	}
	u, err := url.Parse(base)
	if err != nil || u.Host == "" {
		return "", path
	}
	return u.Host, path
}

func randomImpersonationTicket() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashImpersonationTicket(ticket string) string {
	sum := sha256.Sum256([]byte(ticket))
	return hex.EncodeToString(sum[:])
}
