package infra

import (
	"fmt"
	"sync"
)

// EventHandler is a function type that handles events
type EventHandler func(data interface{})

// EventBus manages event subscriptions and publishing
type EventBus struct {
	mu         sync.RWMutex
	handlers   map[string][]EventHandler
	handlerIDs map[string]map[int]EventHandler
	nextID     int
}

// New creates a new EventBus instance
func New() *EventBus {
	return &EventBus{
		handlers:   make(map[string][]EventHandler),
		handlerIDs: make(map[string]map[int]EventHandler),
		nextID:     0,
	}
}

// Subscribe registers a handler for a specific event
// Returns a subscription ID that can be used to unsubscribe
func (eb *EventBus) Subscribe(event string, handler EventHandler) int {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.handlers[event] = append(eb.handlers[event], handler)

	if eb.handlerIDs[event] == nil {
		eb.handlerIDs[event] = make(map[int]EventHandler)
	}

	id := eb.nextID
	eb.handlerIDs[event][id] = handler
	eb.nextID++

	return id
}

// Unsubscribe removes a handler using its subscription ID
func (eb *EventBus) Unsubscribe(event string, id int) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eventHandlers, exists := eb.handlerIDs[event]
	if !exists {
		return fmt.Errorf("event '%s' not found", event)
	}

	handler, exists := eventHandlers[id]
	if !exists {
		return fmt.Errorf("subscription ID %d not found for event '%s'", id, event)
	}

	delete(eventHandlers, id)

	// Remove from handlers slice
	handlers := eb.handlers[event]
	for i, h := range handlers {
		if &h == &handler {
			eb.handlers[event] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}

	return nil
}

// Publish sends an event to all subscribed handlers
func (eb *EventBus) Publish(event string, data interface{}) {
	eb.mu.RLock()
	handlers := eb.handlers[event]
	eb.mu.RUnlock()

	for _, handler := range handlers {
		go handler(data)
	}
}

// PublishSync sends an event to all subscribed handlers synchronously
func (eb *EventBus) PublishSync(event string, data interface{}) {
	eb.mu.RLock()
	handlers := eb.handlers[event]
	eb.mu.RUnlock()

	for _, handler := range handlers {
		handler(data)
	}
}

// Clear removes all handlers for a specific event
func (eb *EventBus) Clear(event string) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	delete(eb.handlers, event)
	delete(eb.handlerIDs, event)
}

// ClearAll removes all handlers for all events
func (eb *EventBus) ClearAll() {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.handlers = make(map[string][]EventHandler)
	eb.handlerIDs = make(map[string]map[int]EventHandler)
}
