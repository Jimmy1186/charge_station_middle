package sub

import (
	"kenmec/jimmy/charge_core/eventbusV2/events"
)

// ===== 一般事件 Handlers =====

type StationEventHandler struct{}

func (h *StationEventHandler) Sub(event events.StationStatus) error {
	//fmt.Printf("StationEventHandler: %s - %s\n", event.Status, event.Other)
	// 你的處理邏輯
	return nil
}

