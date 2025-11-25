package api

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"time"
)

type CANClient struct {
    conn       net.Conn
    addr       string
    writeQueue chan []byte
    ctx        context.Context
    cancel     context.CancelFunc
	isReady    chan struct{}
}

func NewCANClient(ip, port string) *CANClient {
    ctx, cancel := context.WithCancel(context.Background())

    client := &CANClient{
        addr:       net.JoinHostPort(ip, port),
        writeQueue: make(chan []byte, 100), // buffered channel
        ctx:        ctx,
        cancel:     cancel,
		isReady:    make(chan struct{}), //
    }

    go client.run() // main control goroutine

    return client
}

// Central controller goroutine
func (c *CANClient) run() {
    for {
        err := c.connect()
        if err != nil {
            log.Println("Reconnect in 3 seconds...")
            time.Sleep(3 * time.Second)
            continue
        }

        // Connected successfully → run read/write loops
        readDone := make(chan struct{})
        go c.readLoop(readDone)
        go c.writeLoop()

        // Wait for disconnection or shutdown
        select {
        case <-readDone:
            log.Println("Connection lost, reconnecting...")
            c.conn.Close()
        case <-c.ctx.Done():
            log.Println("Shutting down CAN client...")
            c.conn.Close()
            return
        }
    }
}

func (c *CANClient) connect() error {
    conn, err := net.Dial("udp", c.addr)
    if err != nil {
        log.Printf("Dial failed: %v\n", err)
        return err
    }
    c.conn = conn

    log.Println("Connected to device:", c.addr)
    
    // 🔥 新增：成功連線後，立即關閉 isReady 頻道，通知主程式可以發送了
    select {
    case <-c.isReady:
        // 已經關閉，不做任何事 (處理重連)
    default:
        // 第一次連線成功，關閉頻道
        close(c.isReady) 
    }
    
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
    hexStr, err := command(cmd)
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
		log.Fatal(err)
	}

_, err = c.conn.Write(messageBytes)
	if err != nil {
		fmt.Printf("發送數據失敗: %v\n", err)
		return
	}
}


func command(cmd string) (string, error){
	startChargeHex := "01050007FF003DFB"
	stopChargeHex := "0105000700007C0B"
	
	switch(cmd){
	case "start":
		return startChargeHex, nil
	case "stop":
		return stopChargeHex, nil
	default:
		return  "", fmt.Errorf("not found command")
	}
}


