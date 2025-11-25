package main

import (
	"kenmec/jimmy/charge_core/api"
)

func main(){
    // 1. 建立 Client，異步開始嘗試連線
    client := api.NewCANClient("01","127.0.0.1", "8080")

    // 2. 🔥 等待連線成功：確保 client.conn 已經被賦值
    client.WaitForConnection()


    client.IntervalSendReadStatus()

    // time.Sleep(12 *time.Second)

    //  close(stopper)

    // 為了讓 Goroutines 繼續運行，保持主程式不退出
    select {} 
}

