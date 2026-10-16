package dict

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// GormStore stores dictionaries in PostgreSQL.
type GormStore struct {
	db *gorm.DB
}

// NewGormStore returns a PostgreSQL store.
func NewGormStore(db *gorm.DB) *GormStore {
	return &GormStore{db: db}
}

func (s *GormStore) Transaction(ctx context.Context, fn func(Store) error) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&GormStore{db: tx})
	})
}

func (s *GormStore) InsertType(ctx context.Context, item Type) error {
	return mapWrite(s.db.WithContext(ctx).Create(typeRowFrom(item)).Error)
}

func (s *GormStore) UpdateType(ctx context.Context, item Type) error {
	result := s.db.WithContext(ctx).Model(&typeRow{}).Where("id = ? AND deleted = ?", item.ID, false).Updates(map[string]any{
		"name":        item.Name,
		"type":        item.Type,
		"status":      item.Status,
		"remark":      nullable(item.Remark),
		"updater":     item.Updater,
		"update_time": item.UpdateTime,
	})
	if result.Error != nil {
		return mapWrite(result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *GormStore) DeleteType(ctx context.Context, id string, deletedAt time.Time) error {
	result := s.db.WithContext(ctx).Model(&typeRow{}).Where("id = ? AND deleted = ?", id, false).Updates(map[string]any{
		"deleted":      true,
		"deleted_time": deletedAt.UTC(),
		"update_time":  deletedAt.UTC(),
	})
	if result.Error != nil {
		return mapWrite(result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *GormStore) FindType(ctx context.Context, id string) (Type, error) {
	var row typeRow
	err := s.db.WithContext(ctx).First(&row, "id = ? AND deleted = ?", id, false).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Type{}, ErrNotFound
	}
	if err != nil {
		return Type{}, err
	}
	return row.model(), nil
}

func (s *GormStore) FindTypeByCode(ctx context.Context, code string) (Type, error) {
	var row typeRow
	err := s.db.WithContext(ctx).First(&row, `"type" = ? AND deleted = ?`, code, false).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Type{}, ErrNotFound
	}
	if err != nil {
		return Type{}, err
	}
	return row.model(), nil
}

func (s *GormStore) ListTypes(ctx context.Context, keyword string, page, pageSize int) ([]Type, int64, error) {
	query := applyKeyword(s.db.WithContext(ctx).Model(&typeRow{}).Where("deleted = ?", false), keyword, "name", `"type"`)
	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []typeRow
	err := query.Session(&gorm.Session{}).Order("name ASC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&rows).Error
	return typesFrom(rows), total, err
}

func (s *GormStore) InsertData(ctx context.Context, item Data) error {
	return mapWrite(s.db.WithContext(ctx).Create(dataRowFrom(item)).Error)
}

func (s *GormStore) UpdateData(ctx context.Context, item Data) error {
	result := s.db.WithContext(ctx).Model(&dataRow{}).Where("id = ? AND deleted = ?", item.ID, false).Updates(map[string]any{
		"sort":        item.Sort,
		"label":       item.Label,
		"value":       item.Value,
		"status":      item.Status,
		"color_type":  item.ColorType,
		"css_class":   item.CSSClass,
		"remark":      nullable(item.Remark),
		"updater":     item.Updater,
		"update_time": item.UpdateTime,
	})
	if result.Error != nil {
		return mapWrite(result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *GormStore) DeleteData(ctx context.Context, id string) error {
	result := s.db.WithContext(ctx).Model(&dataRow{}).Where("id = ? AND deleted = ?", id, false).Updates(map[string]any{
		"deleted":     true,
		"update_time": time.Now().UTC(),
	})
	if result.Error != nil {
		return mapWrite(result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *GormStore) FindData(ctx context.Context, id string) (Data, error) {
	var row dataRow
	err := s.db.WithContext(ctx).First(&row, "id = ? AND deleted = ?", id, false).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Data{}, ErrNotFound
	}
	if err != nil {
		return Data{}, err
	}
	return row.model(), nil
}

func (s *GormStore) FindDataByValue(ctx context.Context, dictType, value string) (Data, error) {
	var row dataRow
	err := s.db.WithContext(ctx).First(&row, "dict_type = ? AND value = ? AND deleted = ?", dictType, value, false).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Data{}, ErrNotFound
	}
	if err != nil {
		return Data{}, err
	}
	return row.model(), nil
}

func (s *GormStore) ListData(ctx context.Context, dictType, keyword string, page, pageSize int) ([]Data, int64, error) {
	query := applyKeyword(s.db.WithContext(ctx).Model(&dataRow{}).Where("dict_type = ? AND deleted = ?", dictType, false), keyword, "label", "value")
	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []dataRow
	err := query.Session(&gorm.Session{}).Order("sort ASC, label ASC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&rows).Error
	return dataFrom(rows), total, err
}

func (s *GormStore) ListEnabledData(ctx context.Context, dictType string) ([]Data, error) {
	var rows []dataRow
	err := s.db.WithContext(ctx).
		Where("dict_type = ? AND status = ? AND deleted = ?", dictType, StatusNormal, false).
		Order("sort ASC, label ASC").
		Find(&rows).Error
	return dataFrom(rows), err
}

func (s *GormStore) CountData(ctx context.Context, dictType string) (int64, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&dataRow{}).Where("dict_type = ? AND deleted = ?", dictType, false).Count(&count).Error
	return count, err
}

func (s *GormStore) RenameDataType(ctx context.Context, from, to string) error {
	return mapWrite(s.db.WithContext(ctx).Model(&dataRow{}).Where("dict_type = ? AND deleted = ?", from, false).Updates(map[string]any{
		"dict_type":   to,
		"update_time": time.Now().UTC(),
	}).Error)
}

type typeRow struct {
	ID          string     `gorm:"column:id;primaryKey"`
	Name        string     `gorm:"column:name"`
	Type        string     `gorm:"column:type"`
	Status      int        `gorm:"column:status"`
	Remark      *string    `gorm:"column:remark"`
	Creator     string     `gorm:"column:creator"`
	CreateTime  time.Time  `gorm:"column:create_time"`
	Updater     string     `gorm:"column:updater"`
	UpdateTime  time.Time  `gorm:"column:update_time"`
	Deleted     bool       `gorm:"column:deleted"`
	DeletedTime *time.Time `gorm:"column:deleted_time"`
}

func (typeRow) TableName() string { return "sys_dict_types" }

func typeRowFrom(item Type) typeRow {
	return typeRow{
		ID:          item.ID,
		Name:        item.Name,
		Type:        item.Type,
		Status:      item.Status,
		Remark:      nullableString(item.Remark),
		Creator:     item.Creator,
		CreateTime:  item.CreateTime,
		Updater:     item.Updater,
		UpdateTime:  item.UpdateTime,
		Deleted:     item.Deleted,
		DeletedTime: item.DeletedTime,
	}
}

func (row typeRow) model() Type {
	return Type{
		ID:          row.ID,
		Name:        row.Name,
		Type:        row.Type,
		Status:      row.Status,
		Remark:      stringValue(row.Remark),
		Creator:     row.Creator,
		CreateTime:  row.CreateTime,
		Updater:     row.Updater,
		UpdateTime:  row.UpdateTime,
		Deleted:     row.Deleted,
		DeletedTime: row.DeletedTime,
	}
}

func typesFrom(rows []typeRow) []Type {
	items := make([]Type, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.model())
	}
	return items
}

type dataRow struct {
	ID         string    `gorm:"column:id;primaryKey"`
	Sort       int       `gorm:"column:sort"`
	Label      string    `gorm:"column:label"`
	Value      string    `gorm:"column:value"`
	DictType   string    `gorm:"column:dict_type"`
	Status     int       `gorm:"column:status"`
	ColorType  string    `gorm:"column:color_type"`
	CSSClass   string    `gorm:"column:css_class"`
	Remark     *string   `gorm:"column:remark"`
	Creator    string    `gorm:"column:creator"`
	CreateTime time.Time `gorm:"column:create_time"`
	Updater    string    `gorm:"column:updater"`
	UpdateTime time.Time `gorm:"column:update_time"`
	Deleted    bool      `gorm:"column:deleted"`
}

func (dataRow) TableName() string { return "sys_dict_data" }

func dataRowFrom(item Data) dataRow {
	return dataRow{
		ID:         item.ID,
		Sort:       item.Sort,
		Label:      item.Label,
		Value:      item.Value,
		DictType:   item.DictType,
		Status:     item.Status,
		ColorType:  item.ColorType,
		CSSClass:   item.CSSClass,
		Remark:     nullableString(item.Remark),
		Creator:    item.Creator,
		CreateTime: item.CreateTime,
		Updater:    item.Updater,
		UpdateTime: item.UpdateTime,
		Deleted:    item.Deleted,
	}
}

func (row dataRow) model() Data {
	return Data{
		ID:         row.ID,
		Sort:       row.Sort,
		Label:      row.Label,
		Value:      row.Value,
		DictType:   row.DictType,
		Status:     row.Status,
		ColorType:  row.ColorType,
		CSSClass:   row.CSSClass,
		Remark:     stringValue(row.Remark),
		Creator:    row.Creator,
		CreateTime: row.CreateTime,
		Updater:    row.Updater,
		UpdateTime: row.UpdateTime,
		Deleted:    row.Deleted,
	}
}

func dataFrom(rows []dataRow) []Data {
	items := make([]Data, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.model())
	}
	return items
}

func nullable(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func nullableString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func applyKeyword(query *gorm.DB, keyword string, columns ...string) *gorm.DB {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return query
	}
	escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(keyword)
	pattern := "%" + escaped + "%"
	parts := make([]string, 0, len(columns))
	args := make([]any, 0, len(columns))
	for _, column := range columns {
		parts = append(parts, column+" ILIKE ? ESCAPE '\\'")
		args = append(args, pattern)
	}
	return query.Where("("+strings.Join(parts, " OR ")+")", args...)
}

func mapWrite(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		switch pgErr.ConstraintName {
		case "sys_dict_types_type_unique":
			return conflict("字典类型已存在")
		case "sys_dict_data_value_unique":
			return conflict("字典键值已存在")
		default:
			return conflict("数据已存在")
		}
	}
	return err
}
