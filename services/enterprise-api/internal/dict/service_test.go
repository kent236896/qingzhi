package dict

import (
	"context"
	"errors"
	"testing"
)

func TestDictionaryTypeAndDataRules(t *testing.T) {
	svc := NewService(newMemoryStore())
	ctx := context.Background()

	created, err := svc.CreateType(ctx, CreateTypeInput{Name: "通用状态", Type: "common_status", Actor: "admin"})
	if err != nil {
		t.Fatal(err)
	}
	if created.Status != StatusNormal || created.Creator != "admin" {
		t.Fatalf("type = %+v", created)
	}
	if _, err := svc.CreateType(ctx, CreateTypeInput{Name: "重复", Type: "common_status"}); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate type error = %v", err)
	}
	if _, err := svc.CreateType(ctx, CreateTypeInput{Name: "非法", Type: "状态"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("invalid type error = %v", err)
	}

	item, err := svc.CreateData(ctx, CreateDataInput{
		Sort: 1, Label: "开启", Value: "0", DictType: "common_status", ColorType: "success", Actor: "admin",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateData(ctx, CreateDataInput{Label: "重复", Value: "0", DictType: "common_status"}); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate value error = %v", err)
	}
	if _, err := svc.CreateData(ctx, CreateDataInput{Label: "孤儿", Value: "1", DictType: "missing_type"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing type error = %v", err)
	}
	if err := svc.DeleteType(ctx, created.ID); !errors.Is(err, ErrConflict) {
		t.Fatalf("delete occupied type error = %v", err)
	}

	renamed, err := svc.UpdateType(ctx, created.ID, UpdateTypeInput{Type: strPtr("sys_status"), Actor: "editor"})
	if err != nil {
		t.Fatal(err)
	}
	moved, err := svc.GetData(ctx, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if renamed.Type != "sys_status" || moved.DictType != "sys_status" || renamed.Updater != "editor" {
		t.Fatalf("renamed type=%s data=%s updater=%s", renamed.Type, moved.DictType, renamed.Updater)
	}

	disabled := StatusDisabled
	if _, err := svc.UpdateType(ctx, created.ID, UpdateTypeInput{Status: &disabled}); err != nil {
		t.Fatal(err)
	}
	enabled, err := svc.ListEnabledData(ctx, "sys_status")
	if err != nil {
		t.Fatal(err)
	}
	if len(enabled) != 0 {
		t.Fatalf("disabled type lookup = %+v", enabled)
	}

	if _, err := svc.UpdateData(ctx, item.ID, UpdateDataInput{Value: strPtr("1"), ColorType: strPtr("nope")}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("invalid color error = %v", err)
	}
	if err := svc.DeleteData(ctx, item.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteType(ctx, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GetType(ctx, created.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted type error = %v", err)
	}
}

func strPtr(value string) *string {
	return &value
}
