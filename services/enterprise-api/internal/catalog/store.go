package catalog

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Store persists agent or MCP catalog rows.
type Store interface {
	List(ctx context.Context) ([]Item, error)
	Get(ctx context.Context, id string) (Item, error)
	Insert(ctx context.Context, item Item) error
	Update(ctx context.Context, item Item) error
	Delete(ctx context.Context, id string) error
	ReplaceTenants(ctx context.Context, id string, tenantIDs []string) error
	ReplaceShares(ctx context.Context, id string, userIDs []string) error
}

type agentRow struct {
	ID          string    `gorm:"column:id;primaryKey"`
	Name        string    `gorm:"column:name"`
	Description string    `gorm:"column:description"`
	RemoteID    string    `gorm:"column:weknora_agent_id"`
	ScopeType   string    `gorm:"column:scope_type"`
	ScopeID     string    `gorm:"column:scope_id"`
	Status      int       `gorm:"column:status"`
	CreatorID   string    `gorm:"column:creator_id"`
	Creator     string    `gorm:"column:creator"`
	CreateTime  time.Time `gorm:"column:create_time"`
	Updater     string    `gorm:"column:updater"`
	UpdateTime  time.Time `gorm:"column:update_time"`
	Deleted     bool      `gorm:"column:deleted"`
}

func (agentRow) TableName() string { return "sys_agents" }

type mcpRow struct {
	ID            string    `gorm:"column:id;primaryKey"`
	Name          string    `gorm:"column:name"`
	Description   string    `gorm:"column:description"`
	RemoteID      string    `gorm:"column:weknora_mcp_id"`
	TransportType string    `gorm:"column:transport_type"`
	ScopeType     string    `gorm:"column:scope_type"`
	ScopeID       string    `gorm:"column:scope_id"`
	Status        int       `gorm:"column:status"`
	CreatorID     string    `gorm:"column:creator_id"`
	Creator       string    `gorm:"column:creator"`
	CreateTime    time.Time `gorm:"column:create_time"`
	Updater       string    `gorm:"column:updater"`
	UpdateTime    time.Time `gorm:"column:update_time"`
	Deleted       bool      `gorm:"column:deleted"`
}

func (mcpRow) TableName() string { return "sys_mcp_services" }

type linkRow struct {
	ID       string `gorm:"column:id;primaryKey"`
	ParentID string `gorm:"column:parent_id"`
	RefID    string `gorm:"column:ref_id"`
	Deleted  bool   `gorm:"column:deleted"`
}

// AgentStore persists agents.
type AgentStore struct{ db *gorm.DB }

func NewAgentStore(db *gorm.DB) *AgentStore { return &AgentStore{db: db} }

