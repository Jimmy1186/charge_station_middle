package sub

import (
	"kenmec/jimmy/charge_core/api"
	"kenmec/jimmy/charge_core/eventbusV2/events"
)

type MQTTEventHandler struct {
	mqtt *api.MQTT_Client
}

func NewMQTTEventHandler(m *api.MQTT_Client) *MQTTEventHandler {
	return &MQTTEventHandler{mqtt: m}
}

func (h *MQTTEventHandler) Handle(e events.StationStatus) error {
	// 呼叫 MQTT Client 發送狀態
	h.mqtt.PublishStatus(e)
	return nil
}
