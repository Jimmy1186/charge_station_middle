package api

import (
	"fmt"
	"sync"
	"time"

	klog "kenmec/jimmy/charge_core/log"

	"github.com/gorilla/websocket"
)

type WS_Client struct {
	Conn        *websocket.Conn
	isConnected bool
	writeMu     sync.Mutex // 專門保護寫
	url         string
}
// Connect 建立 websocket 連線，會自動啟動重連機制
func Websocket_connect() (*WS_Client, error) {
	url := "ws://127.0.0.1:6000/peripheral/charge_station"

	client := &WS_Client{
		url: url,
	}

	err := client.connect()
	if err != nil {
		return nil, err
	}

	// 開 goroutine 持續讀取訊息
	go client.readLoop()

	return client, nil
}

// connect 建立連線（內部使用，可重試）
func (c *WS_Client) connect() error {
	for {
		conn, _, err := websocket.DefaultDialer.Dial(c.url, nil)
		if err != nil {
			klog.Logger.Error(fmt.Sprintf("connect failed, retrying in 2s... %v", err))
			time.Sleep(2 * time.Second)
			continue
		}

		c.writeMu.Lock()
		c.Conn = conn
		c.isConnected = true
		c.writeMu.Unlock()

		klog.Logger.Info("✅ Connected to TypeScript server")
		return nil
	}
}

// SendMessage 發訊息
func (c *WS_Client) SendMessage(msg string) error {
	if !c.isConnected || c.Conn == nil {
		return fmt.Errorf("not connected")
	}

	c.writeMu.Lock()         // 只鎖寫，不鎖讀
	defer c.writeMu.Unlock() // 確保同時只有一個 goroutine 寫

	return c.Conn.WriteMessage(websocket.TextMessage, []byte(msg))
}

// ReadData 從連線讀取單筆訊息
func (c *WS_Client) ReadData() (string, error) {
	if !c.isConnected || c.Conn == nil {
		return "", fmt.Errorf("not connected")
	}

		_, msg, err := c.Conn.ReadMessage() // 阻塞讀取
		if err != nil {
			c.isConnected = false
			klog.Logger.Error(fmt.Sprintf("read error: %v -> reconnecting", err))
			go c.connect()
			return "", err
		}

		klog.Logger.Info(fmt.Sprintf("📩 ReadData: %s", string(msg)))
	return string(msg), nil
}

// readLoop 持續印出訊息，非阻塞
func (c *WS_Client) readLoop() {
	for {
		msg, err := c.ReadData()
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}
		// 每次讀到訊息就印出
		klog.Logger.Info(fmt.Sprintf("Received: %s", msg))
	}
}

// Close 關閉連線
func (c *WS_Client) Close() error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	c.isConnected = false
	if c.Conn != nil {
		return c.Conn.Close()
	}
	return nil
}
