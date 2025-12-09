package types

type QamsCommand struct {
	StationId string
	Cmd       string
}

type ConnectionTcp struct {
	StationId string `json:"stationId"`
	IsConnect bool   `json:"isConnect"`
	Msg       string `json:"msg"`
}
