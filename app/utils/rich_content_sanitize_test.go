package utils

import (
	"strings"
	"testing"
)

func TestSanitizeRichTextContent_HTML(t *testing.T) {
	raw := `<p>hello</p><script>alert(1)</script><img src=x onerror=alert(1)>`
	got := SanitizeRichTextContent(raw)
	if strings.Contains(got, "script") || strings.Contains(got, "onerror") {
		t.Fatalf("expected dangerous html removed, got %q", got)
	}
	if !strings.Contains(got, "hello") {
		t.Fatalf("expected safe content kept, got %q", got)
	}
}

func TestSanitizeRichTextContent_KeepsPublicImageTenantQuery(t *testing.T) {
	raw := `<div><img src="/api/admin/public/images/3?tenant_code=acme" alt="logo"></div>`
	got := SanitizeRichTextContent(raw)
	if !strings.Contains(got, `/api/admin/public/images/3?tenant_code=acme`) {
		t.Fatalf("expected tenant_code query kept for public img, got %q", got)
	}
}

func TestSanitizeRichTextContent_MarkdownLink(t *testing.T) {
	raw := `[click me](javascript:alert(1))`
	got := SanitizeRichTextContent(raw)
	if strings.Contains(got, "javascript:") {
		t.Fatalf("expected javascript link removed, got %q", got)
	}
}

func TestSanitizeRichTextContent_MarkdownWithRawHTML(t *testing.T) {
	raw := "# title\n\n<p onclick=\"alert(1)\">x</p>"
	got := SanitizeRichTextContent(raw)
	if strings.Contains(got, "onclick") {
		t.Fatalf("expected onclick stripped, got %q", got)
	}
}
