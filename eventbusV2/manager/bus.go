package manager

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"time"
)
type FuncSub[T any] func(T) error

func (f FuncSub[T]) Sub(e T) error {
	return f(e)
}


type Sub[T any] interface {
	Sub(T) error
}

type Middleware[T any] func(T, func(T) error) error

type EventBus[T any] interface {
	PublishSync(event T) error
	PublishAsync(event T) error

	Subscribe(handler Sub[T])
	Use(middleware Middleware[T])

	Close() error
}

type simpleEventBus[T any] struct {
	handlers    []Sub[T]
	middlewares []Middleware[T]

	asyncJobs   chan T  //不阻塞async時用的
	workerCount int
	minWorkers  int
	maxWorkers  int

	mu           sync.RWMutex // protect handlers & middlewares
	workerWg     sync.WaitGroup
	scalerCancel context.CancelFunc
	closed       bool
	closeMu      sync.Mutex
}

// New creates event bus and starts workers + autoscaler
func New[T any]() EventBus[T] {
	cpu := runtime.NumCPU()

	b := &simpleEventBus[T]{
		handlers:    make([]Sub[T], 0),
		middlewares: make([]Middleware[T], 0),

		asyncJobs:  make(chan T, 2000),

		minWorkers:  cpu,
		workerCount: cpu * 5,
		maxWorkers:  cpu * 20,
	}

	// start baseline workers
	for i := 0; i < b.workerCount; i++ {
		b.workerWg.Add(1)
		go b.worker()
	}

	// start scaler
	ctx, cancel := context.WithCancel(context.Background())
	b.scalerCancel = cancel
	go b.autoScaler(ctx)

	return b
}

func (b *simpleEventBus[T]) worker() {
	defer b.workerWg.Done()
	for evt := range b.asyncJobs {
		// each job handled synchronously in worker
		_ = b.exec(evt)
	}
}

// 把需要收到這個事件的人訂閱到 handler
func (b *simpleEventBus[T]) Subscribe(h Sub[T]) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers = append(b.handlers, h)
}

func (b *simpleEventBus[T]) Use(m Middleware[T]) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.middlewares = append(b.middlewares, m)
}

// 可以放到 後面去處裡像是通知之類的比較不重要的事
func (b *simpleEventBus[T]) PublishAsync(event T) error {
	b.closeMu.Lock()
	closed := b.closed
	b.closeMu.Unlock()
	if closed {
		return errors.New("eventbus closed")
	}

	select {
		//把 event 塞到 asyncJobs
	case b.asyncJobs <- event: 
		return nil
	default:
		// 如果asyncJobs滿出來的話 就直接執行
		go b.exec(event)
		return nil
	}
}

// 會堵塞執行續 重要事件像是寫資料庫 順序很重要的事件
func (b *simpleEventBus[T]) PublishSync(event T) error {
	b.closeMu.Lock()
	closed := b.closed
	b.closeMu.Unlock()
	if closed {
		return errors.New("eventbus closed")
	}
	return b.exec(event)
}

/// 當每次呼叫pub相關func 就會觸發這個
func (b *simpleEventBus[T]) exec(event T) error {
	// build final handler with read lock to protect handlers slice
	b.mu.RLock()

	// pub進去到handler這個array後 邊複製到一份
	handlersCopy := make([]Sub[T], len(b.handlers))
	copy(handlersCopy, b.handlers)

	// copy middlewares
	mwsCopy := make([]Middleware[T], len(b.middlewares))
	copy(mwsCopy, b.middlewares)
	b.mu.RUnlock()

	//將複製的每個元素呼叫 Sub 這個發法
	finalHandler := func(e T) error {
		for _, h := range handlersCopy {
			if err := h.Sub(e); err != nil {
				return err
			}
		}
		return nil
	}

	wrapped := finalHandler
	for i := len(mwsCopy) - 1; i >= 0; i-- {
		mw := mwsCopy[i]
		next := wrapped
		wrapped = func(e T) error {
			return mw(e, next)
		}
	}

	return wrapped(event)
}

func (b *simpleEventBus[T]) autoScaler(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			backlog := len(b.asyncJobs)

			// scale logic (simple)
			if backlog > b.workerCount*2 && b.workerCount < b.maxWorkers {
				add := b.workerCount / 2
				if add < 1 {
					add = 1
				}
				for i := 0; i < add; i++ {
					b.workerWg.Add(1)
					go b.worker()
				}
				b.workerCount += add
			}

			// reduce worker count softly - we won't forcibly kill goroutines, just adjust target
			if backlog == 0 && b.workerCount > b.minWorkers {
				b.workerCount -= 1
			}
		}
	}
}

// Close gracefully stops the bus: prevent new publishes, close asyncJobs, wait workers
func (b *simpleEventBus[T]) Close() error {
	b.closeMu.Lock()
	if b.closed {
		b.closeMu.Unlock()
		return nil
	}
	b.closed = true
	b.closeMu.Unlock()

	// stop scaler
	if b.scalerCancel != nil {
		b.scalerCancel()
	}

	// close queue so workers can exit
	close(b.asyncJobs)
	// wait for workers to finish
	b.workerWg.Wait()
	return nil
}
