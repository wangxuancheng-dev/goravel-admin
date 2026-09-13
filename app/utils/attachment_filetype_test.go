package utils

import (
	"bytes"
	"testing"

	apperrors "goravel/app/errors"
)

func TestValidateAttachmentExtension(t *testing.T) {
	if err := ValidateAttachmentExtension("photo.jpg"); err != nil {
		t.Fatalf("expected jpg allowed, got %v", err)
	}
	if err := ValidateAttachmentExtension("shell.php.jpg"); err == nil {
		t.Fatal("expected double extension rejected")
	}
	if err := ValidateAttachmentExtension("x.svg"); err == nil {
		t.Fatal("expected svg rejected")
	}
}

func TestValidateAttachmentFile_JPEG(t *testing.T) {
	data := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F'}
	mime, err := ValidateAttachmentFile("a.jpg", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mime != "image/jpeg" {
		t.Fatalf("expected image/jpeg, got %s", mime)
	}
}

func TestValidateAttachmentFile_PHPDisguised(t *testing.T) {
	data := []byte("<?php echo 1;")
	_, err := ValidateAttachmentFile("a.jpg", data)
	if err == nil {
		t.Fatal("expected php content rejected")
	}
	be, ok := apperrors.GetBusinessError(err)
	if !ok || be.Code != apperrors.ErrAttachmentFileTypeNotAllowed.Code {
		t.Fatalf("expected attachment_file_type_not_allowed, got %v", err)
	}
}

func TestValidateAttachmentFile_MismatchExtension(t *testing.T) {
	pngHeader := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	data := append(pngHeader, bytes.Repeat([]byte{0}, 8)...)
	_, err := ValidateAttachmentFile("a.jpg", data)
	if err == nil {
		t.Fatal("expected png-as-jpg rejected")
	}
	be, ok := apperrors.GetBusinessError(err)
	if !ok || be.Code != apperrors.ErrAttachmentFileContentMismatch.Code {
		t.Fatalf("expected attachment_file_content_mismatch, got %v", err)
	}
}

func TestValidateAttachmentFile_Zip(t *testing.T) {
	data := append([]byte("PK\x03\x04"), bytes.Repeat([]byte{0}, 8)...)
	mime, err := ValidateAttachmentFile("archive.zip", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mime != "application/zip" {
		t.Fatalf("expected application/zip, got %s", mime)
	}
}

func TestValidateAttachmentFile_RAR(t *testing.T) {
	data := append([]byte("Rar!\x1a\x07\x00"), bytes.Repeat([]byte{0}, 8)...)
	mime, err := ValidateAttachmentFile("archive.rar", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mime != "application/vnd.rar" {
		t.Fatalf("expected application/vnd.rar, got %s", mime)
	}
}

func TestValidateAttachmentFile_MP3(t *testing.T) {
	data := append([]byte("ID3"), bytes.Repeat([]byte{0}, 8)...)
	mime, err := ValidateAttachmentFile("song.mp3", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mime != "audio/mpeg" {
		t.Fatalf("expected audio/mpeg, got %s", mime)
	}
}

func TestValidateAttachmentFile_JSON(t *testing.T) {
	data := []byte(`{"name":"test","note":"<script>alert(1)</script>"}`)
	mime, err := ValidateAttachmentFile("config.json", data)
	if err != nil {
		t.Fatalf("json with script string in value should be allowed: %v", err)
	}
	if mime != "application/json" {
		t.Fatalf("expected application/json, got %s", mime)
	}
}

func TestValidateAttachmentFile_Markdown(t *testing.T) {
	data := []byte("# Title\n\nSome **markdown** content.")
	mime, err := ValidateAttachmentFile("readme.md", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mime != "text/markdown" {
		t.Fatalf("expected text/markdown, got %s", mime)
	}
}

func TestValidateAttachmentFile_APK(t *testing.T) {
	data := append([]byte("PK\x03\x04"), bytes.Repeat([]byte{0}, 8)...)
	mime, err := ValidateAttachmentFile("app.apk", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mime != "application/vnd.android.package-archive" {
		t.Fatalf("expected application/vnd.android.package-archive, got %s", mime)
	}
}

func TestValidateAttachmentExtension_Archive(t *testing.T) {
	for _, name := range []string{"a.zip", "b.rar", "c.7z", "d.mp3", "e.flac", "f.json", "g.md", "h.apk", "i.yaml"} {
		if err := ValidateAttachmentExtension(name); err != nil {
			t.Fatalf("%s should be allowed: %v", name, err)
		}
	}
}

func TestAttachmentInlinePreviewSafe(t *testing.T) {
	if !AttachmentInlinePreviewSafe("image/jpeg", "image") {
		t.Fatal("jpeg should be inline safe")
	}
	if AttachmentInlinePreviewSafe("image/svg+xml", "image") {
		t.Fatal("svg should not be inline safe")
	}
	if AttachmentInlinePreviewSafe("text/html", "document") {
		t.Fatal("html should not be inline safe")
	}
	if !AttachmentInlinePreviewSafe("application/pdf", "document") {
		t.Fatal("pdf should be inline safe")
	}
	if !AttachmentInlinePreviewSafe("text/plain", "document") {
		t.Fatal("plain text should be inline safe")
	}
	if !AttachmentInlinePreviewSafe("video/mp4", "video") {
		t.Fatal("mp4 should be inline safe")
	}
}

func TestAttachmentPreviewDisposition(t *testing.T) {
	if got := AttachmentPreviewDisposition("image/png", "image", "inline"); got != "inline" {
		t.Fatalf("expected inline, got %s", got)
	}
	if got := AttachmentPreviewDisposition("text/html", "document", "inline"); got != "attachment" {
		t.Fatalf("expected attachment for html, got %s", got)
	}
}
