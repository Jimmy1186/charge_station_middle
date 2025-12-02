package api

import (
	"context"
	"encoding/hex"
	"fmt"
	"net"
	"time"

	"kenmec/jimmy/charge_core/eventbusV2/events"
	"kenmec/jimmy/charge_core/eventbusV2/pub"
	klog "kenmec/jimmy/charge_core/log"
	"kenmec/jimmy/charge_core/tool"
)

type CANClient struct {
    stationId  string
    conn       net.Conn
    addr       string
    writeQueue chan []byte
    ctx        context.Context
    cancel     context.CancelFunc
	isReady    chan struct{}
    intervalStop chan struct {}

    stationService *pub.StationService
}

func NewCANClient(stationId string, ip string, port string, stationService *pub.StationService) *CANClient {
    ctx, cancel := context.WithCancel(context.Background())

    client := &CANClient{
        stationId:  stationId,
        addr:       net.JoinHostPort(ip, port),
        writeQueue: make(chan []byte, 100), // buffered channel
        ctx:        ctx,
        cancel:     cancel,
		isReady:    make(chan struct{}), 
        stationService: stationService,
    }

    go client.run() // main control goroutine

    return client
}

func (c *CANClient) run() {
    for {
        err := c.connect()
        if err != nil {
      
            klog.Logger.Error("Reconnect in 3 seconds...")
            klog.Logger.Error(fmt.Sprintf("can connect error: %e", err))
            time.Sleep(3 * time.Second)
            continue
        }

        // ---- 連線成功就啟動 interval ----
        c.startInterval()

        readDone := make(chan struct{})
        go c.readLoop(readDone)
        go c.writeLoop()

        select {
        case <-readDone:
            klog.Logger.Info("Connection lost, reconnecting...")
            c.stopInterval()  // <-- 斷線必須停掉 interval
            c.conn.Close()

        case <-c.ctx.Done():
            klog.Logger.Info("Shutting down CAN client...")
            c.stopInterval()  // <-- 關閉也必須停掉 interval
            c.conn.Close()
            return
        }
    }
}


func (c *CANClient) connect() error {
    klog.Logger.Error("try to connect")
    // 1. 使用 net.Dial。這會自動選擇一個本地的隨機埠來發送和接收數據。
    // c.addr 必須是遠端目標的 IP:Port (例如 "192.168.1.100:8080" 或 "127.0.0.1:8080")
    conn, err := net.Dial("udp", c.addr)
    if err != nil {
        klog.Logger.Error(fmt.Sprintf("Dial failed: %v", err))

        return err
    }
    c.conn = conn
    
    // 處理連線就緒通知 (保持您先前新增的邏輯)
    select {
    case <-c.isReady:
       
        // 已經關閉，通常是重連的情況，需要確保頻道再次被初始化
        // 由於 Go Channel 關閉後無法重新打開，我們需要一個更強健的狀態機制
        // 暫時保持不變，但請注意這是重連邏輯的潛在問題
    default:
        // 第一次連線成功，關閉頻道
        close(c.isReady) 
    }
    
    klog.Logger.Info(fmt.Sprintf("Connected to device: %s", c.addr))
    return nil
}


// 新增：等待連線建立完成
func (c *CANClient) WaitForConnection() {
    <-c.isReady
    klog.Logger.Info("CAN client is ready for commands.")
}

func (c *CANClient) readLoop(done chan struct{}) {
    buffer := make([]byte, 1024)

    for {
        n, err := c.conn.Read(buffer)
        if err != nil {
            klog.Logger.Error(fmt.Sprintf("Read error: %v", err))
            close(done)
            return
        }

	    c.handlePacket(buffer[:n])
    }
}

func (c *CANClient) handlePacket(pkt []byte) {
    if len(pkt) < 6 {
        klog.Logger.Error("packet too short")
		return
	}

	// fmt.Printf(
	// 	"Station=%d, Status=%08b, Error=%08b, Other=%08b\n",
	// 	pkt[0], pkt[3], pkt[4], pkt[5],
	// )

    payload := events.StationStatus{
        StationID: fmt.Sprintf("%d", pkt[0]),
        Status:    fmt.Sprintf("%08b", pkt[3]),
        Error:     fmt.Sprintf("%08b", pkt[4]),
        Other:     fmt.Sprintf("%08b", pkt[5]),
    }

    //送到event bus
    c.stationService.PubStationStatus(payload)

}

func (c *CANClient) writeLoop() {
    for {
        select {
        case msg := <-c.writeQueue:
            fmt.Printf("command send to CAN: %v/n", msg)
            _, err := c.conn.Write(msg)
            if err != nil {
                klog.Logger.Error(fmt.Sprintf("Write error: %v", err))
            }

        case <-c.ctx.Done():
            return
        }
    }
}

// Public API method
func (c *CANClient) SendCommand(cmd string) error {
    hexStr, err := tool.Command(c.stationId, cmd)
    if err != nil {
        return err
    }
    
  

    bytes, err := hex.DecodeString(hexStr)
    if err != nil {
        return err
    }

    c.writeQueue <- bytes // send to async goroutine
    return nil
}

func (c *CANClient) Close() {
    c.cancel()
}


func (c *CANClient) SendTextCommandToCAN(){
	messageHex := "800002"
	
	// 將十六進位字串轉換為位元組
	messageBytes, err := hex.DecodeString(messageHex)

    if err != nil{
        klog.Logger.Error(fmt.Sprintf("%v", err))
    }

_, err = c.conn.Write(messageBytes)
    if err != nil {
        klog.Logger.Error(fmt.Sprintf("發送數據失敗: %v", err))
        return
    }
}



func (c *CANClient) startInterval() {

    if c.intervalStop != nil {
        return // 已經在跑了，不要重複開
    }


    // 如果還沒建立 stop channel，就建立
    if c.intervalStop == nil {
        c.intervalStop = make(chan struct{})
    }

    go func(stop <-chan struct{}) {
        ticker := time.NewTicker(2 * time.Second)
        
        for {
            select {
            case <-ticker.C:
                c.SendCommand("read")

            case <-stop:
                ticker.Stop()
        
                return
            }
        }
    }(c.intervalStop)
}

func (c *CANClient) stopInterval() {
    if c.intervalStop != nil {
        close(c.intervalStop)
        c.intervalStop = nil
    }
}
