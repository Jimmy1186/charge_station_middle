package middleware

import "fmt"


func LogMiddleware[T any](e T, next func(T) error) error {
    fmt.Println("[Middleware] Event:", e)
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
