package api

import (
	"encoding/hex"
	"fmt"
	"kenmec/jimmy/charge_core/tool"
	"net"
	"sync"
	"time"

	klog "kenmec/jimmy/charge_core/log"
)

type TCPClient struct {
	stationId string
	address   string

	Conn      net.Conn
	IsConnect bool

	RetryDelay time.Duration

	Ready chan struct{} // ⭐ 每次連線成功就會 close，一次性事件

	mu sync.Mutex

	// callback（可選）
	OnConnect    func()
	OnDisconnect func()
}



func NewTCPClient(stationId, ip string, port int) *TCPClient {
	c := &TCPClient{
		stationId:  stationId,
		address:    fmt.Sprintf("%s:%d", ip, port),
		RetryDelay: 3 * time.Second,
		Ready:      make(chan struct{}), // 第一次 Ready
	}

	go c.connectLoop()
	return c
}



func (c *TCPClient) connectLoop() {
	for {
		// 回圈起點：如果 Ready 已經 close → 重建
		if c.Ready == nil {
			c.Ready = make(chan struct{})
		}

		conn, err := net.Dial("tcp", c.address)
		if err != nil {
		 
			klog.Logger.Warn(fmt.Sprintf("❌ TCP connect failed:", err))
			time.Sleep(c.RetryDelay)
			continue
		}

		// ---- 連線成功 ----
		c.mu.Lock()
		c.Conn = conn
		c.IsConnect = true
		close(c.Ready) // 通知所有等待 "Ready" 的 goroutine
		c.Ready = nil  // 標記為已經通知過
		c.mu.Unlock()

		klog.Logger.Info(fmt.Sprintf("✅ TCP connected:", c.address))

		if c.OnConnect != nil {
			c.OnConnect()
		}

		// ---- 開始 read loop ----
		c.readLoop()

		// ---- read loop 結束 = 斷線 ----
		c.mu.Lock()
		c.IsConnect = false
		c.Conn.Close()
		c.mu.Unlock()

		klog.Logger.Warn(fmt.Sprintf("⚠️ TCP disconnected, retrying..."))

		if c.OnDisconnect != nil {
			c.OnDisconnect()
		}

		// 重連前 sleep
		time.Sleep(c.RetryDelay)
	}
}

// ⭐ 避免阻塞、可安全退出的 readLoop
func (c *TCPClient) readLoop() {
	buf := make([]byte, 1024)

	for {
		c.mu.Lock()
		conn := c.Conn
		c.mu.Unlock()

		n, err := conn.Read(buf)
		if err != nil {
			// readLoop 退出 → 代表斷線
			klog.Logger.Error(fmt.Sprintf("❌ TCP read error:", err))
			return
		}

		c.handlePacket(buf[:n])
	}
}

func (c *TCPClient) handlePacket(pkt []byte) {
	if len(pkt) < 6 {
		klog.Logger.Error("packet too short")
		return
	}
	// TODO：你也可以丟給 MQTT 這裡
}

func (c *TCPClient) SendCommand(cmd string) error {
	if !c.IsConnect {
		return fmt.Errorf("not connected")
	}

	hexStr, err := tool.Command(c.stationId, cmd)
    if err != nil {
        return err
    }
    
  
    bytes, err := hex.DecodeString(hexStr)
    if err != nil {
        return err
    }


	_, err = c.Conn.Write(bytes)
	
	return err
}


// func parsePacket(pkt []byte) (events.StationStatus, bool) {
//     if len(pkt) < 6 {
//         fmt.Println("packet too short")
//         return events.StationStatus{}, false
//     }

//     st := events.StationStatus{
//         StationID: pkt[0],
//         Status:    fmt.Sprintf("%08b", pkt[3]),
//         Error:     fmt.Sprintf("%08b", pkt[4]),
//         Other:     fmt.Sprintf("%08b", pkt[5]),
//     }

//     fmt.Printf(
//         "Station=%d, Status=%s, Error=%s, Other=%s\n",
//         st.StationID, st.Status, st.Error, st.Other,
//     )

//     return st, true
// }

func (c *TCPClient) Send(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.IsConnect {
		return fmt.Errorf("not connected")
	}
	_, err := c.Conn.Write(data)
	return err
}