package main

import (
	"kenmec/jimmy/charge_core/api"
	"kenmec/jimmy/charge_core/config"
	eventbus "kenmec/jimmy/charge_core/infra"
	"kenmec/jimmy/charge_core/log"
)

func main() {
	log.InitLog()
	cfg, err := config.LoadConfig()

	if err != nil {
		panic("Could not load config")
	}

	eb := eventbus.New()
	reqbus := eventbus.NewReqBus()


	api.NewMQTTClient(eb, reqbus, cfg)

	// ⭐ 建立 CANManager
	canManager := api.NewCANManager()
	
// ⭐ 設定多個站
	for _, v := range cfg.Stations {
		canManager.Add(v.ID, v.IP, v.Port, eb, reqbus)
	}

	select {}
}
