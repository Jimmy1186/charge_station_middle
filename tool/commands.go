package tool

import (
	"encoding/hex"
	"fmt"

	"github.com/sigurn/crc16"
)


func Command(stationId, cmd string) (string, error) {
	
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