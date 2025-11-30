package sub

import (
	"kenmec/jimmy/charge_core/api"
	"kenmec/jimmy/charge_core/eventbusV2/events"
)

type MQTTEventSub struct {
	mqtt *api.MQTT_Client
}

func NewMQTTEventSub(m *api.MQTT_Client) *MQTTEventSub {
	return &MQTTEventSub{mqtt: m}
}

//一定要叫Sub
func (h *MQTTEventSub) Sub(e events.StationStatus) error {
	// 呼叫 MQTT Client 發送狀態
	h.mqtt.PublishStatus(e)
	return nil
}
