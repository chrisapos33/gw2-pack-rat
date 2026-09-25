package handler_test

import (
	"context"
	"crypto/rand"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"gw2packrat/internal/db"
)

func testContext(t *testing.T) context.Context {
	t.Helper()
	return context.Background()
}

var testJWTSecret = []byte("test-jwt-secret")

func testEncKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	return key
}

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://gw2packrat:gw2packrat@localhost:5432/gw2packrat"
	}
	pool, err := db.Connect(context.Background(), dsn)
	if err != nil {
		t.Skipf("no test database available (%v) — set TEST_DATABASE_URL to enable", err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), "DELETE FROM gw2_api_keys WHERE user_id IN (SELECT id FROM users WHERE email LIKE '%@test.invalid')")
		pool.Exec(context.Background(), "DELETE FROM users WHERE email LIKE '%@test.invalid'")
		pool.Close()
	})
	return pool
}

// stubGW2 is a fake keyValidator used in handler tests.
type stubGW2 struct {
	permissions []string
	err         error
}

func (s *stubGW2) ValidateKey(_ string) ([]string, error) {
	return s.permissions, s.err
}
