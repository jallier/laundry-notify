package mqtt

import (
	laundryNotify "jallier/laundry-notify"
	"strings"
	"time"

	"github.com/charmbracelet/log"
)

var _ laundryNotify.LaundrySubscriberService = (*LaundrySubscriberService)(nil)

type LaundrySubscriberService struct {
	mqtt         *MQTTManager
	eventService laundryNotify.EventService
}

func NewLaundrySubscriberService(
	mqtt *MQTTManager,
	eventService laundryNotify.EventService,
) *LaundrySubscriberService {
	return &LaundrySubscriberService{mqtt: mqtt, eventService: eventService}
}

func (s *LaundrySubscriberService) Subscribe(topic string) {
	eventsChannel := make(chan [2]string)
	err := s.mqtt.Subscribe(topic, eventsChannel)
	if err != nil {
		log.Error("Error subscribing to MQTT topic", "topic", topic, "error", err)
		return
	}

	go func() {
		for incomingEvent := range eventsChannel {
			log.Debug("Received event", "topic", incomingEvent[0], "payload", incomingEvent[1])

			topicSlice := strings.Split(incomingEvent[0], "/")
			leafTopic := topicSlice[len(topicSlice)-1]

			messageSlice := strings.Split(incomingEvent[1], "=")
			messageKey := messageSlice[0]
			messageValue := messageSlice[1]

			switch messageKey {
			case "started_at":
				s.addNewEvent(leafTopic, messageValue)
			case "finished_at":
				log.Debug("Event finished", "topic", leafTopic, "finished_at", messageValue)
			}
		}
	}()
}

func (s *LaundrySubscriberService) addNewEvent(eventType string, startedAtTimestamp string) error {
	startedAt, err := time.Parse(time.RFC3339, startedAtTimestamp)
	if err != nil {
		log.Error("Error parsing started_at timestamp", "error", err)
		return err
	}

	log.Debug("New event", "type", eventType, "started_at", startedAt)

	// Check if we have an existing event of the same type that hasn't been finished - this means we should just skip this event since its likely a doubleup of the same event
	result, err := s.eventService.FindMostRecentEvent(s.mqtt.ctx, eventType)
	if err != nil {
		log.Error("Error finding most recent event", "error", err)
		return err
	}
	log.Debug("Most recent event", "result", result)
	if result == nil || !result.FinishedAt.IsZero() {
		log.Debug("No existing event found, inserting new event")
	} else {
		log.Debug("existing event found, skipping")
	}

	return nil
}

func finishExistingEvent(eventType string, startedAt time.Time) {}
