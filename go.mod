module kenmec/jimmy/charge_core

go 1.24.0

toolchain go1.24.10

require (
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.3
	github.com/sigurn/crc16 v0.0.0-20240131213347-83fcde1e29d1
	go.uber.org/zap v1.27.1
)

require go.uber.org/multierr v1.10.0 // indirect

require (
	github.com/eclipse/paho.mqtt.golang v1.5.1
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
)
