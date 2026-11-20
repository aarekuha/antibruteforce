package antibruteforce

import "context"

// Service проверяет попытки авторизации по спискам подсетей и счётчикам
type Service struct {
	lists   NetworkLists
	limiter BucketLimiter
}

// NewService создаёт сервис с общими для всех его запросов зависимостями
func NewService(lists NetworkLists, limiter BucketLimiter) *Service {
	return &Service{lists: lists, limiter: limiter}
}

// CheckAttempt проверяет попытку после валидации входных данных
func (s *Service) CheckAttempt(ctx context.Context, attempt Attempt) (bool, error) {
	membership, err := s.lists.Lookup(ctx, attempt.IP)
	if err != nil {
		return false, err
	}
	if membership.InBlacklist {
		return false, nil
	}
	if membership.InWhitelist {
		return true, nil
	}

	keys := [...]BucketKey{
		{Kind: BucketIP, Value: attempt.IP.String()},
		{Kind: BucketLogin, Value: attempt.Login},
		{Kind: BucketPassword, Value: attempt.Password},
	}
	for _, key := range keys {
		allowed, err := s.limiter.TryConsume(ctx, key)
		if err != nil || !allowed {
			return false, err
		}
	}
	return true, nil
}
