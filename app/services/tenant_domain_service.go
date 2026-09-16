package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/goravel/framework/facades"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/tenancy"
)

const tenantDomainCachePrefix = "tenant_domain:v1:"

type TenantDomainService struct{}

func NewTenantDomainService() *TenantDomainService {
	return &TenantDomainService{}
}

func (s *TenantDomainService) cacheTTL() time.Duration {
	sec := facades.Config().GetInt("tenancy.domain_cache_ttl", 60)
	if sec <= 0 {
		sec = 60
	}
	return time.Duration(sec) * time.Second
}

func (s *TenantDomainService) cacheKey(host string) string {
	return tenantDomainCachePrefix + tenancy.NormalizeHost(host)
}

func (s *TenantDomainService) forgetCache(host string) {
	_ = facades.Cache().Forget(s.cacheKey(host))
}

// ResolveActiveCode returns tenant code when host is an active custom domain.
func (s *TenantDomainService) ResolveActiveCode(host string) string {
	host = tenancy.NormalizeHost(host)
	if host == "" || !tenancy.Enabled() {
		return ""
	}
	key := s.cacheKey(host)
	if cached := facades.Cache().GetString(key, ""); cached != "" {
		if cached == "-" {
			return ""
		}
		return cached
	}
	row, tenant, err := s.findActiveByHost(host)
	if err != nil || row == nil || tenant == nil {
		_ = facades.Cache().Put(key, "-", s.cacheTTL())
		return ""
	}
	_ = facades.Cache().Put(key, tenant.Code, s.cacheTTL())
	return tenant.Code
}

// IsActiveHost reports whether host is an active vanity domain (for Domain middleware / CORS).
func (s *TenantDomainService) IsActiveHost(host string) bool {
	return s.ResolveActiveCode(host) != ""
}

// TLSAllow reports whether edge TLS (on-demand) may issue a cert for host.
// Only ssl_mode=edge and status=active; customer_cdn terminates TLS elsewhere.
func (s *TenantDomainService) TLSAllow(host string) bool {
	host = tenancy.NormalizeHost(host)
	if host == "" || !tenancy.Enabled() {
		return false
	}
	row, tenant, err := s.findActiveByHost(host)
	if err != nil || row == nil || tenant == nil {
		return false
	}
	if row.SSLMode != models.TenantDomainSSLEdge {
		return false
	}
	if tenant.Status != models.TenantStatusActive || tenant.Maintenance || !tenant.IsProvisionReady() {
		return false
	}
	return true
}

func (s *TenantDomainService) findActiveByHost(host string) (*models.TenantDomain, *models.Tenant, error) {
	var row models.TenantDomain
	if err := appfacades.PlatformOrmQuery(nil).
		Where("host", host).
		Where("status", models.TenantDomainStatusActive).
		First(&row); err != nil || row.ID == 0 {
		return nil, nil, err
	}
	var tenant models.Tenant
	if err := appfacades.PlatformOrmQuery(nil).Where("id", row.TenantID).First(&tenant); err != nil || tenant.ID == 0 {
		return nil, nil, err
	}
	return &row, &tenant, nil
}

func (s *TenantDomainService) ListByTenant(tenantID uint) ([]models.TenantDomain, error) {
	var list []models.TenantDomain
	err := appfacades.PlatformOrmQuery(nil).
		Where("tenant_id", tenantID).
		Order("is_primary desc, id asc").
		Find(&list)
	return list, err
}

func (s *TenantDomainService) GetByID(tenantID, id uint) (*models.TenantDomain, error) {
	var row models.TenantDomain
	if err := appfacades.PlatformOrmQuery(nil).
		Where("tenant_id", tenantID).
		Where("id", id).
		First(&row); err != nil || row.ID == 0 {
		return nil, apperrors.ErrTenantDomainNotFound
	}
	return &row, nil
}

type TenantDomainCreateInput struct {
	Host      string
	SSLMode   string
	IsPrimary bool
}

