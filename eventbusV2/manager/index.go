package manager

import (
	"kenmec/jimmy/charge_core/eventbusV2/events"
	"kenmec/jimmy/charge_core/eventbusV2/middleware"
)

// BusManager 統一管理所有 EventBus
type BusManager struct {
	// 一般事件總線 (各種業務事件)
	StationEventBus    EventBus[events.StationStatus]
}

// NewBusManager 初始化所有 Bus
func NewBusManager() *BusManager {
	return &BusManager{
		// 初始化一般 EventBus
		StationEventBus:    New[events.StationStatus](),
	}
}

// RegisterSubscribers 註冊所有 subscriber
func (bm *BusManager) RegisterSubscribers(
	stationEventHandler interface{}, 
	) {
	// 註冊一般事件處理
	if h, ok := stationEventHandler.(Sub[events.StationStatus]); ok {
    bm.StationEventBus.Subscribe(h)
}
}

// RegisterMiddlewares 註冊中間件
func (bm *BusManager) RegisterMiddlewares() {
	// 為所有 bus 添加 logging 和 recovery
	bm.StationEventBus.Use(middleware.LogMiddleware[events.StationStatus])

}