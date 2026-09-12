package xlsx

import (
	"archive/zip"
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestParsePositiveIntRejectsOverflow(t *testing.T) {
	t.Parallel()
	if _, err := parsePositiveInt("1048577"); err == nil {
		t.Fatal("expected overflow")
	}
	got, err := parsePositiveInt("1048576")
	if err != nil || got != excelMaxRow {
		t.Fatalf("got %d err=%v", got, err)
	}
}

func TestReadLimitedRejectsOversize(t *testing.T) {
	t.Parallel()
	if _, err := readLimited(strings.NewReader("hello world"), 4); err == nil {
		t.Fatal("expected oversize")
	}
	got, err := readLimited(strings.NewReader("abcd"), 4)
	if err != nil || string(got) != "abcd" {
		t.Fatalf("got %q err=%v", got, err)
	}
}

func TestReadArchiveRejectsTooManyEntries(t *testing.T) {
	t.Parallel()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for i := 0; i < maxZipEntries+1; i++ {
		entry, err := writer.Create(fmt.Sprintf("%d.txt", i))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte("x")); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := readArchive(buffer.Bytes()); err == nil {
		t.Fatal("expected too many entries")
	}
}
