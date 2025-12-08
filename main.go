package main

import (
	"kenmec/jimmy/charge_core/api"
	"kenmec/jimmy/charge_core/log"
)

func main() {
	log.InitLog()
	// ⭐ 建立 CANManager
	canManager := api.NewCANManager()
	// ⭐ 設定多個站
	canManager.Add("01", "127.0.0.1", "8081")
	canManager.Add("02", "127.0.0.1", "8082")


	// 每個 station 可以獨立接 MQTT（如果你要）
	mqtt := api.NewMQTTClient()
	mqtt.Subscribe("charge_station/+/command")





	select {}
}
