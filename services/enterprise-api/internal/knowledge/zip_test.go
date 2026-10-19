package knowledge

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/text/encoding/simplifiedchinese"
)

func TestExtractZipRejectsSlipBombHiddenAndNested(t *testing.T) {
	allowed := allowedSet([]string{".txt", ".md"})
	dest := t.TempDir()

	slip := writeZip(t, map[string]string{"../secret.txt": "no"})
	if _, err := ExtractZip(slip, dest, 20, 1<<20, allowed); err == nil {
		t.Fatal("expected zip slip to fail")
	}

	bomb := writeZip(t, map[string]string{"a.txt": "12345", "b.txt": "12345"})
	if _, err := ExtractZip(bomb, dest, 20, 6, allowed); err == nil {
		t.Fatal("expected zip bomb to fail")
	}

	mixed := writeZip(t, map[string]string{
		"docs/ok.txt":     "yes",
		".hidden.txt":     "hide",
		"__MACOSX/._ok":   "mac",
		"docs/.DS_Store":  "mac",
		"nested/more.zip": "zip",
		"notes.csv":       "nope",
	})
	entries, err := ExtractZip(mixed, dest, 20, 1<<20, allowed)
	if err != nil {
		t.Fatal(err)
	}
	var uploaded, skipped int
	for _, entry := range entries {
		if entry.Skipped {
			skipped++
			continue
		}
		uploaded++
		if entry.Path != "docs/ok.txt" {
			t.Fatalf("uploaded path = %s", entry.Path)
		}
	}
	if uploaded != 1 || skipped < 4 {
		t.Fatalf("uploaded=%d skipped=%d entries=%+v", uploaded, skipped, entries)
	}
}

func TestExtractZipDecodesGBKName(t *testing.T) {
	encoded, err := simplifiedchinese.GBK.NewEncoder().String("说明.txt")
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	header := &zip.FileHeader{Name: encoded, Method: zip.Store}
	header.SetMode(0o644)
	header.Flags &^= 0x800
	entry, err := writer.CreateHeader(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte("内容")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(t.TempDir(), "gbk.zip")
	if err := os.WriteFile(src, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	entries, err := ExtractZip(src, t.TempDir(), 10, 1<<20, allowedSet([]string{".txt"}))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Skipped || entries[0].Name != "说明.txt" {
		t.Fatalf("entries = %+v", entries)
	}
}

func writeZip(t *testing.T, files map[string]string) string {
	t.Helper()
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	for name, content := range files {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ioWrite(entry, content); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "sample.zip")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func ioWrite(writer interface{ Write([]byte) (int, error) }, content string) (int, error) {
	return writer.Write([]byte(content))
}
