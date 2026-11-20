// Package antibruteforce содержит типы и интерфейсы для проверки попыток авторизации
package antibruteforce

import "net/netip"

type Attempt struct {
	Login    string
	Password string
	IP       netip.Addr
}

// ResetTargets задаёт выбранные счётчики: nil означает пропущенное поле,
// указатель на пустую строку выбирает счётчик пустого логина
type ResetTargets struct {
	Login *string
	IP    *netip.Addr
}
