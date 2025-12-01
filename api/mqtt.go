package api

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"kenmec/jimmy/charge_core/eventbusV2/events"
	"kenmec/jimmy/charge_core/eventbusV2/pub"
	klog "kenmec/jimmy/charge_core/log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type  MQTT_Client struct {
	client  mqtt.Client
	can map[string]*CANClient

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


func NewMQTTClient(can map[string]*CANClient, stationService *pub.StationService) *MQTT_Client{

	configs := MQTT_Config{
		broker: "tcp://localhost:1883",
		clientID: fmt.Sprintf("go_charger_%d", time.Now().UnixNano()),
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
		klog.Logger.Warn(fmt.Sprintf("⚠️ MQTT 斷線: %v", err))
	})

	opts.SetOnConnectHandler(func(cli mqtt.Client) {

		klog.Logger.Info("🔌 MQTT 已連線 / 已重新連線成功")
	})

	client := mqtt.NewClient(opts)

	token := client.Connect()
	token.Wait()

	if token.Error() != nil {
		klog.Logger.Error(fmt.Sprintf("❌ 連線失敗: %v", token.Error()))
	
	} else {
		klog.Logger.Info("✅ 成功連線到 MQTT Broker")
	
	}

	return &MQTT_Client{
		client: client,
		can: can,
		configs: configs,

		stationService: stationService,
	}

}


func (m *MQTT_Client) Subscribe(topic string) {

    token := m.client.Subscribe(topic, 0,
        func(c mqtt.Client, ms mqtt.Message) {
			klog.Logger.Info(fmt.Sprintf("mqtt message meta: %+v", ms))
		
            topic := ms.Topic()
            payload := string(ms.Payload())
      
            parts := strings.Split(topic, "/") 
            if len(parts) < 3 {
				klog.Logger.Error(fmt.Sprintf("❌ MQTT topic 格式錯誤: %s", topic))
                return
            }
			
            stationId := parts[1] // 第二段就是 stationId: 01, 02, ...
		
			klog.Logger.Info(fmt.Sprintf("📩 MQTT 收到給 [%s] 的命令: %s", stationId, payload))

			// pubData := events.QamsCommand{
			// 	Cmd: payload,
			// }

			// m.stationService.PubQamsCommand(pubData)

            // 找出對應的 CAN client
            if canClient, ok := m.can[stationId]; ok {
                err := canClient.SendCommand(payload)
				if err != nil {
					klog.Logger.Error(fmt.Sprintf("❌ Station [%s] SendCommand error: %v", stationId, err))
					
				}
            } else {
				klog.Logger.Error(fmt.Sprintf("❌ 找不到 CAN station [%s]", stationId))
				
            }
	

        })

    token.Wait()
	if token.Error() != nil {
		klog.Logger.Error(fmt.Sprintf("❌ 訂閱主題 [%s] 失敗: %v", topic, token.Error()))
		return
	}
	klog.Logger.Info(fmt.Sprintf("✅ 成功訂閱主題: %s", topic))
	
}





func (m *MQTT_Client) PublishStatus(s events.StationStatus) {
    payload, _ := json.Marshal(s)

    token := m.client.Publish(m.configs.statusTopic, 0, false, payload)
    token.Wait()

	if token.Error() != nil {
		klog.Logger.Error(fmt.Sprintf("❌ MQTT publish error: %v", token.Error()))
	} else {
		klog.Logger.Info(fmt.Sprintf("📤 MQTT published: %s", string(payload)))
	}
}


