package infra

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Request represents a request with metadata
type Request struct {
	Topic     string
	Data      interface{}
	Timestamp time.Time
	ID        string
	ReplyTo   string
}

// Response represents a response to a request
type Response struct {
	RequestID string
	Data      interface{}
	Error     error
	Timestamp time.Time
}

// RequestHandler processes requests and returns responses
type RequestHandler func(ctx context.Context, req Request) (interface{}, error)

// ResponseHandler processes responses
type ResponseHandler func(ctx context.Context, res Response)

// RequestSubscription represents an active request handler
type RequestSubscription struct {
	ID      int
	Topic   string
	Handler RequestHandler
}

// RequestResponseBus handles request-response communication patterns
type RequestResponseBus struct {
	mu               sync.RWMutex
	handlers         map[string]*RequestSubscription
	pendingRequests  map[string]chan Response
	responseHandlers map[string]ResponseHandler
	nextID           int
	timeout          time.Duration
	logger           Logger
}

// Logger interface for logging
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// DefaultLogger is a simple console logger
type DefaultLogger struct{}

func (l *DefaultLogger) Debug(msg string, args ...interface{}) {
	fmt.Printf("[DEBUG] "+msg+"\n", args...)
}

func (l *DefaultLogger) Info(msg string, args ...interface{}) {
	fmt.Printf("[INFO] "+msg+"\n", args...)
}

func (l *DefaultLogger) Error(msg string, args ...interface{}) {
	fmt.Printf("[ERROR] "+msg+"\n", args...)
}

// Config holds configuration for the request-response bus
type Config struct {
	DefaultTimeout time.Duration
	Logger         Logger
}

// New creates a new RequestResponseBus with default configuration
func NewReqBus() *RequestResponseBus {
	return NewWithConfig(Config{
		DefaultTimeout: 30 * time.Second,
		Logger:         &DefaultLogger{},
	})
}

// NewWithConfig creates a new RequestResponseBus with custom configuration
func NewWithConfig(config Config) *RequestResponseBus {
	return &RequestResponseBus{
		handlers:         make(map[string]*RequestSubscription),
		pendingRequests:  make(map[string]chan Response),
		responseHandlers: make(map[string]ResponseHandler),
		nextID:           0,
		timeout:          config.DefaultTimeout,
		logger:           config.Logger,
	}
}

// RegisterHandler registers a handler for a specific request topic
func (rb *RequestResponseBus) RegisterHandler(topic string, handler RequestHandler) int {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	sub := &RequestSubscription{
		ID:      rb.nextID,
		Topic:   topic,
		Handler: handler,
	}

	rb.handlers[topic] = sub
	rb.nextID++

	if rb.logger != nil {
		rb.logger.Debug("Registered handler for topic: %s (ID: %d)", topic, sub.ID)
	}

	return sub.ID
}

// UnregisterHandler removes a handler for a topic
func (rb *RequestResponseBus) UnregisterHandler(topic string) error {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if _, exists := rb.handlers[topic]; !exists {
		return fmt.Errorf("no handler registered for topic '%s'", topic)
	}

	delete(rb.handlers, topic)

	if rb.logger != nil {
		rb.logger.Debug("Unregistered handler for topic: %s", topic)
	}

	return nil
}

// Request sends a request and waits for a response
func (rb *RequestResponseBus) Request(topic string, data interface{}) (*Response, error) {
	return rb.RequestWithTimeout(context.Background(), topic, data, rb.timeout)
}

// RequestWithContext sends a request with a context
func (rb *RequestResponseBus) RequestWithContext(ctx context.Context, topic string, data interface{}) (*Response, error) {
	return rb.RequestWithTimeout(ctx, topic, data, rb.timeout)
}

// RequestWithTimeout sends a request with a custom timeout
func (rb *RequestResponseBus) RequestWithTimeout(ctx context.Context, topic string, data interface{}, timeout time.Duration) (*Response, error) {
	// Create request
	req := Request{
		Topic:     topic,
		Data:      data,
		Timestamp: time.Now(),
		ID:        fmt.Sprintf("%s-%d", topic, time.Now().UnixNano()),
	}

	// Check if handler exists
	rb.mu.RLock()
	handler, exists := rb.handlers[topic]
	rb.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no handler registered for topic '%s'", topic)
	}

	// Create response channel
	responseChan := make(chan Response, 1)

	rb.mu.Lock()
	rb.pendingRequests[req.ID] = responseChan
	rb.mu.Unlock()

	// Clean up after
	defer func() {
		rb.mu.Lock()
		delete(rb.pendingRequests, req.ID)
		rb.mu.Unlock()
		close(responseChan)
	}()

	// Execute handler in goroutine
	go func() {
		if rb.logger != nil {
			rb.logger.Debug("Processing request: %s (ID: %s)", req.Topic, req.ID)
		}

		data, err := handler.Handler(ctx, req)

		response := Response{
			RequestID: req.ID,
			Data:      data,
			Error:     err,
			Timestamp: time.Now(),
		}

		// Send response
		select {
		case responseChan <- response:
		case <-time.After(1 * time.Second):
			if rb.logger != nil {
				rb.logger.Error("Response channel timeout for request %s", req.ID)
			}
		}
	}()

	// Wait for response or timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	select {
	case response := <-responseChan:
		if rb.logger != nil {
			rb.logger.Debug("Received response for request: %s", req.ID)
		}
		return &response, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("request timeout after %v: %w", timeout, ctx.Err())
	}
}

