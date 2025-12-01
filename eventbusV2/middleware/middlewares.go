package middleware

import (
	"fmt"
	klog "kenmec/jimmy/charge_core/log"
)

// 您的 LogMiddleware 範例，現在內部可以調用 slog
func LogMiddleware[T any](e T, next func(T) error) error {
    klog.Logger.Info(fmt.Sprintf("%v", e))
    return next(e)
}

func RecoveryMiddleware[T any](e T, next func(T) error) (err error) {
    defer func() {
        if r := recover(); r != nil {
            klog.Logger.Error(fmt.Sprintf("Recovered panic: %v", r))
        }
    }()
    return next(e)
}
