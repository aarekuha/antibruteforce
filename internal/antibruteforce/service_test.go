package antibruteforce

import (
	"context"
	"errors"
	"net/netip"
	"slices"
	"testing"
)

type checkLists struct {
	NetworkLists
	membership ListMembership
	err        error
	lookups    []netip.Addr
}

func (l *checkLists) Lookup(_ context.Context, ip netip.Addr) (ListMembership, error) {
	l.lookups = append(l.lookups, ip)
	return l.membership, l.err
}

type checkLimiter struct {
	BucketLimiter
	stopAt BucketKind
	err    error
	calls  []BucketKey
}

func (l *checkLimiter) TryConsume(_ context.Context, key BucketKey) (bool, error) {
	l.calls = append(l.calls, key)
	if key.Kind == l.stopAt {
		return false, l.err
	}
	return true, nil
}

func TestCheckAttempt(t *testing.T) {
	// Проверяем решение сервиса и учёт попыток:
	// - при совпадении со списком счётчики сохраняют запас
	// - при отказе или ошибке списания прекращаются на текущем этапе
	attempt := Attempt{IP: netip.MustParseAddr("192.0.2.7"), Login: " Alice ", Password: " pass "}
	keys := []BucketKey{
		{Kind: BucketIP, Value: "192.0.2.7"},
		{Kind: BucketLogin, Value: " Alice "},
		{Kind: BucketPassword, Value: " pass "},
	}
	unavailable := errors.New("хранилище недоступно")
	for _, tc := range []struct {
		name        string
		membership  ListMembership
		lookupErr   error
		stopAt      BucketKind
		consumeErr  error
		wantAllowed bool
		wantErr     error
		wantCalls   []BucketKey
	}{
		{name: "blacklist", membership: ListMembership{InBlacklist: true}},
		{name: "whitelist", membership: ListMembership{InWhitelist: true}, wantAllowed: true},
		{name: "both_lists", membership: ListMembership{InBlacklist: true, InWhitelist: true}},
		{name: "lookup_error", membership: ListMembership{InWhitelist: true}, lookupErr: unavailable, wantErr: unavailable},
		{name: "ip_exhausted", stopAt: BucketIP, wantCalls: keys[:1]},
		{name: "login_exhausted", stopAt: BucketLogin, wantCalls: keys[:2]},
		{name: "password_exhausted", stopAt: BucketPassword, wantCalls: keys},
		{name: "ip_error", stopAt: BucketIP, consumeErr: unavailable, wantErr: unavailable, wantCalls: keys[:1]},
		{name: "login_error", stopAt: BucketLogin, consumeErr: unavailable, wantErr: unavailable, wantCalls: keys[:2]},
		{name: "password_error", stopAt: BucketPassword, consumeErr: unavailable, wantErr: unavailable, wantCalls: keys},
		{name: "allowed", wantAllowed: true, wantCalls: keys},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lists := &checkLists{membership: tc.membership, err: tc.lookupErr}
			limiter := &checkLimiter{stopAt: tc.stopAt, err: tc.consumeErr}
			allowed, err := NewService(lists, limiter).CheckAttempt(t.Context(), attempt)
			if allowed != tc.wantAllowed || !errors.Is(err, tc.wantErr) {
				t.Errorf("CheckAttempt() = (%v, %v), want (%v, %v)", allowed, err, tc.wantAllowed, tc.wantErr)
			}
			if !slices.Equal(lists.lookups, []netip.Addr{attempt.IP}) {
				t.Errorf("Lookup calls = %v, want one lookup for %v", lists.lookups, attempt.IP)
			}
			if !slices.Equal(limiter.calls, tc.wantCalls) {
				t.Error("unexpected sequence of bucket keys")
			}
		})
	}
}