// RequestAsync sends a request and handles the response asynchronously
func (rb *RequestResponseBus) RequestAsync(topic string, data interface{}, callback ResponseHandler) error {
	return rb.RequestAsyncWithTimeout(context.Background(), topic, data, rb.timeout, callback)
}

// RequestAsyncWithTimeout sends an async request with a custom timeout
func (rb *RequestResponseBus) RequestAsyncWithTimeout(ctx context.Context, topic string, data interface{}, timeout time.Duration, callback ResponseHandler) error {
	go func() {
		response, err := rb.RequestWithTimeout(ctx, topic, data, timeout)
		if err != nil {
			callback(ctx, Response{
				Error:     err,
				Timestamp: time.Now(),
			})
			return
		}
		callback(ctx, *response)
	}()
	return nil
}

// Broadcast sends a request to all handlers (if multiple handlers per topic supported in future)
func (rb *RequestResponseBus) Broadcast(topic string, data interface{}) ([]*Response, error) {
	rb.mu.RLock()
	handler, exists := rb.handlers[topic]
	rb.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no handler registered for topic '%s'", topic)
	}

	req := Request{
		Topic:     topic,
		Data:      data,
		Timestamp: time.Now(),
		ID:        fmt.Sprintf("%s-%d", topic, time.Now().UnixNano()),
	}

	ctx := context.Background()
	responseData, err := handler.Handler(ctx, req)

	response := &Response{
		RequestID: req.ID,
		Data:      responseData,
		Error:     err,
		Timestamp: time.Now(),
	}

	return []*Response{response}, nil
}

// HasHandler checks if a handler is registered for a topic
func (rb *RequestResponseBus) HasHandler(topic string) bool {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	_, exists := rb.handlers[topic]
	return exists
}

// Topics returns all registered topics
func (rb *RequestResponseBus) Topics() []string {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	topics := make([]string, 0, len(rb.handlers))
	for topic := range rb.handlers {
		topics = append(topics, topic)
	}
	return topics
}

// ClearAll removes all handlers
func (rb *RequestResponseBus) ClearAll() {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.handlers = make(map[string]*RequestSubscription)
}

// Utility functions

// TypedRequestHandler creates a type-safe request handler
func TypedRequestHandler[TReq, TRes any](fn func(ctx context.Context, data TReq) (TRes, error)) RequestHandler {
	return func(ctx context.Context, req Request) (interface{}, error) {
		data, ok := req.Data.(TReq)
		if !ok {
			return nil, fmt.Errorf("invalid request data type")
		}
		return fn(ctx, data)
	}
}

// Chain creates a request pipeline (call multiple services in sequence)
func (rb *RequestResponseBus) Chain(ctx context.Context, topics []string, initialData interface{}) (*Response, error) {
	var currentData interface{} = initialData
	var lastResponse *Response

	for _, topic := range topics {
		response, err := rb.RequestWithContext(ctx, topic, currentData)
		if err != nil {
			return nil, fmt.Errorf("chain failed at topic '%s': %w", topic, err)
		}

		if response.Error != nil {
			return response, nil
		}

		currentData = response.Data
		lastResponse = response
	}

	return lastResponse, nil
}

// Parallel sends requests to multiple topics in parallel and waits for all
func (rb *RequestResponseBus) Parallel(ctx context.Context, requests map[string]interface{}) (map[string]*Response, error) {
	results := make(map[string]*Response)
	var mu sync.Mutex
	var wg sync.WaitGroup
	errChan := make(chan error, len(requests))

	for topic, data := range requests {
		wg.Add(1)
		go func(t string, d interface{}) {
			defer wg.Done()

			response, err := rb.RequestWithContext(ctx, t, d)
			if err != nil {
				errChan <- fmt.Errorf("request to '%s' failed: %w", t, err)
				return
			}

			mu.Lock()
			results[t] = response
			mu.Unlock()
		}(topic, data)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return results, fmt.Errorf("parallel requests had %d errors: %v", len(errs), errs[0])
	}

	return results, nil
}

// RequestOptions provides options for requests
type RequestOptions struct {
	Timeout    time.Duration
	Retries    int
	RetryDelay time.Duration
}

// RequestWithOptions sends a request with advanced options
func (rb *RequestResponseBus) RequestWithOptions(ctx context.Context, topic string, data interface{}, opts RequestOptions) (*Response, error) {
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = rb.timeout
	}

	retries := opts.Retries
	if retries <= 0 {
		retries = 1
	}

	var lastErr error
	for i := 0; i < retries; i++ {
		if i > 0 {
			if rb.logger != nil {
				rb.logger.Debug("Retrying request to %s (attempt %d/%d)", topic, i+1, retries)
			}
			time.Sleep(opts.RetryDelay)
		}

		response, err := rb.RequestWithTimeout(ctx, topic, data, timeout)
		if err == nil {
			return response, nil
		}

		lastErr = err
	}

	return nil, fmt.Errorf("request failed after %d retries: %w", retries, lastErr)
}
