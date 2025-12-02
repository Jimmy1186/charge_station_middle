package sub

// ===== 一般事件 Handlers =====
type FuncSub[T any] func(T) error


func (f FuncSub[T]) Sub(e T) error {
	return f(e)
}

