package manager

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Request-Reply 擴充
type ReplyEvent[T any] struct {
	CorrelationID string
	Payload       T
}

type PendingReply struct {
	Chan    chan interface{}
	Timeout time.Time
}

type ReplyBus[T any] struct {
	EventBus[ReplyEvent[T]]
	pendingReplies sync.Map // map[string]*PendingReply

	cleanupCancel context.CancelFunc
	cleanupWg     sync.WaitGroup
	closed        bool
	closeMu       sync.Mutex
}

func NewReplyBus[T any]() *ReplyBus[T] {
	rb := &ReplyBus[T]{
		EventBus: New[ReplyEvent[T]](),
	}
	// start cleanup goroutine with cancel
	ctx, cancel := context.WithCancel(context.Background())
	rb.cleanupCancel = cancel
	rb.cleanupWg.Add(1)
	go func() {
		defer rb.cleanupWg.Done()
		rb.cleanupExpired(ctx)
	}()
	return rb
}

// PublishAndWait: A 發送 request，等待回覆或 timeout
func (rb *ReplyBus[T]) PublishAndWait(payload T, timeout time.Duration) (interface{}, error) {
	rb.closeMu.Lock()
	if rb.closed {
		rb.closeMu.Unlock()
		return nil, errors.New("replybus closed")
	}
	rb.closeMu.Unlock()

	correlationID := uuid.New().String()
	replyChan := make(chan interface{}, 1)

	p := &PendingReply{
		Chan:    replyChan,
		Timeout: time.Now().Add(timeout),
	}
	rb.pendingReplies.Store(correlationID, p)

	event := ReplyEvent[T]{
		CorrelationID: correlationID,
		Payload:       payload,
	}

	// publish async (so publisher isn't blocked by handlers)
	if err := rb.PublishAsync(event); err != nil {
		rb.pendingReplies.Delete(correlationID)
		return nil, err
	}

	// wait with context to allow external cancel
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	select {
	case resp := <-replyChan:
		rb.pendingReplies.Delete(correlationID)
		return resp, nil
	case <-ctx.Done():
		rb.pendingReplies.Delete(correlationID)
		return nil, errors.New("reply timeout")
	}
}

// Reply: B 呼叫此方法回覆對應 correlationID
func (rb *ReplyBus[T]) Reply(correlationID string, response interface{}) error {
	val, ok := rb.pendingReplies.Load(correlationID)
	if !ok {
		return errors.New("no pending reply found")
	}
	pending := val.(*PendingReply)

	// non-blocking send to avoid deadlock if nobody is reading
	select {
	case pending.Chan <- response:
		rb.pendingReplies.Delete(correlationID)
		return nil
	default:
		// channel full or not ready; drop and delete to avoid leak
		rb.pendingReplies.Delete(correlationID)
		return errors.New("reply channel blocked")
	}
}

func (rb *ReplyBus[T]) cleanupExpired(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// clean remaining
			rb.pendingReplies.Range(func(key, value interface{}) bool {
				rb.pendingReplies.Delete(key)
				return true
			})
			return
		case <-ticker.C:
			now := time.Now()
			rb.pendingReplies.Range(func(key, value interface{}) bool {
				pending := value.(*PendingReply)
				if now.After(pending.Timeout) {
					// try to notify with nil or error, non-blocking
					select {
					case pending.Chan <- errors.New("reply timeout"):
					default:
					}
					rb.pendingReplies.Delete(key)
				}
				return true
			})
		}
	}
}

// Close stops cleanup goroutine and closes embedded EventBus
func (rb *ReplyBus[T]) Close() error {
	rb.closeMu.Lock()
	if rb.closed {
		rb.closeMu.Unlock()
		return nil
	}
	rb.closed = true
	rb.closeMu.Unlock()

	if rb.cleanupCancel != nil {
		rb.cleanupCancel()
	}
	rb.cleanupWg.Wait()

	// try to close underlying EventBus if it supports Close()
	if eb, ok := interface{}(rb.EventBus).(interface{ Close() error }); ok {
		_ = eb.Close()
	}
	return nil
}
