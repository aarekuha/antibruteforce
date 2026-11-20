package redis

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"
)

func TestIncrementLostReply(t *testing.T) {
	// Проверяем потерю ответа после передачи INCREX:
	// - сервер получает команду и закрывает соединение
	// - клиент возвращает ошибку после единственной отправки
	local, server := net.Pipe()
	defer server.Close()
	dials := 0
	client := &Client{pool: &redis.Pool{
		DialContext: func(context.Context) (redis.Conn, error) {
			dials++
			if dials > 1 {
				return nil, errors.New("unexpected reconnect")
			}
			return redis.NewConn(local, time.Second, time.Second), nil
		},
	}}
	defer client.Close()

	const request = "*8\r\n$6\r\nINCREX\r\n$6\r\nbucket\r\n$5\r\nBYINT\r\n$1\r\n1\r\n$6\r\nUBOUND\r\n$1\r\n2\r\n$2\r\nEX\r\n$2\r\n60\r\n"
	received := make(chan string, 1)
	go func() {
		defer server.Close()
		buf := make([]byte, len(request))
		n, _ := io.ReadFull(server, buf)
		received <- string(buf[:n])
	}()
	_, err := client.Increment(t.Context(), "bucket", 2)
	if !(errors.Is(err, io.EOF) || errors.Is(err, io.ErrClosedPipe)) || dials != 1 {
		t.Fatalf("Increment: err=%v, connections=%d", err, dials)
	}
	if got := <-received; got != request {
		t.Fatalf("unexpected request %q", got)
	}
}
