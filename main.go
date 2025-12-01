package main

import (
	"kenmec/jimmy/charge_core/api"
	bus "kenmec/jimmy/charge_core/eventbusV2/manager"
	"kenmec/jimmy/charge_core/eventbusV2/pub"
	"kenmec/jimmy/charge_core/eventbusV2/sub"
	"kenmec/jimmy/charge_core/log"
)

func main() {
		log.InitLog()

	busManager := bus.NewBusManager()
	stationService := pub.NewUserService(busManager)

	// ⭐ 建立 CANManager
	canManager := api.NewCANManager()

	// ⭐ 設定多個站
	can1 := canManager.Add("01", "127.0.0.1", "8081", stationService)
	can2 := canManager.Add("02", "127.0.0.1", "8082", stationService)

	// 等待多個 station ready
	can1.WaitForConnection()
	can2.WaitForConnection()

	// 每個 station 可以獨立接 MQTT（如果你要）
	mqtt := api.NewMQTTClient(canManager.GetAllClient(), stationService)
	mqtt.Subscribe("charge_station/command")

	// Handlers 註冊
	h := &sub.Subs{
		StationEventHandler: &sub.StationEventHandler{},
		MqttSub:             sub.NewMQTTEventSub(mqtt),
	}

	busManager.RegisterSubscribers(h.StationEventHandler)
	busManager.RegisterSubscribers(h.MqttSub)
	busManager.RegisterMiddlewares()

	select {}
}
