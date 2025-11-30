package api

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"kenmec/jimmy/charge_core/eventbusV2/events"
	"kenmec/jimmy/charge_core/eventbusV2/pub"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type  MQTT_Client struct {
	client  mqtt.Client
	can *CANClient

	configs MQTT_Config

	stationService *pub.StationService
}

type MQTT_Config struct {
	broker string 
	clientID string

	user    string
	password string 

	statusTopic  string 
}


func NewMQTTClient(can *CANClient, stationService *pub.StationService) *MQTT_Client{

	configs := MQTT_Config{
		broker: "mqtt://localhost:1883",
		clientID: "go_mqtt_client_charger",
		user: "admin",
		password: "admin",
		statusTopic: "charge_station/status",
	}


	opts := mqtt.NewClientOptions()

	opts.AddBroker(configs.broker)
	opts.SetClientID(configs.clientID)
	opts.SetUsername(configs.user)
	opts.SetPassword(configs.password)

	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(3 * time.Second)

	opts.SetConnectionLostHandler(func(c mqtt.Client, err error) {
		log.Printf("⚠️ MQTT 斷線: %v\n", err)
	})

	opts.SetOnConnectHandler(func(cli mqtt.Client) {
		log.Println("🔌 MQTT 已連線 / 已重新連線成功")
	})

	client := mqtt.NewClient(opts)

	token := client.Connect()
	token.Wait()

	if token.Error() != nil {
		log.Printf("❌ 連線失敗: %v\n", token.Error())
	} else {
		log.Println("✅ 成功連線到 MQTT Broker")
	}

	return &MQTT_Client{
		client: client,
		can: can,
		configs: configs,

		stationService: stationService,
	}

}


func(m *MQTT_Client) Subscribe (topic string) {

	token := m.client.Subscribe(topic, 0, func(c mqtt.Client, ms mqtt.Message) {
			payload := ms.Payload()
			log.Printf("📩 MQTT 收到命令: %s\n", payload)

		// ⭐ 呼叫 CAN 進行實際動作
		// err := m.can.SendCommand(payload)

	})

	token.Wait()
	if token.Error() != nil {
		fmt.Printf("❌ 訂閱主題 [%s] 失敗: %v\n", topic, token.Error())
		return 
	}
	fmt.Printf("✅ 成功訂閱主題: %s\n", topic)
}





func (m *MQTT_Client) PublishStatus(s events.StationStatus) {
    payload, _ := json.Marshal(s)

    token := m.client.Publish(m.configs.statusTopic, 0, false, payload)
    token.Wait()

    if token.Error() != nil {
        fmt.Println("❌ MQTT publish error:", token.Error())
    } else {
        fmt.Println("📤 MQTT published:", string(payload))
    }
}


