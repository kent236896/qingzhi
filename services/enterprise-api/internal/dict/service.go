package dict

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Service applies dictionary rules on top of a Store.
type Service struct {
	store Store
}

// NewService returns a dictionary service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// CreateTypeInput creates one dictionary type.
type CreateTypeInput struct {
	Name   string
	Type   string
	Status int
	Remark string
	Actor  string
}

// UpdateTypeInput patches one dictionary type. Nil fields stay unchanged.
type UpdateTypeInput struct {
	Name   *string
	Type   *string
	Status *int
	Remark *string
	Actor  string
}

// CreateDataInput creates one dictionary item.
type CreateDataInput struct {
	Sort      int
	Label     string
	Value     string
	DictType  string
	Status    int
	ColorType string
	CSSClass  string
	Remark    string
	Actor     string
}

// UpdateDataInput patches one dictionary item.
type UpdateDataInput struct {
	Sort      *int
	Label     *string
	Value     *string
	Status    *int
	ColorType *string
	CSSClass  *string
	Remark    *string
	Actor     string
}

func (s *Service) ListTypes(ctx context.Context, query ListQuery) (Page[Type], error) {
	page, pageSize, err := normalizePage(query.Page, query.PageSize)
	if err != nil {
		return Page[Type]{}, err
	}
	var result Page[Type]
	err = s.store.Transaction(ctx, func(tx Store) error {
		items, total, err := tx.ListTypes(ctx, strings.TrimSpace(query.Keyword), page, pageSize)
		if err != nil {
			return err
		}
		result = newPage(items, page, pageSize, total)
		return nil
	})
	return result, err
}

func (s *Service) CreateType(ctx context.Context, input CreateTypeInput) (Type, error) {
	item := Type{
		Name:    strings.TrimSpace(input.Name),
		Type:    strings.TrimSpace(input.Type),
		Status:  input.Status,
		Remark:  strings.TrimSpace(input.Remark),
		Creator: strings.TrimSpace(input.Actor),
		Updater: strings.TrimSpace(input.Actor),
	}
	if err := validateDictName(item.Name); err != nil {
		return Type{}, err
	}
	if err := validateDictTypeCode(item.Type); err != nil {
		return Type{}, err
	}
	if err := validateStatus(item.Status); err != nil {
		return Type{}, err
	}
	if err := validateRemark(item.Remark); err != nil {
		return Type{}, err
	}
	if err := validateActor(item.Creator); err != nil {
		return Type{}, err
	}
	now := time.Now().UTC()
	item.ID = newID()
	item.CreateTime = now
	item.UpdateTime = now
	err := s.store.Transaction(ctx, func(tx Store) error {
		if _, err := tx.FindTypeByCode(ctx, item.Type); err == nil {
			return conflict("字典类型已存在")
		} else if !errors.Is(err, ErrNotFound) {
			return err
		}
		return tx.InsertType(ctx, item)
	})
	return item, err
}

func (s *Service) GetType(ctx context.Context, id string) (Type, error) {
	item, err := s.store.FindType(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return Type{}, notFound("字典类型不存在")
	}
	return item, err
}

