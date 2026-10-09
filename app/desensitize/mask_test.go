package desensitize

import (
	"context"
	"testing"
)

func TestMaskPhone(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"13800138000", "138****8000"},
		{"1234567", "12****67"},
		{"12", "***"},
		{"abc", "a***"},
	}
	for _, tc := range cases {
		if got := MaskPhone(tc.in); got != tc.want {
			t.Fatalf("MaskPhone(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestMaskEmail(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"alice@example.com", "a***@example.com"},
		{"a@b.co", "a***@b.co"},
		{"nodomain", "n***"},
		{"@x.com", "***@x.com"},
	}
	for _, tc := range cases {
		if got := MaskEmail(tc.in); got != tc.want {
			t.Fatalf("MaskEmail(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestMaskStrategies(t *testing.T) {
	if got := Mask("13800138000", StrategyPhone); got != "138****8000" {
		t.Fatalf("phone strategy: %q", got)
	}
	if got := Mask("secret", StrategyHide); got != "***" {
		t.Fatalf("hide strategy: %q", got)
	}
	if got := Mask("abcdef", StrategyKeepEnds); got != "a***f" {
		t.Fatalf("keep_ends strategy: %q", got)
	}
}

func TestLooksMasked(t *testing.T) {
	if !LooksMasked("138****8000") {
		t.Fatal("expected masked phone")
	}
	if LooksMasked("13800138000") {
		t.Fatal("plain phone should not look masked")
	}
}

func TestPolicyApplyStringModes(t *testing.T) {
	p := Policy{
		Enabled: true,
		Modes:   map[string]bool{ModeList: true, ModeExport: true},
		Modules: map[string]map[string]string{
			"user": {"phone": StrategyPhone, "email": StrategyEmail},
		},
	}
	ctx := context.Background()
	if !p.ModeEnabled(ModeList) {
		t.Fatal("list mode should be enabled")
	}
	if p.ModeEnabled("detail") {
		t.Fatal("detail mode should be off")
	}
	got := p.ApplyString(ctx, "user", "phone", ModeList, "13800138000")
	if got != "138****8000" {
		t.Fatalf("ApplyString=%q", got)
	}
	got = p.ApplyString(ctx, "user", "phone", "detail", "13800138000")
	if got != "13800138000" {
		t.Fatalf("detail should stay plain: %q", got)
	}

	p.Enabled = false
	got = p.ApplyString(ctx, "user", "phone", ModeList, "13800138000")
	if got != "13800138000" {
		t.Fatalf("disabled should stay plain: %q", got)
	}
}

func TestApplyMap(t *testing.T) {
	p := Policy{
		Enabled: true,
		Modes:   map[string]bool{ModeList: true},
		Modules: map[string]map[string]string{
			"admin": {"phone": StrategyPhone, "email": StrategyEmail},
		},
	}
	data := map[string]any{
		"phone": "13800138000",
		"email": "admin@example.com",
		"name":  "bob",
	}
	p.ApplyMap(context.Background(), "admin", ModeList, data)
	if data["phone"] != "138****8000" {
		t.Fatalf("phone=%v", data["phone"])
	}
	if data["email"] != "a***@example.com" {
		t.Fatalf("email=%v", data["email"])
	}
	if data["name"] != "bob" {
		t.Fatalf("name should be untouched")
	}
}
