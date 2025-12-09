package api

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"kenmec/jimmy/charge_core/config"
	eventbus "kenmec/jimmy/charge_core/infra"
	klog "kenmec/jimmy/charge_core/log"
	"kenmec/jimmy/charge_core/types"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MQTT_Client struct {
	client  mqtt.Client
	configs MQTT_Config
	eb      *eventbus.EventBus
	reqEb   *eventbus.RequestResponseBus
}

type MQTT_Config struct {
	broker   string
	clientID string

	user     string
	password string

	subscribeTopic []string
}

func NewMQTTClient(eb *eventbus.EventBus, reqEb *eventbus.RequestResponseBus, cfg *config.Config) *MQTT_Client {

	configs := MQTT_Config{
		broker:         "tcp://localhost:1883",
		clientID:       fmt.Sprintf("go_charger_%d", time.Now().UnixNano()),
		user:           "admin",
		password:       "admin",
		subscribeTopic: []string{"charge_station/+/command"},
	}
	m := &MQTT_Client{eb: eb, reqEb: reqEb, configs: configs}

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

		for _, v := range cfg.Stations {
			reqName := "tcp." + v.ID + ".status"

			// Check if handler exists before requesting
			if !reqEb.HasHandler(reqName) {
				klog.Logger.Warn(fmt.Sprintf("⚠️ Handler %s not ready yet, skipping status check", reqName))
				continue
			}

			response, err := reqEb.Request(reqName, types.ReqTCPStatus{})
			if err != nil {
				klog.Logger.Error(fmt.Sprintf("❌ Failed to get TCP status: %v", err))
				continue
			}
			data := response.Data.(types.ResTCPStatus)
			m.pubTpc(v.ID, data.IsConnect)
		}

		// Subscribe to topics...
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

	go m.subEb()
	go m.heartBeat()
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

func (m *MQTT_Client) subEb() {
	m.eb.Subscribe("connection.tcp", func(data interface{}) {
		d := data.(types.ConnectionTcp)

		pubData := types.ConnectionTcp{
			StationId: d.StationId,
			IsConnect: d.IsConnect,
			Msg:       d.Msg,
		}

		payload, err := json.Marshal(pubData)

		if err != nil {
			klog.Logger.Error(fmt.Sprintf("❌ Failed to marshal JSON payload: %v", err))
			return // Stop publish on error
		}

		klog.Logger.Info(fmt.Sprintf(`MQTT Send to QAMS is connect: %v, msg: %s`, d.IsConnect, d.Msg))

		prefixTopic := "charge_station/" + d.StationId + "/connection/tcp"

		token := m.client.Publish(prefixTopic, 0, true, payload)

		token.Wait()
		if token.Error() != nil {
			klog.Logger.Error(fmt.Sprintf("❌ Publish to topic [%s] failed: %v", "charge_station/connection/tcp", token.Error()))
		}
	})
}

func (m *MQTT_Client) pubTpc(stationId string, isConnect bool) {

	pubData := types.ConnectionTcp{
		StationId: stationId,
		IsConnect: isConnect,
		Msg:       "",
	}

	payload, err := json.Marshal(pubData)

	if err != nil {
		klog.Logger.Error(fmt.Sprintf("❌ Failed to marshal JSON payload: %v", err))
		return // Stop publish on error
	}

	klog.Logger.Info(fmt.Sprintf(`MQTT Send to QAMS is connect: %v`, isConnect))

	prefixTopic := "charge_station/" + stationId + "/connection/tcp"

	token := m.client.Publish(prefixTopic, 0, true, payload)
	token.Wait()
	if token.Error() != nil {
		klog.Logger.Error(fmt.Sprintf("❌ Publish to topic [%s] failed: %v", "charge_station/connection/tcp", token.Error()))
	}

}

func (m *MQTT_Client) heartBeat() {
	i := 0
	for range time.Tick(time.Second * 6) {

		token := m.client.Publish("charge_station/heartbeat", 0, true, string(i))
		token.Wait()
		if token.Error() != nil {
			klog.Logger.Error(fmt.Sprintf("❌ Publish to topic [%s] failed: %v", "charge_station/heartbeat", token.Error()))
		}
		i++
	}
}
