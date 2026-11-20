package redis

import (
	"context"
	"errors"
	"testing"

	"antibruteforce/internal/antibruteforce"
)

func TestTryConsumeKeyAndLimit(t *testing.T) {
	// Проверяем выбор лимита и формирование ключа:
	// - IP используется в каноническом виде
	// - исходные строки логина и пароля хешируются с сохранением пробелов
	for _, tc := range []struct {
		kind  antibruteforce.BucketKind
		value string
		key   string
		limit uint32
	}{
		{antibruteforce.BucketIP, "192.0.2.1", "abf:v2:ip:192.0.2.1", 3},
		{antibruteforce.BucketLogin, " Alice ", "abf:v2:login:f3a5bb89166b4962b88378ac39fc4ddc241006e9464a705dd802d919207d9125", 2},
		{antibruteforce.BucketPassword, " pass ", "abf:v2:password:e3646152cfbd227ff869c8c9194577079b3ab8e256e14c0ec6ac37bbc34a5db4", 1},
	} {
		t.Run(string(tc.kind), func(t *testing.T) {
			calls := 0
			increment := func(ctx context.Context, key string, limit uint32) (bool, error) {
				calls++
				if ctx != t.Context() || key != tc.key || limit != tc.limit {
					t.Fatal("unexpected increment arguments")
				}
				return true, nil
			}
			limiter, err := NewLimiter(increment, Limits{IP: 3, Login: 2, Password: 1}, "test-secret")
			if err != nil {
				t.Fatal(err)
			}
			ok, err := limiter.TryConsume(t.Context(), antibruteforce.BucketKey{Kind: tc.kind, Value: tc.value})
			if err != nil || !ok || calls != 1 {
				t.Fatalf("allowed=%v, err=%v, calls=%d", ok, err, calls)
			}
		})
	}
}

func TestTryConsumeRefusalAndError(t *testing.T) {
	// Проверяем остановку после ответа Redis:
	// - нулевой результат означает исчерпание запаса
	// - ошибка передаётся вызывающему коду без повторного списания
	for _, failure := range []error{nil, context.DeadlineExceeded} {
		calls := 0
		limiter, err := NewLimiter(func(context.Context, string, uint32) (bool, error) {
			calls++
			return false, failure
		}, Limits{IP: 2}, "test-secret")
		if err != nil {
			t.Fatal(err)
		}
		ok, err := limiter.TryConsume(t.Context(), antibruteforce.BucketKey{Kind: antibruteforce.BucketIP, Value: "192.0.2.1"})
		if ok || !errors.Is(err, failure) || calls != 1 {
			t.Fatalf("allowed=%v, err=%v, calls=%d", ok, err, calls)
		}
	}
}