func (s *TenantDomainService) Create(tenantID uint, in TenantDomainCreateInput) (*models.TenantDomain, error) {
	host := tenancy.NormalizeHost(in.Host)
	if host == "" || strings.Contains(host, " ") || tenancy.IsReservedOrPlatformHost(host) {
		return nil, apperrors.ErrTenantDomainInvalid
	}
	ssl := strings.ToLower(strings.TrimSpace(in.SSLMode))
	if ssl == "" {
		ssl = models.TenantDomainSSLEdge
	}
	if ssl != models.TenantDomainSSLEdge && ssl != models.TenantDomainSSLCustomerCDN {
		return nil, apperrors.ErrTenantDomainInvalid.WithMessage("ssl_mode must be edge or customer_cdn")
	}

	var tenant models.Tenant
	if err := appfacades.PlatformOrmQuery(nil).Where("id", tenantID).First(&tenant); err != nil || tenant.ID == 0 {
		return nil, apperrors.ErrTenantNotFound
	}

	var existing models.TenantDomain
	if err := appfacades.PlatformOrmQuery(nil).Where("host", host).First(&existing); err == nil && existing.ID > 0 {
		return nil, apperrors.ErrTenantDomainTaken
	}

	token, err := randomHex(16)
	if err != nil {
		return nil, apperrors.ErrQueryFailed.WithError(err)
	}

	row := models.TenantDomain{
		TenantID:    tenantID,
		Host:        host,
		IsPrimary:   in.IsPrimary,
		Status:      models.TenantDomainStatusPending,
		SSLMode:     ssl,
		VerifyType:  models.TenantDomainVerifyDNSTXT,
		VerifyToken: token,
	}
	if err := appfacades.PlatformOrmQuery(nil).Create(&row); err != nil {
		return nil, apperrors.ErrQueryFailed.WithError(err)
	}
	if in.IsPrimary {
		_ = s.clearOtherPrimary(tenantID, row.ID)
	}
	s.forgetCache(host)
	return &row, nil
}

func (s *TenantDomainService) clearOtherPrimary(tenantID, keepID uint) error {
	_, err := appfacades.PlatformOrmQuery(nil).Model(&models.TenantDomain{}).
		Where("tenant_id", tenantID).
		Where("id", "!=", keepID).
		Where("is_primary", true).
		Update("is_primary", false)
	return err
}

func (s *TenantDomainService) SetPrimary(tenantID, id uint) (*models.TenantDomain, error) {
	row, err := s.GetByID(tenantID, id)
	if err != nil {
		return nil, err
	}
	if row.Status != models.TenantDomainStatusActive {
		return nil, apperrors.ErrTenantDomainNotActive
	}
	_ = s.clearOtherPrimary(tenantID, row.ID)
	if _, err := appfacades.PlatformOrmQuery(nil).Model(row).Update("is_primary", true); err != nil {
		return nil, apperrors.ErrUpdateFailed.WithError(err)
	}
	row.IsPrimary = true
	s.forgetCache(row.Host)
	return row, nil
}

func (s *TenantDomainService) Disable(tenantID, id uint) (*models.TenantDomain, error) {
	row, err := s.GetByID(tenantID, id)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if _, err := appfacades.PlatformOrmQuery(nil).Model(row).Update(map[string]any{
		"status":           models.TenantDomainStatusDisabled,
		"is_primary":       false,
		"last_check_at":    now,
		"last_check_error": "",
	}); err != nil {
		return nil, apperrors.ErrUpdateFailed.WithError(err)
	}
	row.Status = models.TenantDomainStatusDisabled
	row.IsPrimary = false
	s.forgetCache(row.Host)
	return row, nil
}

func (s *TenantDomainService) Delete(tenantID, id uint) error {
	row, err := s.GetByID(tenantID, id)
	if err != nil {
		return err
	}
	host := row.Host
	if _, err := appfacades.PlatformOrmQuery(nil).Where("id", row.ID).Delete(&models.TenantDomain{}); err != nil {
		return apperrors.ErrDeleteFailed.WithError(err)
	}
	s.forgetCache(host)
	return nil
}

// DisableAllForTenant marks all vanity domains disabled (soft-delete tenant path).
func (s *TenantDomainService) DisableAllForTenant(tenantID uint) error {
	list, err := s.ListByTenant(tenantID)
	if err != nil {
		return err
	}
	for i := range list {
		s.forgetCache(list[i].Host)
	}
	_, err = appfacades.PlatformOrmQuery(nil).Model(&models.TenantDomain{}).
		Where("tenant_id", tenantID).
		Update(map[string]any{
			"status":     models.TenantDomainStatusDisabled,
			"is_primary": false,
		})
	return err
}

// PurgeAllForTenant hard-deletes vanity domain rows and clears host cache.
func (s *TenantDomainService) PurgeAllForTenant(tenantID uint) error {
	list, err := s.ListByTenant(tenantID)
	if err != nil {
		return err
	}
	for i := range list {
		s.forgetCache(list[i].Host)
	}
	_, err = appfacades.PlatformOrmQuery(nil).Where("tenant_id", tenantID).ForceDelete(&models.TenantDomain{})
	return err
}

