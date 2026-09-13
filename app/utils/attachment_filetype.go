package utils

import (
	"bytes"
	"path/filepath"
	"strings"

	apperrors "goravel/app/errors"
)

// attachmentExtensionMIME 允许上传的扩展名及对应 MIME（小写，含点）
var attachmentExtensionMIME = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".webp": "image/webp",
	".mp4":  "video/mp4",
	".webm": "video/webm",
	".mov":  "video/quicktime",
	".pdf":  "application/pdf",
	".doc":  "application/msword",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".xls":  "application/vnd.ms-excel",
	".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".ppt":  "application/vnd.ms-powerpoint",
	".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	".txt":  "text/plain",
	".csv":  "text/csv",
	// 压缩包
	".zip": "application/zip",
	".rar": "application/vnd.rar",
	".7z":  "application/x-7z-compressed",
	".gz":  "application/gzip",
	".tar": "application/x-tar",
	".bz2": "application/x-bzip2",
	// 音频
	".mp3":  "audio/mpeg",
	".wav":  "audio/wav",
	".flac": "audio/flac",
	".ogg":  "audio/ogg",
	".aac":  "audio/aac",
	".m4a":  "audio/mp4",
	".wma":  "audio/x-ms-wma",
	// 文本 / 配置
	".json":      "application/json",
	".md":        "text/markdown",
	".markdown":  "text/markdown",
	".yaml":      "text/yaml",
	".yml":       "text/yaml",
	".ini":       "text/plain",
	".conf":      "text/plain",
	".cfg":       "text/plain",
	".log":       "text/plain",
	// 安装包（APK/IPA 本质为 ZIP）
	".apk": "application/vnd.android.package-archive",
	".ipa": "application/octet-stream",
}

var attachmentBlockedExtensions = map[string]struct{}{
	".php": {}, ".phtml": {}, ".php3": {}, ".php4": {}, ".php5": {}, ".phar": {},
	".html": {}, ".htm": {}, ".js": {}, ".mjs": {}, ".jsx": {},
	".ts": {}, ".tsx": {}, ".svg": {}, ".xml": {}, ".xsl": {},
	".asp": {}, ".aspx": {}, ".jsp": {}, ".sh": {}, ".bash": {},
	".bat": {}, ".cmd": {}, ".exe": {}, ".dll": {}, ".com": {},
	".jar": {}, ".war": {}, ".py": {}, ".rb": {}, ".pl": {},
	".cgi": {}, ".htaccess": {}, ".swf": {}, ".wasm": {},
}

var attachmentDangerousSnippets = []string{
	"<?php", "<script", "<html", "<!doctype", "#!/bin/", "#!/usr/bin/",
}

// ValidateAttachmentExtension 校验附件文件名扩展名（分片 init 等场景，尚无文件内容）
func ValidateAttachmentExtension(filename string) error {
	ext, err := normalizeAttachmentExtension(filename)
	if err != nil {
		return err
	}
	if _, ok := attachmentExtensionMIME[ext]; !ok {
		return apperrors.ErrAttachmentFileTypeNotAllowed
	}
	return nil
}

// ValidateAttachmentFile 校验扩展名白名单、危险特征，并用 magic bytes 校验真实类型
func ValidateAttachmentFile(filename string, data []byte) (string, error) {
	ext, err := normalizeAttachmentExtension(filename)
	if err != nil {
		return "", err
	}

	expectedMIME, ok := attachmentExtensionMIME[ext]
	if !ok {
		return "", apperrors.ErrAttachmentFileTypeNotAllowed
	}

	if len(data) == 0 {
		return "", apperrors.ErrAttachmentFileContentMismatch
	}

	if containsDangerousAttachmentContent(ext, data) {
		return "", apperrors.ErrAttachmentFileTypeNotAllowed
	}

	detected := detectAttachmentMIME(data)
	if detected == "" {
		return "", apperrors.ErrAttachmentFileContentMismatch
	}

	if !attachmentMIMECompatible(ext, expectedMIME, detected) {
		return "", apperrors.ErrAttachmentFileContentMismatch
	}

	return expectedMIME, nil
}

