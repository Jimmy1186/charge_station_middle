package manager

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"time"
)

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

	asyncJobs   chan T
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

// PublishAsync puts event to queue; fallback to goroutine if queue is full
func (b *simpleEventBus[T]) PublishAsync(event T) error {
	b.closeMu.Lock()
	closed := b.closed
	b.closeMu.Unlock()
	if closed {
		return errors.New("eventbus closed")
	}

	select {
	case b.asyncJobs <- event:
		return nil
	default:
		// fallback to avoid blocking producer: handle in new goroutine
		go b.exec(event)
		return nil
	}
}

// PublishSync executes handlers in current goroutine and returns error
func (b *simpleEventBus[T]) PublishSync(event T) error {
	b.closeMu.Lock()
	closed := b.closed
	b.closeMu.Unlock()
	if closed {
		return errors.New("eventbus closed")
	}
	return b.exec(event)
}

func (b *simpleEventBus[T]) exec(event T) error {
	// build final handler with read lock to protect handlers slice
	b.mu.RLock()
	handlersCopy := make([]Sub[T], len(b.handlers))
	copy(handlersCopy, b.handlers)

	// copy middlewares
	mwsCopy := make([]Middleware[T], len(b.middlewares))
	copy(mwsCopy, b.middlewares)
	b.mu.RUnlock()

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
