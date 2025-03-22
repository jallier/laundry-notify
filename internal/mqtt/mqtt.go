package mqtt

import (
	"context"
	"fmt"

	"github.com/charmbracelet/log"
	MQTT "github.com/eclipse/paho.mqtt.golang"
)

type Client = MQTT.Client

type MQTTManager struct {
	MqttOpts   *MQTT.ClientOptions
	mqttClient *MQTT.Client
	ctx        context.Context
	cancel     func()
}

func NewMqttOpts() *MQTT.ClientOptions {
	opts := MQTT.NewClientOptions()
	return opts
}

func NewMQTTManager() *MQTTManager {
	manager := &MQTTManager{
		MqttOpts:   nil,
		mqttClient: nil,
	}
	manager.ctx, manager.cancel = context.WithCancel(context.Background())
	return manager
}

func (m *MQTTManager) Connect() (MQTT.Token, error) {
	if m.MqttOpts == nil {
		return nil, fmt.Errorf("no mqtt options provided")
	}
	if m.mqttClient != nil {
		return nil, fmt.Errorf("already connected to MQTT broker")
	}

	// Log events
	m.MqttOpts.SetAutoReconnect(true)
	m.MqttOpts.OnConnectionLost = func(cl MQTT.Client, err error) {
		log.Info("mqtt connection lost")
	}
	m.MqttOpts.OnReconnecting = func(MQTT.Client, *MQTT.ClientOptions) {
		log.Info("mqtt attempting to reconnect")
	}

	newClient := MQTT.NewClient(m.MqttOpts)
	m.mqttClient = &newClient
	token := (*m.mqttClient).Connect()
	token.Wait()
	if token.Error() != nil {
		log.Error("Error connecting to MQTT broker: %v", token.Error())
		return nil, token.Error()
	}
	log.Debug("Connected to MQTT broker")
	return token, nil
}

func (m *MQTTManager) Disconnect() {
	(*m.mqttClient).Disconnect(250)
}

func (m *MQTTManager) Subscribe(topic string, eventChannel chan<- [2]string) error {
	log.Debug("Subscribing to MQTT topic...", "topic", topic)
	token := (*m.mqttClient).Subscribe(topic, byte(0), func(client MQTT.Client, msg MQTT.Message) {
		eventChannel <- [2]string{msg.Topic(), string(msg.Payload())}
	})
	token.Wait()
	if err := token.Error(); err != nil {
		log.Error("Error subscribing to MQTT topic: %v", token.Error())
		return err
	}
	return nil
}
