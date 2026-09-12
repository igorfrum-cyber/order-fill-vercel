package httpapi

import "testing"

func TestJobUploadContentTypeIgnoresClient(t *testing.T) {
	t.Parallel()
	got := jobUploadContentType("order.xlsx", "text/html")
	if got != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		t.Fatalf("xlsx: %q", got)
	}
	got = jobUploadContentType("macro.xlsm", "application/javascript")
	if got != "application/vnd.ms-excel.sheet.macroEnabled.12" {
		t.Fatalf("xlsm: %q", got)
	}
	got = jobUploadContentType("notes.bin", "text/html")
	if got != "application/octet-stream" {
		t.Fatalf("unknown: %q", got)
	}
}
