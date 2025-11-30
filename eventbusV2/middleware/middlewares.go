package middleware

import (
	"fmt"
	"log/slog"
)

// 您的 LogMiddleware 範例，現在內部可以調用 slog
func LogMiddleware[T any](e T, next func(T) error) error {
    slog.Info("Event:", "details", e)
    return next(e)
}

func RecoveryMiddleware[T any](e T, next func(T) error) (err error) {
    defer func() {
        if r := recover(); r != nil {
            fmt.Println("Recovered panic:", r)
        }
    }()
    return next(e)
}