// AttachmentInlinePreviewSafe 是否允许 inline 预览（避免 HTML/SVG/脚本类 XSS）
func AttachmentInlinePreviewSafe(mimeType, fileType string) bool {
	mimeType = strings.ToLower(strings.TrimSpace(mimeType))
	if mimeType == "" {
		return false
	}
	if strings.Contains(mimeType, "html") ||
		strings.Contains(mimeType, "javascript") ||
		strings.Contains(mimeType, "svg") ||
		strings.Contains(mimeType, "xml") {
		return false
	}
	switch mimeType {
	case "image/jpeg", "image/png", "image/gif", "image/webp",
		"video/mp4", "video/webm", "video/quicktime",
		"application/pdf",
		"text/plain", "text/csv", "text/markdown", "text/x-markdown":
		return true
	}
	// text/* except blocked families above
	if strings.HasPrefix(mimeType, "text/") {
		return true
	}
	switch fileType {
	case "image", "video":
		return false
	default:
		return false
	}
}

// AttachmentPreviewDisposition 根据 MIME 决定预览 Content-Disposition
func AttachmentPreviewDisposition(mimeType, fileType, requested string) string {
	if requested == "attachment" {
		return "attachment"
	}
	if AttachmentInlinePreviewSafe(mimeType, fileType) {
		return "inline"
	}
	return "attachment"
}

func normalizeAttachmentExtension(filename string) (string, error) {
	base := strings.ToLower(strings.TrimSpace(filepath.Base(filename)))
	if base == "" || base == "." || base == ".." {
		return "", apperrors.ErrAttachmentFileTypeNotAllowed
	}

	// 拒绝双扩展名伪装（如 shell.php.jpg、x.html.png）
	nameWithoutExt := strings.TrimSuffix(base, filepath.Ext(base))
	for blocked := range attachmentBlockedExtensions {
		if strings.HasSuffix(nameWithoutExt, blocked) {
			return "", apperrors.ErrAttachmentFileTypeNotAllowed
		}
	}

	ext := strings.ToLower(filepath.Ext(base))
	if ext == "" {
		return "", apperrors.ErrAttachmentFileTypeNotAllowed
	}
	if _, blocked := attachmentBlockedExtensions[ext]; blocked {
		return "", apperrors.ErrAttachmentFileTypeNotAllowed
	}
	return ext, nil
}

func containsDangerousAttachmentContent(ext string, data []byte) bool {
	sample := data
	if len(sample) > 512 {
		sample = sample[:512]
	}
	trimmed := bytes.TrimSpace(sample)
	lower := strings.ToLower(string(trimmed))

	// 文本类：仅检查文件头，避免 JSON 字符串中含 "<script" 等误杀
	if isAttachmentTextExtension(ext) {
		headerSnippets := []string{"<?php", "#!/bin/", "#!/usr/bin/", "<!doctype html", "<html"}
		for _, snippet := range headerSnippets {
			if strings.HasPrefix(lower, snippet) {
				return true
			}
		}
		return false
	}

	lowerFull := strings.ToLower(string(sample))
	for _, snippet := range attachmentDangerousSnippets {
		if strings.Contains(lowerFull, snippet) {
			return true
		}
	}
	return false
}

func isAttachmentTextExtension(ext string) bool {
	switch ext {
	case ".txt", ".csv", ".json", ".md", ".markdown", ".yaml", ".yml", ".ini", ".conf", ".cfg", ".log":
		return true
	default:
		return false
	}
}