func (s *Service) UpdateType(ctx context.Context, id string, input UpdateTypeInput) (Type, error) {
	if input.Name == nil && input.Type == nil && input.Status == nil && input.Remark == nil {
		return Type{}, invalid("没有可更新的字段")
	}
	actor := strings.TrimSpace(input.Actor)
	if err := validateActor(actor); err != nil {
		return Type{}, err
	}
	var updated Type
	err := s.store.Transaction(ctx, func(tx Store) error {
		item, err := tx.FindType(ctx, id)
		if errors.Is(err, ErrNotFound) {
			return notFound("字典类型不存在")
		}
		if err != nil {
			return err
		}
		oldCode := item.Type
		if input.Name != nil {
			item.Name = strings.TrimSpace(*input.Name)
			if err := validateDictName(item.Name); err != nil {
				return err
			}
		}
		if input.Type != nil {
			item.Type = strings.TrimSpace(*input.Type)
			if err := validateDictTypeCode(item.Type); err != nil {
				return err
			}
			if item.Type != oldCode {
				existing, err := tx.FindTypeByCode(ctx, item.Type)
				if err == nil && existing.ID != item.ID {
					return conflict("字典类型已存在")
				}
				if err != nil && !errors.Is(err, ErrNotFound) {
					return err
				}
			}
		}
		if input.Remark != nil {
			item.Remark = strings.TrimSpace(*input.Remark)
			if err := validateRemark(item.Remark); err != nil {
				return err
			}
		}
		if input.Status != nil {
			if err := validateStatus(*input.Status); err != nil {
				return err
			}
			item.Status = *input.Status
		}
		item.Updater = actor
		item.UpdateTime = time.Now().UTC()
		if err := tx.UpdateType(ctx, item); err != nil {
			return err
		}
		if item.Type != oldCode {
			if err := tx.RenameDataType(ctx, oldCode, item.Type); err != nil {
				return err
			}
		}
		updated = item
		return nil
	})
	return updated, err
}

func (s *Service) DeleteType(ctx context.Context, id string) error {
	return s.store.Transaction(ctx, func(tx Store) error {
		item, err := tx.FindType(ctx, id)
		if errors.Is(err, ErrNotFound) {
			return notFound("字典类型不存在")
		}
		if err != nil {
			return err
		}
		count, err := tx.CountData(ctx, item.Type)
		if err != nil {
			return err
		}
		if count > 0 {
			return conflict("字典类型下仍有数据，不能删除")
		}
		return tx.DeleteType(ctx, id, time.Now().UTC())
	})
}

func (s *Service) ListData(ctx context.Context, dictType string, query ListQuery) (Page[Data], error) {
	dictType = strings.TrimSpace(dictType)
	if err := validateDictTypeCode(dictType); err != nil {
		return Page[Data]{}, err
	}
	page, pageSize, err := normalizePage(query.Page, query.PageSize)
	if err != nil {
		return Page[Data]{}, err
	}
	var result Page[Data]
	err = s.store.Transaction(ctx, func(tx Store) error {
		if _, err := tx.FindTypeByCode(ctx, dictType); errors.Is(err, ErrNotFound) {
			return notFound("字典类型不存在")
		} else if err != nil {
			return err
		}
		items, total, err := tx.ListData(ctx, dictType, strings.TrimSpace(query.Keyword), page, pageSize)
		if err != nil {
			return err
		}
		result = newPage(items, page, pageSize, total)
		return nil
	})
	return result, err
}

func (s *Service) ListEnabledData(ctx context.Context, dictType string) ([]Data, error) {
	dictType = strings.TrimSpace(dictType)
	if err := validateDictTypeCode(dictType); err != nil {
		return nil, err
	}
	var items []Data
	err := s.store.Transaction(ctx, func(tx Store) error {
		item, err := tx.FindTypeByCode(ctx, dictType)
		if errors.Is(err, ErrNotFound) {
			return notFound("字典类型不存在")
		}
		if err != nil {
			return err
		}
		if item.Status != StatusNormal {
			items = []Data{}
			return nil
		}
		found, err := tx.ListEnabledData(ctx, dictType)
		if err != nil {
			return err
		}
		items = found
		return nil
	})
	if items == nil && err == nil {
		items = []Data{}
	}
	return items, err
}

func (s *Service) CreateData(ctx context.Context, input CreateDataInput) (Data, error) {
	item := Data{
		Sort:      input.Sort,
		Label:     strings.TrimSpace(input.Label),
		Value:     strings.TrimSpace(input.Value),
		DictType:  strings.TrimSpace(input.DictType),
		Status:    input.Status,
		ColorType: strings.TrimSpace(input.ColorType),
		CSSClass:  strings.TrimSpace(input.CSSClass),
		Remark:    strings.TrimSpace(input.Remark),
		Creator:   strings.TrimSpace(input.Actor),
		Updater:   strings.TrimSpace(input.Actor),
	}
	if err := validateData(item); err != nil {
		return Data{}, err
	}
	now := time.Now().UTC()
	item.ID = newID()
	item.CreateTime = now
	item.UpdateTime = now
	err := s.store.Transaction(ctx, func(tx Store) error {
		if _, err := tx.FindTypeByCode(ctx, item.DictType); errors.Is(err, ErrNotFound) {
			return notFound("字典类型不存在")
		} else if err != nil {
			return err
		}
		if _, err := tx.FindDataByValue(ctx, item.DictType, item.Value); err == nil {
			return conflict("字典键值已存在")
		} else if !errors.Is(err, ErrNotFound) {
			return err
		}
		return tx.InsertData(ctx, item)
	})
	return item, err
}

