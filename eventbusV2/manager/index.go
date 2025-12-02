package manager

import (
	"kenmec/jimmy/charge_core/eventbusV2/events"
	"kenmec/jimmy/charge_core/eventbusV2/middleware"
)

// BusManager 統一管理所有 EventBus
type BusManager struct {
	// 一般事件總線 (各種業務事件)
    StationEventBus EventBus[events.StationStatus]

    QamsCommandBus  EventBus[events.QamsCommand]
}

// NewBusManager 初始化所有 Bus
func NewBusManager() *BusManager {
	return &BusManager{
		// 初始化一般 EventBus
		StationEventBus: New[events.StationStatus](),
        QamsCommandBus:  New[events.QamsCommand](),
	}
}



// RegisterMiddlewares 註冊中間件
func (bm *BusManager) RegisterMiddlewares() {
	// 為所有 bus 添加 logging 和 recovery
	bm.StationEventBus.Use(middleware.LogMiddleware[events.StationStatus])

}