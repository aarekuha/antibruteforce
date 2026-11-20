//go:build integration

package redis

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"testing"
	"time"

	"antibruteforce/internal/antibruteforce"
	"antibruteforce/internal/config"
	"github.com/gomodule/redigo/redis"
)

func TestTryConsumeRedis(t *testing.T) {
	// Проверяем поведение общего счётчика в Redis:
	// - успешное списание обновляет TTL, отказ сохраняет срок истечения
	// - из двух конкурентных обращений последнюю попытку получает одно
	// - после истечения TTL снова доступен полный запас
	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancel()
	const localConfig = "../../../config.yaml"
	path := localConfig
	if _, err := os.Stat(path); os.IsNotExist(err) {
		path = "../../../config.example.yaml"
	}
	t.Setenv("ABF_HMAC_SECRET", "test-secret")
	t.Setenv("ABF_LIMITS_LOGIN", "2")
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	client := NewClient(cfg.Redis)
	t.Cleanup(func() { client.Close() })
	command := func(name string, args ...any) (int64, error) {
		conn, err := client.pool.GetContext(ctx)
		if err != nil {
			return 0, err
		}
		defer conn.Close()
		return redis.Int64(redis.DoContext(conn, ctx, name, args...))
	}
	limiter, err := NewLimiter(client.Increment, Limits(cfg.Limits), cfg.HMACSecret)
	if err != nil {
		t.Fatal(err)
	}
	key := antibruteforce.BucketKey{Kind: antibruteforce.BucketLogin, Value: rand.Text()}
	sum := sha256.Sum256([]byte(key.Value))
	redisKey := "abf:v2:login:" + hex.EncodeToString(sum[:])
	read := func(cmd string) int64 {
		t.Helper()
		n, err := command(cmd, redisKey)
		if err != nil {
			t.Fatal(err)
		}
		return n
	}
	consume := func(want bool) {
		t.Helper()
		got, err := limiter.TryConsume(ctx, key)
		if got != want || err != nil {
			t.Fatalf("TryConsume = %v, %v; want %v", got, err, want)
		}
	}
	freshTTL := func() bool {
		ttl := read("PTTL")
		return ttl >= 55000 && ttl <= 60000
	}
	consume(true)
	if read("GET") != 1 || !freshTTL() {
		t.Fatal("expected one consumed attempt with a fresh TTL")
	}
	// Сократить TTL тестового ключа перед проверкой его обновления
	if _, err := command("PEXPIRE", redisKey, "20000"); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	type result struct {
		allowed bool
		err     error
	}
	results := make(chan result, 2)
	for range 2 {
		go func() {
			<-start
			ok, err := limiter.TryConsume(ctx, key)
			results <- result{ok, err}
		}()
	}
	close(start)
	first, second := <-results, <-results
	if first.err != nil || second.err != nil || first.allowed == second.allowed {
		t.Fatalf("concurrent results = %+v, %+v", first, second)
	}
	if read("GET") != 2 || !freshTTL() {
		t.Fatal("expected two consumed attempts with a refreshed TTL")
	}
	expiration := read("PEXPIRETIME")
	consume(false)
	if read("PEXPIRETIME") != expiration || read("GET") != 2 {
		t.Fatal("refusal changed the bucket")
	}
	if _, err := command("PEXPIRE", redisKey, "1"); err != nil {
		t.Fatal(err)
	}
	for read("EXISTS") != 0 {
		time.Sleep(time.Millisecond)
	}
	consume(true)
	if read("GET") != 1 {
		t.Fatal("expected a fresh bucket after expiration")
	}
}
