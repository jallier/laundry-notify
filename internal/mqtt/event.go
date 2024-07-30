package mqtt

import (
	MQTT "github.com/eclipse/paho.mqtt.golang"
)

type MQTTManager struct {
	mqttOpts *MQTT.ClientOptions
}
