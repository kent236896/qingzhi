package redis

import (
	"context"
	"strings"
	"testing"
	"time"

	"qingzhi/enterprise-api/internal/config"
	"qingzhi/enterprise-api/internal/platform/database"
)

func TestPingFailureDoesNotLeakPassword(t *testing.T) {
	const password = "super-secret-redis-password"
	client := Open(config.RedisConfig{
		Host:     "127.0.0.1",
		Port:     1,
		Password: password,
		DB:       0,
	})
	t.Cleanup(func() { _ = client.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := client.Ping(ctx)
	if err == nil {
		t.Fatal("expected ping to fail")
	}
	if strings.Contains(err.Error(), password) {
		t.Fatalf("error leaked password: %s", err)
	}
	connErr, ok := err.(*database.ConnectionError)
	if !ok {
		t.Fatalf("error type = %T", err)
	}
	if strings.Contains(connErr.Diagnostic(), password) {
		t.Fatalf("diagnostic leaked password: %s", connErr.Diagnostic())
	}
	if connErr.Error() != "redis unavailable" {
		t.Fatalf("error = %s", connErr.Error())
	}
}
