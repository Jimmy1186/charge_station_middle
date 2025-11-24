package api

import (
	"encoding/hex"
	"fmt"
	"net"
)


func Tcp_connector() {
	// --- 設定參數 ---
	// 查找消息的目標 Port 埠號
	const targetPort = "8080"
	// 廣播地址 (發送到網路內所有設備)
	const broadcastIP = "192.168.1.1"
	const protocol = "udp"


	startChargeHex := "01050007FF003DFB"
	//stopChargeHex := "0105000700007C0B"
	
	// 將十六進位字串轉換為位元組
	messageBytes, err := hex.DecodeString(startChargeHex)
	if err != nil {
		fmt.Printf("解碼消息失敗: %v\n", err)
		return
	}

	// 設定目標地址 (廣播 IP + 8080 埠)
	targetAddress := net.JoinHostPort(broadcastIP, targetPort)

	// --- 建立連線 (UDP Client) ---
	// net.Dial("udp", ...) 建立一個 UDP 客戶端，用於發送數據
	conn, err := net.Dial(protocol, targetAddress)
	if err != nil {
		fmt.Printf("連線建立失敗 (UDP Dial): %v\n", err)
		return
	}
	defer conn.Close()

	// --- 發送消息 ---
	_, err = conn.Write(messageBytes)
	if err != nil {
		fmt.Printf("發送數據失敗: %v\n", err)
		return
	}
	
	// --- 接收響應 ---
	buffer := make([]byte, 1024) // 設置一個足夠大的緩衝區來接收數據包
	for {
		// 讀取數據：n 是實際讀取到的位元組數，addr 是發送方的地址
		n, err := conn.Read(buffer)
		if err != nil {
			fmt.Printf("❌ 讀取數據發生錯誤: %v\n", err)
			continue // 繼續下一次循環，避免因單次錯誤而中斷程式
		}
		
		// 處理接收到的數據
		parseDiscoveryResponse(buffer[:n])
		
	}

}

// --- 響應解析函式 ---
func parseDiscoveryResponse(data []byte) {
	


	fmt.Printf("--- 解析結果 (根據範例格式) ---\n")
	fmt.Printf("設備 TCP 埠號 : %X (原始數據位元組)\n", data)
}