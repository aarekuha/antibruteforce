package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadRedis(t *testing.T) {
	// Проверяем итоговую конфигурацию клиента:
	// - значения из YAML действуют при отсутствии переменных окружения
	// - явно заданные переменные изменяют только соответствующие параметры
	path := filepath.Join(t.TempDir(), "config.yaml")
	data := `http: {listen_addr: ":8080"}
postgres: {addr: "localhost:5432", database: test, user: test}
redis:
  addr: "localhost:6380"
  password: from-file
  max_idle: 3
  idle_timeout: 45s
  connect_timeout: 1s
  read_timeout: 2s
  write_timeout: 3s
hmac_secret: from-file
`
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"ADDR", "PASSWORD", "MAX_IDLE", "IDLE_TIMEOUT", "CONNECT_TIMEOUT", "READ_TIMEOUT", "WRITE_TIMEOUT"} {
		name := "ABF_REDIS_" + key
		t.Setenv(name, "")
		if err := os.Unsetenv(name); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("ABF_HMAC_SECRET", "test-secret")
	want := Redis{Addr: "localhost:6380", Password: "from-file", MaxIdle: 3, IdleTimeout: 45 * time.Second, ConnectTimeout: time.Second, ReadTimeout: 2 * time.Second, WriteTimeout: 3 * time.Second}
	check := func() {
		t.Helper()
		cfg, err := Load(path)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Redis != want {
			t.Fatal("unexpected Redis configuration")
		}
	}
	check()
	t.Setenv("ABF_REDIS_PASSWORD", "")
	t.Setenv("ABF_REDIS_MAX_IDLE", "0")
	t.Setenv("ABF_REDIS_READ_TIMEOUT", "250ms")
	want.Password, want.MaxIdle, want.ReadTimeout = "", 0, 250*time.Millisecond
	check()

	t.Setenv("ABF_REDIS_READ_TIMEOUT", "-1s")
	if _, err := Load(path); err == nil || !strings.Contains(err.Error(), "redis.read_timeout") {
		t.Fatalf("expected read timeout error, got %v", err)
	}
	t.Setenv("ABF_REDIS_READ_TIMEOUT", "2s")
	t.Setenv("ABF_HMAC_SECRET", "")
	if _, err := Load(path); err == nil || !strings.Contains(err.Error(), "hmac_secret") {
		t.Fatalf("expected missing HMAC secret error, got %v", err)
	}
}
