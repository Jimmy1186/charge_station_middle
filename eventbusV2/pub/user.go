package pub

// service/order_service.go

import (
	"kenmec/jimmy/charge_core/eventbusV2/events"
	"kenmec/jimmy/charge_core/eventbusV2/manager"
)




type StationService struct {
	busManager *manager.BusManager
}

func NewUserService(bm *manager.BusManager) *StationService {
	return &StationService{busManager: bm}
}

// 使用一般 EventBus (不需要回應)
func (s *StationService) PubStationStatus(payload events.StationStatus ) {
	// 發送事件通知其他模組
	s.busManager.StationEventBus.PublishAsync(events.StationStatus{
		StationID: payload.StationID,
		Status: payload.Status,
		Other: payload.Other,
		Error: payload.Error,
	})
	
	///fmt.Println("station event published")
}


// func(s *StationService) PubQamsCommand(payload events.QamsCommand){

// 	s.busManager.QamsCommandEventBus.PublishAsync(events.QamsCommand{
// 		Cmd: payload.Cmd,
// 	})

// }
