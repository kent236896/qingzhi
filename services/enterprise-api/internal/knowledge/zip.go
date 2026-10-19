package knowledge

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
)

// ZipEntry is one file extracted from an upload archive.
type ZipEntry struct {
	Name       string
	Path       string
	Skipped    bool
	SkipReason string
}

// ExtractZip unpacks a zip into dest and returns the files that should be uploaded.
// It rejects path traversal and stops when the entry count or unpacked size exceeds the limits.
func ExtractZip(src, dest string, maxEntries int, maxBytes int64, allowed map[string]struct{}) ([]ZipEntry, error) {
	reader, err := zip.OpenReader(src)
	if err != nil {
		return nil, invalid("压缩包无法读取")
	}
	defer reader.Close()
	if maxEntries > 0 && len(reader.File) > maxEntries {
		return nil, invalid(fmt.Sprintf("压缩包条目不能超过 %d 个", maxEntries))
	}
	var unpacked int64
	entries := make([]ZipEntry, 0)
	for _, file := range reader.File {
		name, err := zipName(file)
		if err != nil {
			return nil, err
		}
		cleaned, ok := safeZipPath(name)
		if !ok {
			return nil, invalid("压缩包包含不安全的路径")
		}
		if cleaned == "" || strings.HasSuffix(name, "/") || file.FileInfo().IsDir() {
			continue
		}
		base := path.Base(cleaned)
		if shouldSkipZip(cleaned, base, file) {
			reason := "已跳过"
			if isNestedZip(base) {
				reason = "不处理嵌套压缩包"
			}
			entries = append(entries, ZipEntry{Name: base, Path: cleaned, Skipped: true, SkipReason: reason})
			continue
		}
		ext := strings.ToLower(path.Ext(base))
		if _, allow := allowed[ext]; !allow {
			entries = append(entries, ZipEntry{Name: base, Path: cleaned, Skipped: true, SkipReason: "扩展名不在白名单"})
			continue
		}
		target := filepath.Join(dest, filepath.FromSlash(cleaned))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return nil, err
		}
		room := int64(1 << 62)
		if maxBytes > 0 {
			room = maxBytes - unpacked
		}
		written, err := unpackFile(file, target, room)
		if err != nil {
			return nil, err
		}
		unpacked += written
		entries = append(entries, ZipEntry{Name: base, Path: cleaned})
	}
	return entries, nil
}

func zipName(file *zip.File) (string, error) {
	name := file.Name
	if file.NonUTF8 || !utf8.ValidString(name) {
		decoded, err := simplifiedchinese.GBK.NewDecoder().String(name)
		if err == nil && decoded != "" {
			return decoded, nil
		}
	}
	return name, nil
}

func safeZipPath(name string) (string, bool) {
	name = strings.ReplaceAll(name, "\\", "/")
	if strings.Contains(name, ":") || strings.HasPrefix(name, "/") || strings.Contains(name, "\x00") {
		return "", false
	}
	cleaned := path.Clean(name)
	if cleaned == "." || strings.HasPrefix(cleaned, "../") || cleaned == ".." {
		return "", false
	}
	return cleaned, true
}

func shouldSkipZip(cleaned, base string, file *zip.File) bool {
	if file.Mode()&os.ModeSymlink != 0 {
		return true
	}
	lower := strings.ToLower(cleaned)
	if strings.HasPrefix(lower, "__macosx/") || strings.Contains(lower, "/__macosx/") {
		return true
	}
	if strings.EqualFold(base, ".ds_store") || strings.EqualFold(base, "thumbs.db") || strings.HasPrefix(base, ".") {
		return true
	}
	for _, part := range strings.Split(cleaned, "/") {
		if strings.HasPrefix(part, ".") {
			return true
		}
	}
	return isNestedZip(base)
}

func isNestedZip(base string) bool {
	ext := strings.ToLower(path.Ext(base))
	return ext == ".zip" || ext == ".rar" || ext == ".7z"
}

func unpackFile(file *zip.File, target string, room int64) (int64, error) {
	if room < 0 {
		room = 0
	}
	if file.UncompressedSize64 > uint64(room) && room >= 0 {
		return 0, invalid("解压后的总大小超出限制")
	}
	src, err := file.Open()
	if err != nil {
		return 0, err
	}
	defer src.Close()
	dst, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, err
	}
	defer dst.Close()
	limited := io.LimitReader(src, room+1)
	written, err := io.Copy(dst, limited)
	if err != nil {
		return written, err
	}
	if written > room {
		return written, invalid("解压后的总大小超出限制")
	}
	return written, nil
}

func allowedSet(exts []string) map[string]struct{} {
	out := make(map[string]struct{}, len(exts))
	for _, ext := range exts {
		out[strings.ToLower(ext)] = struct{}{}
	}
	return out
}
