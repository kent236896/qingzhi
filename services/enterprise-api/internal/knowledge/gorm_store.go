package knowledge

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

// GormStore persists knowledge records in PostgreSQL.
type GormStore struct {
	db *gorm.DB
}

func NewGormStore(db *gorm.DB) *GormStore { return &GormStore{db: db} }

func (s *GormStore) Transaction(ctx context.Context, fn func(Store) error) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(NewGormStore(tx))
	})
}

func (s *GormStore) InsertBase(ctx context.Context, item Base) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(baseRowFrom(item)).Error; err != nil {
			return err
		}
		return replaceTenants(tx, item.ID, item.TenantIDs)
	})
}

func (s *GormStore) UpdateBase(ctx context.Context, item Base) error {
	return s.db.WithContext(ctx).Model(&baseRow{}).Where("id = ? AND deleted = ?", item.ID, false).Updates(map[string]any{
		"name": item.Name, "description": item.Description, "updater": item.Updater, "update_time": item.UpdateTime,
		"wiki_wait_started_at": item.WikiWaitStart,
	}).Error
}

func (s *GormStore) DeleteBase(ctx context.Context, id string) error {
	now := time.Now().UTC()
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&baseRow{}).Where("id = ? AND deleted = ?", id, false).Updates(map[string]any{"deleted": true, "update_time": now}).Error; err != nil {
			return err
		}
		if err := tx.Model(&tenantRow{}).Where("knowledge_base_id = ? AND deleted = ?", id, false).Update("deleted", true).Error; err != nil {
			return err
		}
		if err := tx.Model(&shareRow{}).Where("knowledge_base_id = ? AND deleted = ?", id, false).Update("deleted", true).Error; err != nil {
			return err
		}
		if err := tx.Model(&shareScopeRow{}).Where("knowledge_base_id = ? AND deleted = ?", id, false).Update("deleted", true).Error; err != nil {
			return err
		}
		return tx.Model(&fileRow{}).Where("knowledge_base_id = ? AND deleted = ?", id, false).Updates(map[string]any{"deleted": true, "update_time": now}).Error
	})
}

func (s *GormStore) FindBase(ctx context.Context, id string) (Base, error) {
	var row baseRow
	err := s.db.WithContext(ctx).First(&row, "id = ? AND deleted = ?", id, false).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Base{}, ErrNotFound
	}
	if err != nil {
		return Base{}, err
	}
	item := row.model()
	if item.TenantIDs, err = s.tenantIDs(ctx, id); err != nil {
		return Base{}, err
	}
	if item.ShareUserIDs, err = s.shareIDs(ctx, id); err != nil {
		return Base{}, err
	}
	item.ShareOrgIDs, item.ShareRoleIDs, err = s.shareScopeIDs(ctx, id)
	return item, err
}

func (s *GormStore) ListBases(ctx context.Context) ([]Base, error) {
	var rows []baseRow
	if err := s.db.WithContext(ctx).Where("deleted = ?", false).Order("create_time DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]Base, 0, len(rows))
	for _, row := range rows {
		item := row.model()
		var err error
		if item.TenantIDs, err = s.tenantIDs(ctx, item.ID); err != nil {
			return nil, err
		}
		if item.ShareUserIDs, err = s.shareIDs(ctx, item.ID); err != nil {
			return nil, err
		}
		if item.ShareOrgIDs, item.ShareRoleIDs, err = s.shareScopeIDs(ctx, item.ID); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *GormStore) ReplaceTenants(ctx context.Context, baseID string, tenantIDs []string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return replaceTenants(tx, baseID, tenantIDs)
	})
}

