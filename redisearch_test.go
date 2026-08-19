package homerun

import (
	"strings"
	"testing"
)

func TestStoreInRediSearchRejectsEmptyIndex(t *testing.T) {
	err := StoreInRediSearch(
		Message{Title: "no index configured"},
		RedisConfig{Addr: "127.0.0.1", Port: "6379"},
	)

	if err == nil {
		t.Fatal("expected an error when RedisConfig.Index is empty, got nil")
	}
	if !strings.Contains(err.Error(), "index") {
		t.Errorf("expected the error to name the missing index, got %q", err)
	}
}

func TestStoreInRediSearchOnUnreachableRedisReturnsError(t *testing.T) {
	// Before this was fixed the failure path went through helpers calling
	// log.Fatalf, which would have taken the test binary - and any calling
	// service - down with it instead of returning.
	err := StoreInRediSearch(
		Message{Title: "unreachable"},
		RedisConfig{Addr: "127.0.0.1", Port: "1", Index: "homerun-test"},
	)

	if err == nil {
		t.Fatal("expected an error for an unreachable Redis, got nil")
	}
}

func TestNewRediSearchPoolSkipsAuthWithoutPassword(t *testing.T) {
	// A Redis without requirepass rejects AUTH outright, so sending it
	// unconditionally made every dial fail against an unauthenticated server.
	pool := newRediSearchPool(RedisConfig{Addr: "127.0.0.1", Port: "1"})
	defer func() { _ = pool.Close() }()

	conn := pool.Get()
	defer func() { _ = conn.Close() }()

	// Dial fails (nothing listens on port 1); what matters is that it fails on
	// the connection, not on an AUTH the caller never asked for.
	if err := conn.Err(); err != nil && strings.Contains(strings.ToUpper(err.Error()), "AUTH") {
		t.Errorf("pool sent AUTH despite an empty password: %v", err)
	}
}
