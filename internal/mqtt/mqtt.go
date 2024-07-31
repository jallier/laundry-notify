package mqtt

import (
	"context"
	"fmt"

	"github.com/charmbracelet/log"
	MQTT "github.com/eclipse/paho.mqtt.golang"
)

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