func (s *GormStore) ReplaceShares(ctx context.Context, baseID string, userIDs []string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&shareRow{}).Where("knowledge_base_id = ? AND deleted = ?", baseID, false).Update("deleted", true).Error; err != nil {
			return err
		}
		now := time.Now().UTC()
		for _, userID := range userIDs {
			row := shareRow{ID: newID(), KnowledgeBaseID: baseID, UserID: userID, CreateTime: now}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *GormStore) ReplaceShareScopes(ctx context.Context, baseID string, orgIDs, roleIDs []string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&shareScopeRow{}).Where("knowledge_base_id = ? AND deleted = ?", baseID, false).Update("deleted", true).Error; err != nil {
			return err
		}
		now := time.Now().UTC()
		for _, orgID := range orgIDs {
			row := shareScopeRow{ID: newID(), KnowledgeBaseID: baseID, TargetType: ShareTargetOrg, TargetID: orgID, CreateTime: now}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		for _, roleID := range roleIDs {
			row := shareScopeRow{ID: newID(), KnowledgeBaseID: baseID, TargetType: ShareTargetRole, TargetID: roleID, CreateTime: now}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *GormStore) InsertFile(ctx context.Context, item File) error {
	return s.db.WithContext(ctx).Create(fileRowFrom(item)).Error
}

func (s *GormStore) UpdateFile(ctx context.Context, item File) error {
	return s.db.WithContext(ctx).Model(&fileRow{}).Where("id = ? AND deleted = ?", item.ID, false).Updates(map[string]any{
		"parse_status": item.ParseStatus, "error_msg": item.ErrorMsg, "folder_path": item.FolderPath,
		"file_size": item.FileSize, "update_time": item.UpdateTime, "weknora_knowledge_id": item.RemoteID,
	}).Error
}

func (s *GormStore) ListFiles(ctx context.Context, baseID string) ([]File, error) {
	var rows []fileRow
	err := s.db.WithContext(ctx).Where("knowledge_base_id = ? AND deleted = ?", baseID, false).Order("create_time DESC").Find(&rows).Error
	return filesFrom(rows), err
}

func (s *GormStore) FindFiles(ctx context.Context, baseID string, ids []string) ([]File, error) {
	var rows []fileRow
	err := s.db.WithContext(ctx).Where("knowledge_base_id = ? AND id IN ? AND deleted = ?", baseID, ids, false).Find(&rows).Error
	return filesFrom(rows), err
}

func (s *GormStore) DeleteFiles(ctx context.Context, baseID string, ids []string) error {
	now := time.Now().UTC()
	return s.db.WithContext(ctx).Model(&fileRow{}).Where("knowledge_base_id = ? AND id IN ? AND deleted = ?", baseID, ids, false).
		Updates(map[string]any{"deleted": true, "update_time": now}).Error
}

func (s *GormStore) tenantIDs(ctx context.Context, baseID string) ([]string, error) {
	var ids []string
	err := s.db.WithContext(ctx).Model(&tenantRow{}).Where("knowledge_base_id = ? AND deleted = ?", baseID, false).Pluck("tenant_id", &ids).Error
	if ids == nil {
		ids = []string{}
	}
	return ids, err
}

func (s *GormStore) shareIDs(ctx context.Context, baseID string) ([]string, error) {
	var ids []string
	err := s.db.WithContext(ctx).Model(&shareRow{}).Where("knowledge_base_id = ? AND deleted = ?", baseID, false).Pluck("user_id", &ids).Error
	if ids == nil {
		ids = []string{}
	}
	return ids, err
}

func (s *GormStore) shareScopeIDs(ctx context.Context, baseID string) ([]string, []string, error) {
	var rows []shareScopeRow
	err := s.db.WithContext(ctx).Where("knowledge_base_id = ? AND deleted = ?", baseID, false).Find(&rows).Error
	if err != nil {
		return nil, nil, err
	}
	orgs := make([]string, 0)
	roles := make([]string, 0)
	for _, row := range rows {
		switch row.TargetType {
		case ShareTargetOrg:
			orgs = append(orgs, row.TargetID)
		case ShareTargetRole:
			roles = append(roles, row.TargetID)
		}
	}
	return orgs, roles, nil
}

func replaceTenants(tx *gorm.DB, baseID string, tenantIDs []string) error {
	if err := tx.Model(&tenantRow{}).Where("knowledge_base_id = ? AND deleted = ?", baseID, false).Update("deleted", true).Error; err != nil {
		return err
	}
	for _, tenantID := range tenantIDs {
		if err := tx.Create(&tenantRow{ID: newID(), KnowledgeBaseID: baseID, TenantID: tenantID}).Error; err != nil {
			return err
		}
	}
	return nil
}

type baseRow struct {
	ID            string     `gorm:"type:uuid;primaryKey"`
	Name          string     `gorm:"column:name"`
	Description   string     `gorm:"column:description"`
	RemoteID      string     `gorm:"column:weknora_kb_id"`
	ScopeType     string     `gorm:"column:scope_type"`
	ScopeID       string     `gorm:"column:scope_id"`
	WikiEnabled   bool       `gorm:"column:wiki_enabled"`
	Status        int        `gorm:"column:status"`
	CreatorID     string     `gorm:"column:creator_id"`
	WikiWaitStart *time.Time `gorm:"column:wiki_wait_started_at"`
	Creator       string     `gorm:"column:creator"`
	CreateTime    time.Time  `gorm:"column:create_time"`
	Updater       string     `gorm:"column:updater"`
	UpdateTime    time.Time  `gorm:"column:update_time"`
	Deleted       bool       `gorm:"column:deleted"`
}

func (baseRow) TableName() string { return "sys_knowledge_bases" }

func baseRowFrom(item Base) *baseRow {
	return &baseRow{
		ID: item.ID, Name: item.Name, Description: item.Description, RemoteID: item.RemoteID,
		ScopeType: item.ScopeType, ScopeID: item.ScopeID, WikiEnabled: item.WikiEnabled, Status: item.Status,
		CreatorID: item.CreatorID, WikiWaitStart: item.WikiWaitStart, Creator: item.Creator,
		CreateTime: item.CreateTime, Updater: item.Updater, UpdateTime: item.UpdateTime,
	}
}

func (row baseRow) model() Base {
	return Base{
		ID: row.ID, Name: row.Name, Description: row.Description, RemoteID: row.RemoteID,
		ScopeType: row.ScopeType, ScopeID: row.ScopeID, WikiEnabled: row.WikiEnabled, Status: row.Status,
		CreatorID: row.CreatorID, WikiWaitStart: row.WikiWaitStart, Creator: row.Creator,
		CreateTime: row.CreateTime, Updater: row.Updater, UpdateTime: row.UpdateTime, Deleted: row.Deleted,
	}
}

type tenantRow struct {
	ID              string `gorm:"type:uuid;primaryKey"`
	KnowledgeBaseID string `gorm:"column:knowledge_base_id"`
	TenantID        string `gorm:"column:tenant_id"`
	Deleted         bool   `gorm:"column:deleted"`
}

func (tenantRow) TableName() string { return "sys_knowledge_base_tenants" }

type shareRow struct {
	ID              string    `gorm:"type:uuid;primaryKey"`
	KnowledgeBaseID string    `gorm:"column:knowledge_base_id"`
	UserID          string    `gorm:"column:user_id"`
	Creator         string    `gorm:"column:creator"`
	CreateTime      time.Time `gorm:"column:create_time"`
	Deleted         bool      `gorm:"column:deleted"`
}

func (shareRow) TableName() string { return "sys_knowledge_shares" }

type shareScopeRow struct {
	ID              string    `gorm:"type:uuid;primaryKey"`
	KnowledgeBaseID string    `gorm:"column:knowledge_base_id"`
	TargetType      string    `gorm:"column:target_type"`
	TargetID        string    `gorm:"column:target_id"`
	Creator         string    `gorm:"column:creator"`
	CreateTime      time.Time `gorm:"column:create_time"`
	Deleted         bool      `gorm:"column:deleted"`
}

func (shareScopeRow) TableName() string { return "sys_knowledge_share_scopes" }

type fileRow struct {
	ID          string    `gorm:"type:uuid;primaryKey"`
	BaseID      string    `gorm:"column:knowledge_base_id"`
	RemoteID    string    `gorm:"column:weknora_knowledge_id"`
	FileName    string    `gorm:"column:file_name"`
	FolderPath  string    `gorm:"column:folder_path"`
	FileSize    int64     `gorm:"column:file_size"`
	ParseStatus string    `gorm:"column:parse_status"`
	ErrorMsg    string    `gorm:"column:error_msg"`
	CreatorID   string    `gorm:"column:creator_id"`
	Creator     string    `gorm:"column:creator"`
	CreateTime  time.Time `gorm:"column:create_time"`
	Updater     string    `gorm:"column:updater"`
	UpdateTime  time.Time `gorm:"column:update_time"`
	Deleted     bool      `gorm:"column:deleted"`
}

func (fileRow) TableName() string { return "sys_knowledge_files" }

func fileRowFrom(item File) *fileRow {
	return &fileRow{
		ID: item.ID, BaseID: item.BaseID, RemoteID: item.RemoteID, FileName: item.FileName, FolderPath: item.FolderPath,
		FileSize: item.FileSize, ParseStatus: item.ParseStatus, ErrorMsg: item.ErrorMsg, CreatorID: item.CreatorID,
		CreateTime: item.CreateTime, UpdateTime: item.UpdateTime,
	}
}

func (row fileRow) model() File {
	return File{
		ID: row.ID, BaseID: row.BaseID, RemoteID: row.RemoteID, FileName: row.FileName, FolderPath: row.FolderPath,
		FileSize: row.FileSize, ParseStatus: row.ParseStatus, ErrorMsg: row.ErrorMsg, CreatorID: row.CreatorID,
		CreateTime: row.CreateTime, UpdateTime: row.UpdateTime, Deleted: row.Deleted,
	}
}

func filesFrom(rows []fileRow) []File {
	items := make([]File, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.model())
	}
	return items
}
