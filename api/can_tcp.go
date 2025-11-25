package api

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/sigurn/crc16"
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
}

func NewCANClient(stationId, ip, port string) *CANClient {
    ctx, cancel := context.WithCancel(context.Background())

    client := &CANClient{
        stationId:  stationId,
        addr:       net.JoinHostPort(ip, port),
        writeQueue: make(chan []byte, 100), // buffered channel
        ctx:        ctx,
        cancel:     cancel,
		isReady:    make(chan struct{}), //
    }

    go client.run() // main control goroutine

    return client
}

func (c *CANClient) run() {
    for {
        err := c.connect()
        if err != nil {
            log.Println(err)
            log.Println("Reconnect in 3 seconds...")
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
            log.Println("Connection lost, reconnecting...")
            c.stopInterval()  // <-- 斷線必須停掉 interval
            c.conn.Close()

        case <-c.ctx.Done():
            log.Println("Shutting down CAN client...")
            c.stopInterval()  // <-- 關閉也必須停掉 interval
            c.conn.Close()
            return
        }
    }
}


func (c *CANClient) connect() error {
    // 1. 使用 net.Dial。這會自動選擇一個本地的隨機埠來發送和接收數據。
    // c.addr 必須是遠端目標的 IP:Port (例如 "192.168.1.100:8080" 或 "127.0.0.1:8080")
    conn, err := net.Dial("udp", c.addr)
    if err != nil {
        log.Printf("Dial failed: %v\n", err)
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
    
    log.Println("Connected to device:", c.addr)
    return nil
}


// 新增：等待連線建立完成
func (c *CANClient) WaitForConnection() {
    <-c.isReady
    log.Println("CAN client is ready for commands.")
}

func (c *CANClient) readLoop(done chan struct{}) {
    buffer := make([]byte, 1024)

    for {
        n, err := c.conn.Read(buffer)
        if err != nil {
            log.Println("Read error:", err)
            close(done)
            return
        }

        log.Printf("Received %d bytes: %X\n", n, buffer[:n])
    }
}

func (c *CANClient) writeLoop() {
    for {
        select {
        case msg := <-c.writeQueue:
            _, err := c.conn.Write(msg)
            if err != nil {
                log.Println("Write error:", err)
            }

        case <-c.ctx.Done():
            return
        }
    }
}

// Public API method
func (c *CANClient) SendCommand(cmd string) error {
    hexStr, err := command(c.stationId, cmd)
    if err != nil {
        return err
    }
    
    fmt.Printf(hexStr)

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
		log.Fatal(err)
	}

_, err = c.conn.Write(messageBytes)
	if err != nil {
		fmt.Printf("發送數據失敗: %v\n", err)
		return
	}
}






func command(stationId, cmd string) (string, error) {
	switch cmd {
	case "start":
		return buildCommand(stationId, "050007FF00")
	case "stop":
		return buildCommand(stationId, "0500070000")
	case "read":
		return buildCommand(stationId, "01004000")
	default:
		return "", fmt.Errorf("unknown cmd")
	}
}



var table = crc16.MakeTable(crc16.CRC16_MODBUS)

func buildCommand(stationId, payload string) (string, error) {
    fullStr := stationId + payload

    // 轉 []byte
    data, err := hex.DecodeString(fullStr)
    if err != nil {
        return "", fmt.Errorf("hex decode failed: %v", err)
    }

    // 計算 CRC16-MODBUS
    crcValue := crc16.Checksum(data, table)

    // 轉成大寫 HEX（4 字碼）
    crcHex := fmt.Sprintf("%04X", crcValue)

    // Little endian：低位在前
    crcLE := crcHex[2:4] + crcHex[0:2]

    return fullStr + crcLE, nil
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
        log.Println("Ticker started")

        for {
            select {
            case <-ticker.C:
                c.SendCommand("read")

            case <-stop:
                ticker.Stop()
                log.Println("Ticker stopped.")
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
