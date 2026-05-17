package documents

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
)

// allowedMIME maps MIME types allowed for upload to their canonical file_type label.
var allowedMIME = map[string]string{
	"application/pdf":                                                          "pdf",
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": "docx",
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":       "xlsx",
	"application/msword":            "doc",
	"application/vnd.ms-excel":      "xls",
	"text/plain":                    "txt",
	"image/png":                     "png",
	"image/jpeg":                    "jpg",
}


// FileInfo holds validated file metadata.
type FileInfo struct {
	MIMEType string
	FileType string
	Ext      string
}

// Validate reads the first 512 bytes of data to detect the MIME type and
// checks the filename extension. Returns FileInfo or an error.
func Validate(filename string, header []byte) (FileInfo, error) {
	detected := http.DetectContentType(header)
	// Strip params (e.g. "text/plain; charset=utf-8")
	detected = strings.SplitN(detected, ";", 2)[0]
	detected = strings.TrimSpace(detected)

	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))

	// If detected is application/zip, check magic bytes + extension for OOXML.
	if detected == "application/zip" || detected == "application/octet-stream" {
		detected = resolveByExtension(ext)
	}

	// Also handle PK zip detection for Office files.
	if detected == "application/zip" {
		detected = resolveByExtension(ext)
	}

	ft, ok := allowedMIME[detected]
	if !ok {
		// Try extension fallback (e.g. plain text without BOM)
		detected = resolveByExtension(ext)
		ft, ok = allowedMIME[detected]
		if !ok {
			return FileInfo{}, fmt.Errorf("%w: %s", ErrFileTypeBlocked, ext)
		}
	}

	return FileInfo{
		MIMEType: detected,
		FileType: ft,
		Ext:      ext,
	}, nil
}

func resolveByExtension(ext string) string {
	switch ext {
	case "pdf":
		return "application/pdf"
	case "docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case "xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case "doc":
		return "application/msword"
	case "xls":
		return "application/vnd.ms-excel"
	case "txt":
		return "text/plain"
	case "png":
		return "image/png"
	case "jpg", "jpeg":
		return "image/jpeg"
	default:
		return ""
	}
}

