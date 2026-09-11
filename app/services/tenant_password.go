package services

import (
	"strings"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
)

// tenantSecretPrefix marks APP_KEY-encrypted tenant DB passwords at rest.
const tenantSecretPrefix = "enc:v1:"

// SealTenantPassword encrypts a tenant DB password for platform DB storage.
func SealTenantPassword(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	if strings.HasPrefix(plain, tenantSecretPrefix) {
		return "", apperrors.ErrInvalidArgument.WithMessage("do not pass already-sealed password")
	}
	enc, err := appfacades.Crypt().EncryptString(plain)
	if err != nil {
		return "", err
	}
	return tenantSecretPrefix + enc, nil
}

// RevealTenantPassword decrypts a sealed password (enc:v1:...). Plaintext is rejected.
func RevealTenantPassword(stored string) (string, error) {
	if stored == "" {
		return "", nil
	}
	if !strings.HasPrefix(stored, tenantSecretPrefix) {
		return "", apperrors.ErrTenantPasswordCorrupt
	}
	plain, err := appfacades.Crypt().DecryptString(strings.TrimPrefix(stored, tenantSecretPrefix))
	if err != nil {
		return "", apperrors.ErrTenantPasswordCorrupt.WithError(err)
	}
	return plain, nil
}

// TenantHasPassword reports whether a password credential is stored.
func TenantHasPassword(stored string) bool {
	return strings.TrimSpace(stored) != ""
}
