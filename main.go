package main

import (
	"fmt"
	"kenmec/jimmy/charge_core/api"
	"kenmec/jimmy/charge_core/eventbusV2/events"
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
	mqtt := api.NewMQTTClient(stationService)
	mqtt.Subscribe("charge_station/+/command")

	

	// Handlers 註冊  這邊只是集中訂閱者的資料
	mainStringSubs := &sub.Subs{
		MqttSub:             sub.NewMQTTEventSub(mqtt),
	}


	// busManager.StationEventBus.Subscribe(bus.FuncSub[events.StationStatus](func(e events.StationStatus) error {
	// 	// fmt.Println("Station:", e)
	// 	return nil
	// }))

	busManager.QamsCommandBus.Subscribe(bus.FuncSub[events.QamsCommand](func(e events.QamsCommand) error {
	    targetCan, ok:= canManager.Get(e.StationId)
		if ok == false {
			return fmt.Errorf("not found station")
		}

		targetCan.SendCommand(e.Cmd)
		return nil
	}))
	

    busManager.StationEventBus.Subscribe(mainStringSubs.MqttSub)
	
	busManager.RegisterMiddlewares()

	select {}
}
