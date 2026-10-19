package knowledge

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"qingzhi/enterprise-api/internal/weknora"
)

// LocalFile is one file already stored in a temporary path.
type LocalFile struct {
	Name       string
	Path       string
	CustomPath string
	Size       int64
}

func (s *Service) Upload(ctx context.Context, actor Actor, baseID string, files []LocalFile) ([]UploadResult, error) {
	item, _, err := s.writable(ctx, actor, baseID)
	if err != nil {
		return nil, err
	}
	if s.remote == nil {
		return nil, notReady("知识库服务未配置")
	}
	if len(files) > s.cfg.MaxFiles {
		return nil, invalid("一次上传的文件过多")
	}
	results := make([]UploadResult, len(files))
	sem := make(chan struct{}, s.cfg.UploadConcurrency)
	var wg sync.WaitGroup
	for i, file := range files {
		wg.Add(1)
		go func(index int, file LocalFile) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				results[index] = UploadResult{FileName: file.Name, Status: "failed", Error: "已取消"}
				return
			case sem <- struct{}{}:
				defer func() { <-sem }()
			}
			results[index] = s.uploadOne(ctx, actor, item, file)
		}(i, file)
	}
	wg.Wait()
	return results, nil
}

func (s *Service) uploadOne(ctx context.Context, actor Actor, item Base, file LocalFile) UploadResult {
	if file.Path == "" {
		return UploadResult{FileName: file.Name, Status: "skipped", Error: "扩展名不在白名单或文件无法读取"}
	}
	started := time.Now()
	custom := file.CustomPath
	if custom == "" {
		custom = file.Name
	}
	remote, err := s.remote.CreateKnowledgeFromFile(ctx, item.RemoteID, file.Path, custom)
	status := "success"
	message := ""
	if errors.Is(err, weknora.ErrDuplicate) {
		status = "duplicate"
	} else if err != nil {
		if s.log != nil {
			s.log.Warn("knowledge file upload failed", slog.String("component", "knowledge"), slog.String("kb_id", item.ID), slog.String("weknora_kb_id", item.RemoteID), slog.String("file_name", file.Name), slog.String("error", err.Error()))
		}
		return UploadResult{FileName: file.Name, Status: "failed", Error: remoteError(err).Error()}
	}
	folder, name := splitFolder(custom)
	if remote.FolderPath != "" {
		folder = remote.FolderPath
	}
	if remote.FileName != "" {
		name = remote.FileName
	}
	if name == "" || name == "." {
		name = file.Name
	}
	now := time.Now().UTC()
	record := File{
		ID: newID(), BaseID: item.ID, RemoteID: remote.ID, FileName: name, FolderPath: emptyFolder(folder),
		FileSize: file.Size, ParseStatus: statusValue(status, remote.ParseStatus), ErrorMsg: message,
		CreatorID: actor.UserID, CreateTime: now, UpdateTime: now,
	}
	if status == "duplicate" {
		record.ParseStatus = "duplicate"
	}
	if err := s.store.InsertFile(ctx, record); err != nil {
		return UploadResult{FileName: file.Name, Status: "failed", Error: "文件已上传，但本地记录失败"}
	}
	if s.log != nil {
		s.log.Info("knowledge file uploaded", slog.String("component", "knowledge"), slog.String("kb_id", item.ID), slog.String("weknora_kb_id", item.RemoteID), slog.String("file_name", name), slog.Int64("duration_ms", time.Since(started).Milliseconds()))
	}
	return UploadResult{FileName: name, Status: status, FileID: record.ID}
}