func (s *Service) GetData(ctx context.Context, id string) (Data, error) {
	item, err := s.store.FindData(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return Data{}, notFound("字典数据不存在")
	}
	return item, err
}

func (s *Service) UpdateData(ctx context.Context, id string, input UpdateDataInput) (Data, error) {
	if input.Sort == nil && input.Label == nil && input.Value == nil && input.Status == nil && input.ColorType == nil && input.CSSClass == nil && input.Remark == nil {
		return Data{}, invalid("没有可更新的字段")
	}
	actor := strings.TrimSpace(input.Actor)
	if err := validateActor(actor); err != nil {
		return Data{}, err
	}
	var updated Data
	err := s.store.Transaction(ctx, func(tx Store) error {
		item, err := tx.FindData(ctx, id)
		if errors.Is(err, ErrNotFound) {
			return notFound("字典数据不存在")
		}
		if err != nil {
			return err
		}
		if input.Sort != nil {
			item.Sort = *input.Sort
		}
		if input.Label != nil {
			item.Label = strings.TrimSpace(*input.Label)
		}
		if input.Value != nil {
			item.Value = strings.TrimSpace(*input.Value)
		}
		if input.ColorType != nil {
			item.ColorType = strings.TrimSpace(*input.ColorType)
		}
		if input.CSSClass != nil {
			item.CSSClass = strings.TrimSpace(*input.CSSClass)
		}
		if input.Remark != nil {
			item.Remark = strings.TrimSpace(*input.Remark)
		}
		if input.Status != nil {
			item.Status = *input.Status
		}
		if err := validateData(item); err != nil {
			return err
		}
		existing, err := tx.FindDataByValue(ctx, item.DictType, item.Value)
		if err == nil && existing.ID != item.ID {
			return conflict("字典键值已存在")
		}
		if err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}
		item.Updater = actor
		item.UpdateTime = time.Now().UTC()
		if err := tx.UpdateData(ctx, item); err != nil {
			return err
		}
		updated = item
		return nil
	})
	return updated, err
}

func (s *Service) DeleteData(ctx context.Context, id string) error {
	return s.store.Transaction(ctx, func(tx Store) error {
		if _, err := tx.FindData(ctx, id); errors.Is(err, ErrNotFound) {
			return notFound("字典数据不存在")
		} else if err != nil {
			return err
		}
		return tx.DeleteData(ctx, id)
	})
}

func validateData(item Data) error {
	if err := validateDictTypeCode(item.DictType); err != nil {
		return err
	}
	if err := validateLabel(item.Label); err != nil {
		return err
	}
	if err := validateValue(item.Value); err != nil {
		return err
	}
	if err := validateSort(item.Sort); err != nil {
		return err
	}
	if err := validateStatus(item.Status); err != nil {
		return err
	}
	if err := validateColorType(item.ColorType); err != nil {
		return err
	}
	if err := validateCSSClass(item.CSSClass); err != nil {
		return err
	}
	if err := validateRemark(item.Remark); err != nil {
		return err
	}
	return validateActor(item.Creator)
}

func newPage[T any](items []T, page, pageSize int, total int64) Page[T] {
	if items == nil {
		items = []T{}
	}
	return Page[T]{Items: items, Page: page, PageSize: pageSize, Total: total}
}

func newID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		panic(err)
	}
	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", buf[0:4], buf[4:6], buf[6:8], buf[8:10], buf[10:16])
}
