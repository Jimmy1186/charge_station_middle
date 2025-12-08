package main

import (
	"kenmec/jimmy/charge_core/api"
	eventbus "kenmec/jimmy/charge_core/infra"
	"kenmec/jimmy/charge_core/log"
)

func main() {
	log.InitLog()

	eb := eventbus.New()

	// ⭐ 建立 CANManager
	canManager := api.NewCANManager()
	// ⭐ 設定多個站
	canManager.Add("01", "127.0.0.1", "8000", eb)
	// canManager.Add("02", "127.0.0.1", "8080",eb)

	// 每個 station 可以獨立接 MQTT（如果你要）
	api.NewMQTTClient(eb)


	select {}
}
