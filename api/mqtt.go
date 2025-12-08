package api

import (
	"fmt"
	"strings"
	"time"

	eventbus "kenmec/jimmy/charge_core/infra"
	klog "kenmec/jimmy/charge_core/log"
	"kenmec/jimmy/charge_core/types"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MQTT_Client struct {
	client  mqtt.Client
	configs MQTT_Config
	eb      *eventbus.EventBus
}

type MQTT_Config struct {
	broker   string
	clientID string

	user     string
	password string

	subscribeTopic []string
}

func NewMQTTClient(eb *eventbus.EventBus) *MQTT_Client {

	configs := MQTT_Config{
		broker:         "tcp://localhost:1883",
		clientID:       fmt.Sprintf("go_charger_%d", time.Now().UnixNano()),
		user:           "admin",
		password:       "admin",
		subscribeTopic: []string{"charge_station/+/command"},
	}
	m := &MQTT_Client{eb: eb, configs: configs}

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

		// 如果之前有訂閱過主題，重連後要補訂閱
		if len(m.configs.subscribeTopic) != 0 {
			for _, v := range m.configs.subscribeTopic {
				m.Subscribe(v)
				klog.Logger.Info("🔄 已自動重新訂閱主題: " + v)
			}
		}
	})

	m.client = mqtt.NewClient(opts)

	token := m.client.Connect()
	token.Wait()

	if token.Error() != nil {
		klog.Logger.Error(fmt.Sprintf("❌ 連線失敗: %v", token.Error()))

	} else {
		klog.Logger.Info("✅ 成功連線到 MQTT Broker")

	}

	return m
}

func (m *MQTT_Client) Subscribe(topic string) {

	token := m.client.Subscribe(topic, 0,
		func(c mqtt.Client, ms mqtt.Message) {
			//klog.Logger.Info(fmt.Sprintf("mqtt message meta: %+v", ms))

			topic := ms.Topic()
			payload := string(ms.Payload())

			parts := strings.Split(topic, "/")
			if len(parts) < 3 {
				klog.Logger.Error(fmt.Sprintf("❌ MQTT topic 格式錯誤: %s", topic))
				return
			}

			stationId := parts[1] // 第二段就是 stationId: 01, 02, ...

			klog.Logger.Info(fmt.Sprintf("📩 MQTT 收到給 [%s] 的命令: %s", stationId, payload))

			m.eb.Publish("qams.command", types.QamsCommand{
				StationId: stationId,
				Cmd:       payload,
			})

			//送到eventbus

		})

	token.Wait()
	if token.Error() != nil {
		klog.Logger.Error(fmt.Sprintf("❌ 訂閱主題 [%s] 失敗: %v", topic, token.Error()))
		return
	}
	klog.Logger.Info(fmt.Sprintf("✅ 成功訂閱主題: %s", topic))

}
