package services

import (
	"strings"
	"testing"

	"goravel/app/models"
)

func TestDedicatedTenantDBUsername(t *testing.T) {
	if got := DedicatedTenantDBUsername("acme"); got != "t_acme" {
		t.Fatalf("short code: %q", got)
	}
	long := strings.Repeat("a", 40)
	got := DedicatedTenantDBUsername(long)
	if len(got) != mysqlMaxUserLen {
		t.Fatalf("long code: len %d (%q)", len(got), got)
	}
	if got == DedicatedTenantDBUsername(strings.Repeat("a", 39)+"b") {
		t.Fatal("codes sharing a long prefix must not share an account")
	}
	if err := validateSQLIdent(got); err != nil {
		t.Fatalf("not a valid identifier: %v", err)
	}
}

func TestGenerateDBPassword(t *testing.T) {
	a, err := generateDBPassword()
	if err != nil {
		t.Fatal(err)
	}
	b, _ := generateDBPassword()
	if len(a) != 32 || !isAlnum(a) || a == b {
		t.Fatalf("password %q / %q", a, b)
	}
}

func TestClassifyDedicatedDBUserEnsure(t *testing.T) {
	code := "acme"
	want := DedicatedTenantDBUsername(code)

	action, _ := ClassifyDedicatedDBUserEnsure(nil, DedicatedDBUserEnsureOpts{})
	if action != DedicatedDBUserActionSkipped {
		t.Fatalf("nil: %s", action)
	}

	mk := func(host, user string) *models.Tenant {
		tn := &models.Tenant{Code: code, Host: host, Username: user}
		tn.ID = 1
		return tn
	}

	action, detail := ClassifyDedicatedDBUserEnsure(mk("10.0.0.1", ""), DedicatedDBUserEnsureOpts{})
	if action != DedicatedDBUserActionSkipped || !strings.Contains(detail, "remote") {
		t.Fatalf("remote: %s %s", action, detail)
	}

	action, _ = ClassifyDedicatedDBUserEnsure(mk("", ""), DedicatedDBUserEnsureOpts{})
	if action != DedicatedDBUserActionCreated {
		t.Fatalf("empty user: %s", action)
	}

	action, detail = ClassifyDedicatedDBUserEnsure(mk("", want), DedicatedDBUserEnsureOpts{})
	if action != DedicatedDBUserActionSkipped || !strings.Contains(detail, "--rotate") {
		t.Fatalf("dedicated without rotate: %s %s", action, detail)
	}
	action, _ = ClassifyDedicatedDBUserEnsure(mk("", want), DedicatedDBUserEnsureOpts{RotatePassword: true})
	if action != DedicatedDBUserActionRotated {
		t.Fatalf("dedicated with rotate: %s", action)
	}

	action, detail = ClassifyDedicatedDBUserEnsure(mk("", "ops"), DedicatedDBUserEnsureOpts{})
	if action != DedicatedDBUserActionSkipped || !strings.Contains(detail, "custom") {
		t.Fatalf("custom: %s %s", action, detail)
	}
	action, _ = ClassifyDedicatedDBUserEnsure(mk("", "ops"), DedicatedDBUserEnsureOpts{ForceReplaceCustom: true})
	if action != DedicatedDBUserActionCreated {
		t.Fatalf("force custom: %s", action)
	}
}
