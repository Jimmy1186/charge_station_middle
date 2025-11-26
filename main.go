package main

import (
	"fmt"
	"kenmec/jimmy/charge_core/api"
)

func main() {

	// 1. 建立 CAN Client
	// can := api.NewCANClient("01", "127.0.0.1", "8080")

	// // 2. 等待 CAN Ready
	// can.WaitForConnection()

	// // 3. 建立 MQTT Client（並把 can 傳進去）
	// mqttClient := api.NewMQTTClient(can)

	// // 4. 開始訂閱指令
	// mqttClient.Subscribe("charge_station/command")

	client := api.NewTCPClient("01","192.168..0.168", 8899)

	client.OnConnect = func() {
			client.SendCommand("start")
		fmt.Println("🔥 已連線，可以開始讀取資料")
	}

	client.OnDisconnect = func() {
		fmt.Println("💥 斷線了，系統會自動重連")
	}


	

	select {} // 不讓 main 結束
}
