package services

import "testing"

func TestValidateCodegenNames(t *testing.T) {
	if err := validateCodegenNames("Article", "articles"); err != nil {
		t.Fatalf("valid names rejected: %v", err)
	}
	if err := validateCodegenNames("my-module", "my_modules"); err != nil {
		t.Fatalf("hyphen module should normalize: %v", err)
	}
	for _, tc := range []struct{ mod, table string }{
		{"", "articles"},
		{"../etc", "articles"},
		{"foo/bar", "articles"},
		{"article", "articles;drop"},
		{"article", "../users"},
		{"article.evil", "articles"},
	} {
		if err := validateCodegenNames(tc.mod, tc.table); err == nil {
			t.Fatalf("expected reject for module=%q table=%q", tc.mod, tc.table)
		}
	}
}
