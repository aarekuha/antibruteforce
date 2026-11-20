package antibruteforce

import (
	"context"
	"net/netip"
)

type ListKind string

const (
	Blacklist ListKind = "blacklist"
	Whitelist ListKind = "whitelist"
)

type ListMembership struct {
	InBlacklist bool
	InWhitelist bool
}

type NetworkLists interface {
	// Lookup возвращает принадлежность адреса обоим спискам по одному снимку данных
	Lookup(ctx context.Context, ip netip.Addr) (ListMembership, error)
	Add(ctx context.Context, list ListKind, subnet netip.Prefix) error
	Remove(ctx context.Context, list ListKind, subnet netip.Prefix) error
}
