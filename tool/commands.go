package tool

import (
	"encoding/hex"
	"fmt"
	"log"
)

func Command(stationId, cmd string) ([]byte, error) {

	switch cmd {
	case "start":
		return buildCommand(stationId, "00000000000001")
	case "stop":
		return buildCommand(stationId, "00000000000000")
	//  case "read":
	//  	return buildCommand(stationId, "00000000000077")
	default:
		return []byte{}, fmt.Errorf("unknown cmd")
	}
}

func buildCommand(stationId, payload string) ([]byte, error) {
	const startBuffer string = "00000800000f00"

	bufferedStr, err := hex.DecodeString(startBuffer + stationId + payload)

	if err != nil {
		log.Fatal("字串解析失敗，請確保字串長度為偶數且只包含 0-9, a-f:", err)
		return nil, err
	}

	checksum := calculateChecksum(bufferedStr)

	//fmt.Println(checksum, "check sum")
	bufferedStr = append(bufferedStr, checksum)

	return bufferedStr, nil

}

func calculateChecksum(data []byte) byte {
	var sum int
	// 遍歷前 15 個位元組
	for i := 0; i < 15; i++ {
		sum += int(data[i])
	}
	// 取得低 8 位 (Low 8 bits)
	return byte(sum & 0xFF)
}
