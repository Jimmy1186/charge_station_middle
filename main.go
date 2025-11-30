package main

import (
	"kenmec/jimmy/charge_core/api"
	bus "kenmec/jimmy/charge_core/eventbusV2/manager"
	"kenmec/jimmy/charge_core/eventbusV2/pub"
	"kenmec/jimmy/charge_core/eventbusV2/sub"
	"kenmec/jimmy/charge_core/log"

	"github.com/rs/zerolog"
)

func main() {
	log.InitLog()
   zerolog.TimeFieldFormat = zerolog.TimeFormatUnix


   busManager := bus.NewBusManager()



   stationService := pub.NewUserService(busManager)

	// 1. 建立 CAN Client
	can := api.NewCANClient("01", "127.0.0.1", "8080" ,stationService)

	// 2. 等待 CAN Ready
	can.WaitForConnection()

	// 3. 建立 MQTT Client（並把 can 傳進去）
	mqttClient := api.NewMQTTClient(can,stationService)

	// 4. 開始訂閱指令
	mqttClient.Subscribe("charge_station/command")
	mqttSub := sub.NewMQTTEventSub(mqttClient)

	   	// 2. 建立所有 Handlers
   h:= &sub.Subs{
	StationEventHandler: &sub.StationEventHandler{},
	MqttSub: mqttSub,
}

   busManager.RegisterSubscribers(h.StationEventHandler)
   busManager.RegisterSubscribers(h.MqttSub)
   busManager.RegisterMiddlewares()
   

	

	select {} // 不讓 main 結束
}
