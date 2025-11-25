package main

import (
	"kenmec/jimmy/charge_core/api"
)

func main(){
    // 1. 建立 Client，異步開始嘗試連線
    client := api.NewCANClient("192.168.1.1", "8000")

    // 2. 🔥 等待連線成功：確保 client.conn 已經被賦值
    client.WaitForConnection()

    // 3. 現在可以安全地發送命令了
    client.SendTextCommandToCAN()

    // 為了讓 Goroutines 繼續運行，保持主程式不退出
    select {} 
}