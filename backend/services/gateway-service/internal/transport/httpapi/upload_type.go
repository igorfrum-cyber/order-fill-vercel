package httpapi

import (
	"path/filepath"
	"strings"
)

func jobUploadContentType(name, _ string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".xlsm":
		return "application/vnd.ms-excel.sheet.macroEnabled.12"
	default:
		return "application/octet-stream"
	}
}
