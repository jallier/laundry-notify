package mqtt

import (
	laundryNotify "jallier/laundry-notify"

	"github.com/charmbracelet/log"
)

var _ laundryNotify.LaundrySubscriberService = (*LaundrySubscriberService)(nil)

type LaundrySubscriberService struct {
	mqtt *MQTTManager
}

func NewLaundrySubscriberService(mqtt *MQTTManager) *LaundrySubscriberService {
	return &LaundrySubscriberService{mqtt: mqtt}
}

func (s *LaundrySubscriberService) Subscribe(topic string) {
	log.Debug("Subscribing to MQTT topic...")
}
