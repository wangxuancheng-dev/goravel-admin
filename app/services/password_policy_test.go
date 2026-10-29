package services

import (
	"reflect"
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
		if !ok || be.Code != apperrors.ErrPasswordTooShort.Code {
			t.Fatalf("want password_too_short, got %v", err)
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
		if err := ValidatePasswordAgainstPolicy("kestrel7", policy); err != nil {
			t.Fatalf("unexpected: %v", err)
		}
	})

	t.Run("require special", func(t *testing.T) {
		p := policy
		p.RequireSpecial = true
		if err := ValidatePasswordAgainstPolicy("kestrel7", p); err == nil {
			t.Fatal("expected special char error")
		}
		if err := ValidatePasswordAgainstPolicy("kestrel7!", p); err != nil {
			t.Fatalf("unexpected: %v", err)
		}
	})

	t.Run("low strength policy allows simple password", func(t *testing.T) {
		low := PasswordPolicy{MinLength: 4, RequireLetter: false, RequireNumber: false, RequireSpecial: false}
		if err := ValidatePasswordAgainstPolicy("zk7m", low); err != nil {
			t.Fatalf("unexpected: %v", err)
		}
		// Common passwords are rejected even under a relaxed policy.
		for _, pw := range []string{"1234", "abcd"} {
			if err := ValidatePasswordAgainstPolicy(pw, low); err == nil {
				t.Fatalf("%q should be rejected as common", pw)
			}
		}
	})
}

func passwordErrCode(err error) string {
	if be, ok := apperrors.GetBusinessError(err); ok {
		return be.Code
	}
	if err != nil {
		return err.Error()
	}
	return ""
}

func TestValidatePasswordStrictPolicy(t *testing.T) {
	strict := PasswordPolicy{MinLength: 8, RequireLetter: true, RequireNumber: true, RequireSpecial: true}
	cases := []struct {
		password string
		want     string
		missing  apperrors.TransList
	}{
		{"Kx7#", apperrors.ErrPasswordTooShort.Code, nil},
		{"abc", apperrors.ErrPasswordRequirementsUnmet.Code, apperrors.TransList{"password_char_number", "password_char_special"}},
		{"12345678!", apperrors.ErrPasswordMissingChars.Code, apperrors.TransList{"password_char_letter"}},
		{"kestrelhawk", apperrors.ErrPasswordMissingChars.Code, apperrors.TransList{"password_char_number", "password_char_special"}},
		{"Kestrel!!", apperrors.ErrPasswordMissingChars.Code, apperrors.TransList{"password_char_number"}},
		{"Kestrel77", apperrors.ErrPasswordMissingChars.Code, apperrors.TransList{"password_char_special"}},
		// Meet every rule but are top brute-force guesses.
		{"Admin@123", apperrors.ErrPasswordTooCommon.Code, nil},
		{"P@ssw0rd1", apperrors.ErrPasswordTooCommon.Code, nil},
		{"Password1!", apperrors.ErrPasswordTooCommon.Code, nil},
		{"Qwer1234!", apperrors.ErrPasswordTooCommon.Code, nil},
		{"Aa123456!", apperrors.ErrPasswordTooCommon.Code, nil},
		{"kestrel7!", "", nil}, // no upper/lower-case requirement
		{"Abc@2024abc", "", nil},
		{"Zq9#mLp2", "", nil},
		{"Xk#93847261", "", nil},
	}
	for _, c := range cases {
		err := ValidatePasswordAgainstPolicy(c.password, strict)
		if got := passwordErrCode(err); got != c.want {
			t.Errorf("%q: got %q, want %q", c.password, got, c.want)
			continue
		}
		if c.missing != nil {
			be, _ := apperrors.GetBusinessError(err)
			if got, _ := be.Params["missing"].(apperrors.TransList); !reflect.DeepEqual(got, c.missing) {
				t.Errorf("%q: missing %v, want %v", c.password, got, c.missing)
			}
		}
	}
}

func TestValidateAccountPasswordRejectsUsername(t *testing.T) {
	strict := PasswordPolicy{MinLength: 8, RequireLetter: true, RequireNumber: true, RequireSpecial: true}
	if got := passwordErrCode(ValidateAccountPassword("Zhangsan#2024", "zhangsan", strict)); got != apperrors.ErrPasswordContainsUsername.Code {
		t.Fatalf("got %q", got)
	}
	if got := passwordErrCode(ValidateAccountPassword("Zh@ngs4n#2024", "zhangsan", strict)); got != apperrors.ErrPasswordContainsUsername.Code {
		t.Fatalf("leet variant: got %q", got)
	}
	// Very short usernames are not checked (would block too much).
	if err := ValidateAccountPassword("Kestrel7!", "ke", strict); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestHasSequentialDigits(t *testing.T) {
	for s, want := range map[string]bool{"1234": true, "x9876y": true, "8888": true, "123": false, "1357": false, "12a34": false, "9384726": false} {
		if got := hasSequentialDigits(s); got != want {
			t.Errorf("%q: got %v want %v", s, got, want)
		}
	}
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