func detectAttachmentMIME(data []byte) string {
	if len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return "image/jpeg"
	}
	if len(data) >= 8 && bytes.Equal(data[:8], []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}) {
		return "image/png"
	}
	if len(data) >= 6 && (bytes.HasPrefix(data, []byte("GIF87a")) || bytes.HasPrefix(data, []byte("GIF89a"))) {
		return "image/gif"
	}
	if len(data) >= 12 && bytes.Equal(data[0:4], []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WEBP")) {
		return "image/webp"
	}
	if len(data) >= 4 && bytes.Equal(data[:4], []byte("%PDF")) {
		return "application/pdf"
	}
	if len(data) >= 12 && bytes.Equal(data[4:8], []byte("ftyp")) {
		brand := string(data[8:12])
		switch brand {
		case "qt  ":
			return "video/quicktime"
		case "M4A ", "M4B ", "F4A ", "F4B ":
			return "audio/mp4"
		default:
			return "video/mp4"
		}
	}
	if len(data) >= 4 && bytes.Equal(data[:4], []byte{0x1A, 0x45, 0xDF, 0xA3}) {
		return "video/webm"
	}
	if len(data) >= 4 && (bytes.Equal(data[:4], []byte("PK\x03\x04")) ||
		bytes.Equal(data[:4], []byte("PK\x05\x06")) ||
		bytes.Equal(data[:4], []byte("PK\x07\x08"))) {
		return "application/zip"
	}
	if len(data) >= 7 && bytes.HasPrefix(data, []byte("Rar!\x1a\x07")) {
		return "application/vnd.rar"
	}
	if len(data) >= 6 && bytes.Equal(data[:6], []byte{0x37, 0x7A, 0xBC, 0xAF, 0x27, 0x1C}) {
		return "application/x-7z-compressed"
	}
	if len(data) >= 2 && data[0] == 0x1F && data[1] == 0x8B {
		return "application/gzip"
	}
	if len(data) >= 262 && bytes.Equal(data[257:262], []byte("ustar")) {
		return "application/x-tar"
	}
	if len(data) >= 3 && bytes.Equal(data[:3], []byte("BZh")) {
		return "application/x-bzip2"
	}
	if len(data) >= 3 && bytes.Equal(data[:3], []byte("ID3")) {
		return "audio/mpeg"
	}
	if len(data) >= 2 && data[0] == 0xFF && (data[1]&0xE0) == 0xE0 {
		return "audio/mpeg"
	}
	if len(data) >= 12 && bytes.Equal(data[0:4], []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WAVE")) {
		return "audio/wav"
	}
	if len(data) >= 4 && bytes.Equal(data[:4], []byte("fLaC")) {
		return "audio/flac"
	}
	if len(data) >= 4 && bytes.Equal(data[:4], []byte("OggS")) {
		return "audio/ogg"
	}
	if len(data) >= 2 && data[0] == 0xFF && (data[1]&0xF6) == 0xF0 {
		return "audio/aac"
	}
	if len(data) >= 16 && bytes.Equal(data[0:16], []byte{0x30, 0x26, 0xB2, 0x75, 0x8E, 0x66, 0xCF, 0x11, 0xA6, 0xD9, 0x00, 0xAA, 0x00, 0x62, 0xCE, 0x6C}) {
		return "audio/x-ms-wma"
	}
	if len(data) >= 8 && bytes.Equal(data[:8], []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}) {
		return "application/x-ole-storage"
	}
	if isAttachmentJSON(data) {
		return "application/json"
	}
	if isAttachmentPlainText(data) {
		return "text/plain"
	}
	return ""
}

func isAttachmentJSON(data []byte) bool {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return false
	}
	first := trimmed[0]
	return first == '{' || first == '['
}

func attachmentMIMECompatible(ext, expectedMIME, detected string) bool {
	switch ext {
	case ".jpg", ".jpeg":
		return detected == "image/jpeg"
	case ".png":
		return detected == "image/png"
	case ".gif":
		return detected == "image/gif"
	case ".webp":
		return detected == "image/webp"
	case ".mp4":
		return detected == "video/mp4"
	case ".webm":
		return detected == "video/webm"
	case ".mov":
		return detected == "video/mp4" || detected == "video/quicktime"
	case ".pdf":
		return detected == "application/pdf"
	case ".docx", ".xlsx", ".pptx":
		return detected == "application/zip"
	case ".doc", ".xls", ".ppt":
		return detected == "application/x-ole-storage"
	case ".txt", ".csv":
		return detected == "text/plain"
	case ".zip":
		return detected == "application/zip"
	case ".rar":
		return detected == "application/vnd.rar"
	case ".7z":
		return detected == "application/x-7z-compressed"
	case ".gz":
		return detected == "application/gzip"
	case ".tar":
		return detected == "application/x-tar"
	case ".bz2":
		return detected == "application/x-bzip2"
	case ".mp3":
		return detected == "audio/mpeg"
	case ".wav":
		return detected == "audio/wav"
	case ".flac":
		return detected == "audio/flac"
	case ".ogg":
		return detected == "audio/ogg"
	case ".aac":
		return detected == "audio/aac"
	case ".m4a":
		return detected == "audio/mp4"
	case ".wma":
		return detected == "audio/x-ms-wma"
	case ".json":
		return detected == "application/json" || detected == "text/plain"
	case ".md", ".markdown", ".yaml", ".yml", ".ini", ".conf", ".cfg", ".log":
		return detected == "text/plain"
	case ".apk", ".ipa":
		return detected == "application/zip"
	default:
		return detected == expectedMIME
	}
}

func isAttachmentPlainText(data []byte) bool {
	sample := data
	if len(sample) > 512 {
		sample = sample[:512]
	}
	if len(sample) == 0 {
		return false
	}
	for _, b := range sample {
		if b == 0 {
			return false
		}
		if b < 9 && b != '\n' && b != '\r' && b != '\t' {
			return false
		}
	}
	return true
}