// Verify checks DNS and activates the domain on success.
func (s *TenantDomainService) Verify(tenantID, id uint) (*models.TenantDomain, error) {
	row, err := s.GetByID(tenantID, id)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	checkErr := s.checkDNS(row)
	updates := map[string]any{
		"last_check_at": now,
	}
	if checkErr != nil {
		updates["last_check_error"] = checkErr.Error()
		_, _ = appfacades.PlatformOrmQuery(nil).Model(row).Update(updates)
		return nil, apperrors.ErrTenantDomainVerifyFailed.WithMessage(checkErr.Error())
	}
	updates["last_check_error"] = ""
	updates["status"] = models.TenantDomainStatusActive
	updates["verified_at"] = now
	if _, err := appfacades.PlatformOrmQuery(nil).Model(row).Update(updates); err != nil {
		return nil, apperrors.ErrUpdateFailed.WithError(err)
	}
	row.Status = models.TenantDomainStatusActive
	row.VerifiedAt = &now
	row.LastCheckAt = &now
	row.LastCheckError = ""
	s.forgetCache(row.Host)
	return row, nil
}

func (s *TenantDomainService) checkDNS(row *models.TenantDomain) error {
	if row == nil {
		return fmt.Errorf("nil domain")
	}
	// TXT: {prefix}.{host} contains token (works for both ssl modes as ownership proof).
	txtName := tenancy.DomainVerifyPrefix() + "." + row.Host
	txts, err := net.LookupTXT(txtName)
	tokenOK := false
	if err == nil {
		want := row.VerifyToken
		for _, t := range txts {
			if strings.Contains(t, want) {
				tokenOK = true
				break
			}
		}
	}
	if !tokenOK {
		return fmt.Errorf("missing DNS TXT %s containing token", txtName)
	}

	switch row.SSLMode {
	case models.TenantDomainSSLEdge:
		target := tenancy.DomainTarget()
		if target == "" {
			// Ownership TXT is enough when ingress target is not configured (dev).
			return nil
		}
		cnames, cerr := net.LookupCNAME(row.Host)
		if cerr == nil {
			cnames = tenancy.NormalizeHost(strings.TrimSuffix(cnames, "."))
			if cnames == target || strings.HasSuffix(cnames, "."+target) {
				return nil
			}
		}
		// Also accept when host resolves and CNAME chain includes target via LookupHost of target equality is hard;
		// require CNAME to target when target is set.
		return fmt.Errorf("host must CNAME to %s (got %q)", target, cnames)
	case models.TenantDomainSSLCustomerCDN:
		if _, err := net.LookupHost(row.Host); err != nil {
			return fmt.Errorf("host has no A/AAAA record: %v", err)
		}
		return nil
	default:
		return fmt.Errorf("unknown ssl_mode")
	}
}

func (s *TenantDomainService) DNSGuide(row *models.TenantDomain) map[string]any {
	if row == nil {
		return map[string]any{}
	}
	txtName := tenancy.DomainVerifyPrefix() + "." + row.Host
	guide := map[string]any{
		"verify_type":  row.VerifyType,
		"txt_name":     txtName,
		"txt_value":    row.VerifyToken,
		"ssl_mode":     row.SSLMode,
		"domain_target": tenancy.DomainTarget(),
	}
	switch row.SSLMode {
	case models.TenantDomainSSLEdge:
		guide["instruction"] = "Add TXT for ownership; CNAME host to TENANCY_DOMAIN_TARGET; edge issues TLS (tls-allow)."
	case models.TenantDomainSSLCustomerCDN:
		guide["instruction"] = "Add TXT for ownership; point CDN to origin; keep Host as custom domain; TLS on customer CDN."
	}
	return guide
}

func TenantDomainToJSON(row *models.TenantDomain) map[string]any {
	if row == nil {
		return nil
	}
	return map[string]any{
		"id":               row.ID,
		"tenant_id":        row.TenantID,
		"host":             row.Host,
		"is_primary":       row.IsPrimary,
		"status":           row.Status,
		"ssl_mode":         row.SSLMode,
		"verify_type":      row.VerifyType,
		"verify_token":     row.VerifyToken,
		"verified_at":      row.VerifiedAt,
		"last_check_at":    row.LastCheckAt,
		"last_check_error": row.LastCheckError,
		"created_at":       row.CreatedAt,
		"updated_at":       row.UpdatedAt,
		"dns_guide":        NewTenantDomainService().DNSGuide(row),
	}
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
