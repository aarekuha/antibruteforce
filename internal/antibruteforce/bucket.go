package antibruteforce

import "context"

type BucketKind string

const (
	BucketIP       BucketKind = "ip"
	BucketLogin    BucketKind = "login"
	BucketPassword BucketKind = "password"
)

// BucketKey обозначает один счётчик. Value содержит исходный логин,
// исходный пароль или каноническую строку IPv4
type BucketKey struct {
	Kind  BucketKind
	Value string
}

type BucketLimiter interface {
	// TryConsume атомарно списывает одну попытку и возвращает true при успехе
	// Исчерпанный лимит даёт false, nil. При ошибке результат списания может быть неизвестен
	TryConsume(ctx context.Context, key BucketKey) (bool, error)
	// Reset сбрасывает все выбранные счётчики логина и IP за один вызов
	Reset(ctx context.Context, keys []BucketKey) error
}
