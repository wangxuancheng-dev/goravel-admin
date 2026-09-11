package services

import (
	"strings"

	appfacades "goravel/app/facades"
)

// tenantSecretPrefix marks APP_KEY-encrypted tenant DB passwords at rest.
const tenantSecretPrefix = "enc:v1:"

// SealTenantPassword encrypts a tenant DB password for platform DB storage.
// Empty input stays empty. Already-sealed values are returned unchanged.
func SealTenantPassword(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	if strings.HasPrefix(plain, tenantSecretPrefix) {
		return plain, nil
	}
	enc, err := appfacades.Crypt().EncryptString(plain)
	if err != nil {
		return "", err
	}
	return tenantSecretPrefix + enc, nil
}

// RevealTenantPassword decrypts a sealed password, or returns legacy plaintext as-is.
func RevealTenantPassword(stored string) (string, error) {
	if stored == "" {
		return "", nil
	}
	if !strings.HasPrefix(stored, tenantSecretPrefix) {
		return stored, nil
	}
	return appfacades.Crypt().DecryptString(strings.TrimPrefix(stored, tenantSecretPrefix))
}

// TenantHasPassword reports whether a password credential is stored (sealed or plain).
func TenantHasPassword(stored string) bool {
	return strings.TrimSpace(stored) != ""
}
