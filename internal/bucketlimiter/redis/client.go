package redis

import (
	"antibruteforce/internal/config"
	"context"
	"errors"

	"github.com/gomodule/redigo/redis"
)

// Client использует пул соединений с одним сервером Redis
type Client struct {
	pool *redis.Pool
}

func NewClient(cfg config.Redis) *Client {
	return &Client{pool: &redis.Pool{
		MaxIdle:     cfg.MaxIdle,
		IdleTimeout: cfg.IdleTimeout,
		DialContext: func(ctx context.Context) (redis.Conn, error) {
			return redis.DialContext(
				ctx, "tcp", cfg.Addr,
				redis.DialPassword(cfg.Password),
				redis.DialConnectTimeout(cfg.ConnectTimeout),
				redis.DialReadTimeout(cfg.ReadTimeout),
				redis.DialWriteTimeout(cfg.WriteTimeout),
			)
		},
	}}
}

func (client *Client) Close() error {
	return client.pool.Close()
}

// Increment атомарно учитывает попытку и обновляет TTL только при успешном списании
// При достижении лимита срок сохраняется; после истечения счёт начинается с нуля
func (client *Client) Increment(ctx context.Context, key string, limit uint32) (bool, error) {
	conn, err := client.pool.GetContext(ctx)
	if err != nil {
		return false, err
	}
	defer conn.Close()
	// При потере ответа команда уже могла выполниться, поэтому повтора нет
	reply, err := redis.Int64s(redis.DoContext(conn, ctx, "INCREX", key, "BYINT", 1, "UBOUND", limit, "EX", 60))
	if err != nil {
		return false, err
	}
	// Второй элемент ответа содержит фактически применённое приращение
	if len(reply) != 2 || (reply[1] != 0 && reply[1] != 1) {
		return false, errors.New("unexpected Redis INCREX reply")
	}
	return reply[1] == 1, nil
}