func (s *AgentStore) List(ctx context.Context) ([]Item, error) {
	var rows []agentRow
	if err := s.db.WithContext(ctx).Where("deleted = false").Order("update_time desc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]Item, 0, len(rows))
	for _, row := range rows {
		item := agentFromRow(row)
		item.TenantIDs, _ = s.loadTenants(ctx, row.ID)
		item.ShareUserIDs, _ = s.loadShares(ctx, row.ID)
		out = append(out, item)
	}
	return out, nil
}

func (s *AgentStore) Get(ctx context.Context, id string) (Item, error) {
	var row agentRow
	err := s.db.WithContext(ctx).Where("id = ? AND deleted = false", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Item{}, ErrNotFound
	}
	if err != nil {
		return Item{}, err
	}
	item := agentFromRow(row)
	item.TenantIDs, _ = s.loadTenants(ctx, row.ID)
	item.ShareUserIDs, _ = s.loadShares(ctx, row.ID)
	return item, nil
}

func (s *AgentStore) Insert(ctx context.Context, item Item) error {
	row := agentToRow(item)
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	if err := s.ReplaceTenants(ctx, item.ID, item.TenantIDs); err != nil {
		return err
	}
	return s.ReplaceShares(ctx, item.ID, item.ShareUserIDs)
}

func (s *AgentStore) Update(ctx context.Context, item Item) error {
	res := s.db.WithContext(ctx).Model(&agentRow{}).Where("id = ? AND deleted = false", item.ID).Updates(map[string]any{
		"name": item.Name, "description": item.Description, "updater": item.Updater, "update_time": item.UpdateTime,
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	if err := s.ReplaceTenants(ctx, item.ID, item.TenantIDs); err != nil {
		return err
	}
	return s.ReplaceShares(ctx, item.ID, item.ShareUserIDs)
}

func (s *AgentStore) Delete(ctx context.Context, id string) error {
	res := s.db.WithContext(ctx).Model(&agentRow{}).Where("id = ? AND deleted = false", id).Updates(map[string]any{
		"deleted": true, "update_time": time.Now().UTC(),
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *AgentStore) ReplaceTenants(ctx context.Context, id string, tenantIDs []string) error {
	_ = s.db.WithContext(ctx).Exec(`UPDATE sys_agent_tenants SET deleted = true WHERE agent_id = ? AND deleted = false`, id)
	for _, tid := range tenantIDs {
		row := map[string]any{"id": uuid.NewString(), "agent_id": id, "tenant_id": tid, "deleted": false}
		if err := s.db.WithContext(ctx).Table("sys_agent_tenants").Create(row).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *AgentStore) ReplaceShares(ctx context.Context, id string, userIDs []string) error {
	_ = s.db.WithContext(ctx).Exec(`UPDATE sys_agent_shares SET deleted = true WHERE agent_id = ? AND deleted = false`, id)
	for _, uid := range userIDs {
		row := map[string]any{"id": uuid.NewString(), "agent_id": id, "user_id": uid, "creator": "", "create_time": time.Now().UTC(), "deleted": false}
		if err := s.db.WithContext(ctx).Table("sys_agent_shares").Create(row).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *AgentStore) loadTenants(ctx context.Context, id string) ([]string, error) {
	var ids []string
	err := s.db.WithContext(ctx).Table("sys_agent_tenants").Where("agent_id = ? AND deleted = false", id).Pluck("tenant_id", &ids).Error
	return ids, err
}

func (s *AgentStore) loadShares(ctx context.Context, id string) ([]string, error) {
	var ids []string
	err := s.db.WithContext(ctx).Table("sys_agent_shares").Where("agent_id = ? AND deleted = false", id).Pluck("user_id", &ids).Error
	return ids, err
}

func agentFromRow(row agentRow) Item {
	return Item{
		ID: row.ID, Name: row.Name, Description: row.Description, RemoteID: row.RemoteID,
		ScopeType: row.ScopeType, ScopeID: row.ScopeID, Status: row.Status, CreatorID: row.CreatorID,
		Creator: row.Creator, Updater: row.Updater, CreateTime: row.CreateTime, UpdateTime: row.UpdateTime,
	}
}

func agentToRow(item Item) agentRow {
	return agentRow{
		ID: item.ID, Name: item.Name, Description: item.Description, RemoteID: item.RemoteID,
		ScopeType: item.ScopeType, ScopeID: item.ScopeID, Status: item.Status, CreatorID: item.CreatorID,
		Creator: item.Creator, CreateTime: item.CreateTime, Updater: item.Updater, UpdateTime: item.UpdateTime,
	}
}

// MCPStore persists MCP toolbox entries.
type MCPStore struct{ db *gorm.DB }

func NewMCPStore(db *gorm.DB) *MCPStore { return &MCPStore{db: db} }

func (s *MCPStore) List(ctx context.Context) ([]Item, error) {
	var rows []mcpRow
	if err := s.db.WithContext(ctx).Where("deleted = false").Order("update_time desc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]Item, 0, len(rows))
	for _, row := range rows {
		item := mcpFromRow(row)
		item.TenantIDs, _ = s.loadTenants(ctx, row.ID)
		item.ShareUserIDs, _ = s.loadShares(ctx, row.ID)
		out = append(out, item)
	}
	return out, nil
}

func (s *MCPStore) Get(ctx context.Context, id string) (Item, error) {
	var row mcpRow
	err := s.db.WithContext(ctx).Where("id = ? AND deleted = false", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Item{}, ErrNotFound
	}
	if err != nil {
		return Item{}, err
	}
	item := mcpFromRow(row)
	item.TenantIDs, _ = s.loadTenants(ctx, row.ID)
	item.ShareUserIDs, _ = s.loadShares(ctx, row.ID)
	return item, nil
}

func (s *MCPStore) Insert(ctx context.Context, item Item) error {
	row := mcpToRow(item)
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	if err := s.ReplaceTenants(ctx, item.ID, item.TenantIDs); err != nil {
		return err
	}
	return s.ReplaceShares(ctx, item.ID, item.ShareUserIDs)
}

func (s *MCPStore) Update(ctx context.Context, item Item) error {
	res := s.db.WithContext(ctx).Model(&mcpRow{}).Where("id = ? AND deleted = false", item.ID).Updates(map[string]any{
		"name": item.Name, "description": item.Description, "transport_type": item.TransportType,
		"updater": item.Updater, "update_time": item.UpdateTime,
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	if err := s.ReplaceTenants(ctx, item.ID, item.TenantIDs); err != nil {
		return err
	}
	return s.ReplaceShares(ctx, item.ID, item.ShareUserIDs)
}

func (s *MCPStore) Delete(ctx context.Context, id string) error {
	res := s.db.WithContext(ctx).Model(&mcpRow{}).Where("id = ? AND deleted = false", id).Updates(map[string]any{
		"deleted": true, "update_time": time.Now().UTC(),
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *MCPStore) ReplaceTenants(ctx context.Context, id string, tenantIDs []string) error {
	_ = s.db.WithContext(ctx).Exec(`UPDATE sys_mcp_service_tenants SET deleted = true WHERE mcp_service_id = ? AND deleted = false`, id)
	for _, tid := range tenantIDs {
		row := map[string]any{"id": uuid.NewString(), "mcp_service_id": id, "tenant_id": tid, "deleted": false}
		if err := s.db.WithContext(ctx).Table("sys_mcp_service_tenants").Create(row).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *MCPStore) ReplaceShares(ctx context.Context, id string, userIDs []string) error {
	_ = s.db.WithContext(ctx).Exec(`UPDATE sys_mcp_service_shares SET deleted = true WHERE mcp_service_id = ? AND deleted = false`, id)
	for _, uid := range userIDs {
		row := map[string]any{"id": uuid.NewString(), "mcp_service_id": id, "user_id": uid, "creator": "", "create_time": time.Now().UTC(), "deleted": false}
		if err := s.db.WithContext(ctx).Table("sys_mcp_service_shares").Create(row).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *MCPStore) loadTenants(ctx context.Context, id string) ([]string, error) {
	var ids []string
	err := s.db.WithContext(ctx).Table("sys_mcp_service_tenants").Where("mcp_service_id = ? AND deleted = false", id).Pluck("tenant_id", &ids).Error
	return ids, err
}

func (s *MCPStore) loadShares(ctx context.Context, id string) ([]string, error) {
	var ids []string
	err := s.db.WithContext(ctx).Table("sys_mcp_service_shares").Where("mcp_service_id = ? AND deleted = false", id).Pluck("user_id", &ids).Error
	return ids, err
}

func mcpFromRow(row mcpRow) Item {
	return Item{
		ID: row.ID, Name: row.Name, Description: row.Description, RemoteID: row.RemoteID,
		TransportType: row.TransportType, ScopeType: row.ScopeType, ScopeID: row.ScopeID, Status: row.Status,
		CreatorID: row.CreatorID, Creator: row.Creator, Updater: row.Updater, CreateTime: row.CreateTime, UpdateTime: row.UpdateTime,
	}
}

func mcpToRow(item Item) mcpRow {
	return mcpRow{
		ID: item.ID, Name: item.Name, Description: item.Description, RemoteID: item.RemoteID,
		TransportType: item.TransportType, ScopeType: item.ScopeType, ScopeID: item.ScopeID, Status: item.Status,
		CreatorID: item.CreatorID, Creator: item.Creator, CreateTime: item.CreateTime, Updater: item.Updater, UpdateTime: item.UpdateTime,
	}
}
