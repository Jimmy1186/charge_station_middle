package api

import (
	eventbus "kenmec/jimmy/charge_core/infra"
	"sync"
)

type CANManager struct {
	mu     sync.RWMutex
	client map[string]*CANClient
}

func NewCANManager() *CANManager {
	return &CANManager{
		client: make(map[string]*CANClient),
	}
}

func (m *CANManager) Add(stationId, ip, port string, eb *eventbus.EventBus, reqEb *eventbus.RequestResponseBus) *CANClient {
	m.mu.Lock()
	defer m.mu.Unlock()

	client := NewCANClient(stationId, ip, port, eb, reqEb)
	m.client[stationId] = client

	client.WaitForConnection()
	return client
}

func (m *CANManager) GetAllClient() map[string]*CANClient {
	m.mu.RLock()
	defer m.mu.RUnlock()

	copy := make(map[string]*CANClient)
	for k, v := range m.client {
		copy[k] = v
	}
	return copy
}

func (m *CANManager) Get(stationId string) (*CANClient, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	c, ok := m.client[stationId]
	return c, ok
}

func (m *CANManager) Remove(stationId string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	c, ok := m.client[stationId]

	if ok {
		c.Close()
		delete(m.client, stationId)
	}
}

func (m *CANManager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, c := range m.client {
		c.Close()
		delete(m.client, id)
	}
}