func (s *Service) UploadZip(ctx context.Context, actor Actor, baseID, zipPath string) ([]UploadResult, error) {
	dest, err := os.MkdirTemp("", "kb-zip-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dest)
	entries, err := ExtractZip(zipPath, dest, s.cfg.ZipMaxEntries, int64(s.cfg.ZipMaxTotalMB)<<20, allowedSet(s.cfg.AllowedExts))
	if err != nil {
		return nil, err
	}
	results := make([]UploadResult, 0, len(entries))
	files := make([]LocalFile, 0)
	for _, entry := range entries {
		if entry.Skipped {
			results = append(results, UploadResult{FileName: entry.Path, Status: "skipped", Error: entry.SkipReason})
			continue
		}
		info, statErr := os.Stat(filepath.Join(dest, filepath.FromSlash(entry.Path)))
		if statErr != nil {
			results = append(results, UploadResult{FileName: entry.Path, Status: "failed", Error: "解压文件不存在"})
			continue
		}
		files = append(files, LocalFile{Name: entry.Name, Path: filepath.Join(dest, filepath.FromSlash(entry.Path)), CustomPath: entry.Path, Size: info.Size()})
	}
	if len(files) > 0 {
		uploaded, err := s.Upload(ctx, actor, baseID, files)
		if err != nil {
			return nil, err
		}
		results = append(results, uploaded...)
	}
	return results, nil
}

func (s *Service) ListFiles(ctx context.Context, actor Actor, baseID string, sync bool) ([]File, error) {
	item, _, err := s.visible(ctx, actor, baseID)
	if err != nil {
		return nil, err
	}
	if sync {
		if err := s.syncFiles(ctx, item); err != nil {
			return nil, err
		}
	}
	files, err := s.store.ListFiles(ctx, baseID)
	for i := range files {
		files[i].RemoteID = ""
	}
	return files, err
}

func (s *Service) syncFiles(ctx context.Context, item Base) error {
	if s.remote == nil || item.RemoteID == "" {
		return nil
	}
	remoteFiles, err := s.remote.ListKnowledge(ctx, item.RemoteID)
	if err != nil {
		return remoteError(err)
	}
	byRemote := map[string]weknora.RemoteKnowledge{}
	for _, file := range remoteFiles {
		byRemote[file.ID] = file
	}
	local, err := s.store.ListFiles(ctx, item.ID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	for _, file := range local {
		remote, ok := byRemote[file.RemoteID]
		if !ok || file.RemoteID == "" {
			continue
		}
		file.ParseStatus = remote.ParseStatus
		file.ErrorMsg = trimError(remote.ErrorMessage)
		file.FolderPath = remote.FolderPath
		file.FileSize = remote.FileSize
		file.UpdateTime = now
		if err := s.store.UpdateFile(ctx, file); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) DeleteFiles(ctx context.Context, actor Actor, baseID string, ids []string) error {
	item, _, err := s.writable(ctx, actor, baseID)
	if err != nil {
		return err
	}
	files, err := s.store.FindFiles(ctx, baseID, ids)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return notFound("文件不存在")
	}
	remoteIDs := make([]string, 0, len(files))
	localIDs := make([]string, 0, len(files))
	for _, file := range files {
		localIDs = append(localIDs, file.ID)
		if file.RemoteID != "" {
			remoteIDs = append(remoteIDs, file.RemoteID)
		}
	}
	if s.remote != nil && len(remoteIDs) > 0 {
		if len(remoteIDs) == 1 {
			if err := s.remote.DeleteKnowledge(ctx, remoteIDs[0]); err != nil && !errors.Is(err, weknora.ErrNotFound) {
				return remoteError(err)
			}
		} else if err := s.remote.BatchDeleteKnowledge(ctx, item.RemoteID, remoteIDs); err != nil {
			return remoteError(err)
		}
	}
	return s.store.DeleteFiles(ctx, baseID, localIDs)
}

func (s *Service) ReparseFiles(ctx context.Context, actor Actor, baseID string, ids []string) error {
	item, _, err := s.writable(ctx, actor, baseID)
	if err != nil {
		return err
	}
	files, err := s.store.FindFiles(ctx, baseID, ids)
	if err != nil {
		return err
	}
	remoteIDs := make([]string, 0, len(files))
	for _, file := range files {
		if file.RemoteID != "" {
			remoteIDs = append(remoteIDs, file.RemoteID)
		}
	}
	if s.remote == nil || len(remoteIDs) == 0 {
		return invalid("没有可重新解析的文件")
	}
	if len(remoteIDs) == 1 {
		err = s.remote.ReparseKnowledge(ctx, remoteIDs[0])
	} else {
		err = s.remote.BatchReparseKnowledge(ctx, item.RemoteID, remoteIDs)
	}
	if err != nil {
		return remoteError(err)
	}
	now := time.Now().UTC()
	for _, file := range files {
		file.ParseStatus = "pending"
		file.ErrorMsg = ""
		file.UpdateTime = now
		if err := s.store.UpdateFile(ctx, file); err != nil {
			return err
		}
	}
	return nil
}

func SaveLimited(src io.Reader, maxBytes int64) (string, int64, error) {
	file, err := os.CreateTemp("", "kb-upload-*")
	if err != nil {
		return "", 0, err
	}
	written, err := io.Copy(file, io.LimitReader(src, maxBytes+1))
	closeErr := file.Close()
	if err != nil || closeErr != nil {
		os.Remove(file.Name())
		if err == nil {
			err = closeErr
		}
		return "", 0, err
	}
	if written > maxBytes {
		os.Remove(file.Name())
		return "", 0, invalid("文件超过大小限制")
	}
	return file.Name(), written, nil
}

func AcceptExt(name string, allowed []string) bool {
	ext := strings.ToLower(path.Ext(name))
	for _, item := range allowed {
		if ext == strings.ToLower(item) {
			return true
		}
	}
	return false
}

func statusValue(uploadStatus, remoteStatus string) string {
	if uploadStatus == "duplicate" {
		return "duplicate"
	}
	if remoteStatus == "" {
		return "pending"
	}
	return remoteStatus
}

func emptyFolder(folder string) string {
	if folder == "." || folder == "/" {
		return ""
	}
	return folder
}

func trimError(message string) string {
	message = strings.TrimSpace(message)
	if len(message) > 500 {
		return message[:500]
	}
	return message
}
