package services

import (
	"testing"

	apperrors "goravel/app/errors"
)

func TestValidatePasswordAgainstPolicy(t *testing.T) {
	policy := PasswordPolicy{
		MinLength:      8,
		RequireLetter:  true,
		RequireNumber:  true,
		RequireSpecial: false,
	}

	t.Run("too short", func(t *testing.T) {
		err := ValidatePasswordAgainstPolicy("Ab1", policy)
		be, ok := apperrors.GetBusinessError(err)
		if !ok || be.Code != apperrors.ErrPasswordTooWeak.Code {
			t.Fatalf("want password_too_weak, got %v", err)
		}
	})

	t.Run("missing letter", func(t *testing.T) {
		err := ValidatePasswordAgainstPolicy("12345678", policy)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("missing number", func(t *testing.T) {
		err := ValidatePasswordAgainstPolicy("abcdefgh", policy)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("ok without special", func(t *testing.T) {
		if err := ValidatePasswordAgainstPolicy("abcdefg1", policy); err != nil {
			t.Fatalf("unexpected: %v", err)
		}
	})

	t.Run("require special", func(t *testing.T) {
		p := policy
		p.RequireSpecial = true
		if err := ValidatePasswordAgainstPolicy("abcdefg1", p); err == nil {
			t.Fatal("expected special char error")
		}
		if err := ValidatePasswordAgainstPolicy("abcdefg1!", p); err != nil {
			t.Fatalf("unexpected: %v", err)
		}
	})

	t.Run("low strength policy allows simple password", func(t *testing.T) {
		low := PasswordPolicy{MinLength: 4, RequireLetter: false, RequireNumber: false, RequireSpecial: false}
		if err := ValidatePasswordAgainstPolicy("1234", low); err != nil {
			t.Fatalf("unexpected: %v", err)
		}
		if err := ValidatePasswordAgainstPolicy("abcd", low); err != nil {
			t.Fatalf("unexpected: %v", err)
		}
	})
}

func TestMatchSensitiveConfirm(t *testing.T) {
	t.Run("empty code", func(t *testing.T) {
		err := MatchSensitiveConfirm(SensitiveConfirmInput{})
		be, ok := apperrors.GetBusinessError(err)
		if !ok || be.Code != apperrors.ErrSensitiveConfirmRequired.Code {
			t.Fatalf("want required, got %v", err)
		}
	})

	t.Run("2fa valid", func(t *testing.T) {
		err := MatchSensitiveConfirm(SensitiveConfirmInput{
			Has2FA:       true,
			ConfirmCode:  "123456",
			GoogleSecret: "secret",
			VerifyTOTP: func(secret, code string) bool {
				return secret == "secret" && code == "123456"
			},
		})
		if err != nil {
			t.Fatalf("unexpected: %v", err)
		}
	})

	t.Run("2fa invalid", func(t *testing.T) {
		err := MatchSensitiveConfirm(SensitiveConfirmInput{
			Has2FA:      true,
			ConfirmCode: "000000",
			VerifyTOTP:  func(secret, code string) bool { return false },
		})
		be, ok := apperrors.GetBusinessError(err)
		if !ok || be.Code != apperrors.ErrSensitiveConfirmInvalid.Code {
			t.Fatalf("want invalid, got %v", err)
		}
	})

	t.Run("password valid", func(t *testing.T) {
		err := MatchSensitiveConfirm(SensitiveConfirmInput{
			Has2FA:       false,
			ConfirmCode:  "plain",
			PasswordHash: "hash",
			CheckPassword: func(plain, hash string) bool {
				return plain == "plain" && hash == "hash"
			},
		})
		if err != nil {
			t.Fatalf("unexpected: %v", err)
		}
	})

	t.Run("password invalid", func(t *testing.T) {
		err := MatchSensitiveConfirm(SensitiveConfirmInput{
			Has2FA:        false,
			ConfirmCode:   "wrong",
			PasswordHash:  "hash",
			CheckPassword: func(plain, hash string) bool { return false },
		})
		be, ok := apperrors.GetBusinessError(err)
		if !ok || be.Code != apperrors.ErrSensitiveConfirmInvalid.Code {
			t.Fatalf("want invalid, got %v", err)
		}
	})
}
