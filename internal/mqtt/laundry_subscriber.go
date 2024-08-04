package mqtt

import (
	"database/sql"
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
				s.finishExistingEvent(leafTopic, messageValue)
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

	log.Info("New event received", "type", eventType, "started_at", startedAt)

	// Check if we have an existing event of the same type that hasn't been finished - this means we should just skip this event since its likely a doubleup of the same event
	result, err := s.eventService.FindMostRecentEvent(s.mqtt.ctx, eventType)
	if err != nil {
		log.Error("Error finding most recent event", "error", err)
		return err
	}
	log.Debug("Most recent event", "result", result)
	if result == nil || !result.FinishedAt.Valid {
		log.Debug("No existing unfinished event found, inserting new event")
		err := s.eventService.CreateEvent(s.mqtt.ctx, &laundryNotify.Event{
			Type:      eventType,
			StartedAt: sql.NullTime{Time: startedAt, Valid: true},
		})
		if err != nil {
			log.Error("Error creating new event", "error", err)
			return err
		}
		log.Info("New event inserted", "type", eventType, "started_at", startedAt)
	} else {
		log.Info("existing event found, skipping")
		return nil
	}

	return nil
}

func (s *LaundrySubscriberService) finishExistingEvent(eventType string, finishedAtTimestamp string) error {
	finishedAt, err := time.Parse(time.RFC3339, finishedAtTimestamp)
	if err != nil {
		log.Error("Error parsing finished_at timestamp", "error", err)
		return err
	}

	log.Info("New event received", "type", eventType, "finished_at", finishedAt)

	// Check if we have an existing event of the same type that hasn't been finished - this means we should just skip this event since its likely a doubleup of the same event
	result, err := s.eventService.FindMostRecentEvent(s.mqtt.ctx, eventType)
	if err != nil {
		log.Error("Error finding most recent event", "error", err)
		return err
	}
	log.Debug("Most recent event", "result", result)
	if !result.FinishedAt.Valid {
		log.Debug("Existing unfinished event found, updating event")
		_, err := s.eventService.UpdateEvent(s.mqtt.ctx, result.Id, laundryNotify.EventUpdate{
			FinishedAt: sql.NullTime{Time: finishedAt, Valid: true},
		})
		if err != nil {
			log.Error("Error updating existing event", "error", err)
			return err
		}
		log.Info("Existing event updated", "type", eventType, "finished_at", finishedAt)
	} else {
		log.Info("No existing unfinished event found, skipping")
		return nil
	}

	return nil
}
