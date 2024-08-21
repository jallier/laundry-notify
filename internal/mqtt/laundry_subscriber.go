package mqtt

import (
	"database/sql"
	"fmt"
	laundryNotify "jallier/laundry-notify"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var _ laundryNotify.LaundrySubscriberService = (*LaundrySubscriberService)(nil)

type LaundrySubscriberService struct {
	mqtt             *MQTTManager
	eventService     laundryNotify.EventService
	userEventService laundryNotify.UserEventService
	ntfyService      laundryNotify.LaundryNotifyService
}

func NewLaundrySubscriberService(
	mqtt *MQTTManager,
	eventService laundryNotify.EventService,
	userEventService laundryNotify.UserEventService,
	ntfyService laundryNotify.LaundryNotifyService,
) *LaundrySubscriberService {
	return &LaundrySubscriberService{
		mqtt:             mqtt,
		eventService:     eventService,
		userEventService: userEventService,
		ntfyService:      ntfyService,
	}
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
	if result == nil || result.FinishedAt.Valid {
		log.Debug("No existing unfinished event found, inserting new event")
		event := &laundryNotify.Event{
			Type:      eventType,
			StartedAt: sql.NullTime{Time: startedAt, Valid: true},
		}
		err := s.eventService.CreateEvent(s.mqtt.ctx, event)
		if err != nil {
			log.Error("Error creating new event", "error", err)
			return err
		}
		log.Info("New event inserted", "type", eventType, "started_at", startedAt, "id", event.Id)
		log.Debug("Checking for users subscribed to future event")
		userEvents, n, err := s.userEventService.FindUpcomingUserEvents(s.mqtt.ctx, eventType)
		if err != nil {
			log.Error("Error finding upcoming user events", "error", err)
			return err
		}
		log.Debug("Upcoming user events", "count", n, "events", userEvents)
		if n == 0 {
			log.Info("No users subscribed to future event")
			return nil
		}
		for _, userEvent := range userEvents {
			_, err = s.userEventService.UpdateUserEvent(s.mqtt.ctx, userEvent.Id, laundryNotify.UserEventUpdate{EventId: event.Id})
			if err != nil {
				log.Error("Error updating user event", "error", err)
				continue
			}
			log.Debug("Updated user events", "event_id", userEvent.Id)
		}
	} else {
		log.Info("existing event found, not adding event")
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
	mostRecentEvent, err := s.eventService.FindMostRecentEvent(s.mqtt.ctx, eventType)
	if err != nil {
		log.Error("Error finding most recent event", "error", err)
		return err
	}
	log.Debug("Most recent event", "result", mostRecentEvent)
	if mostRecentEvent.FinishedAt.Valid {
		log.Info("No existing unfinished event found, skipping")
		return nil
	}

	log.Debug("Existing unfinished event found, updating event")
	_, err = s.eventService.UpdateEvent(s.mqtt.ctx, mostRecentEvent.Id, laundryNotify.EventUpdate{
		FinishedAt: sql.NullTime{Time: finishedAt, Valid: true},
	})
	if err != nil {
		log.Error("Error updating existing event", "error", err)
		return err
	}
	log.Info("Existing event updated", "type", eventType, "finished_at", finishedAt)

	// Check the user events for any users that are subscribed to this event type
	usernames, err := s.userEventService.FindUserNamesByEventId(s.mqtt.ctx, mostRecentEvent.Id)
	if err != nil {
		log.Error("Error finding usernames by event id", "error", err)
		return err
	}

	for _, username := range usernames {
		topic := strings.ReplaceAll(username, " ", "_")
		title := fmt.Sprintf("%s event finished", toTitleCase(eventType))
		message := "Your laundry is ready!"
		err := s.ntfyService.Notify(topic, title, message)
		if err != nil {
			log.Error("Error notifying user", "error", err)
			return err
		}
		log.Info("User notified", "username", username, "eventId", mostRecentEvent.Id)
	}

	return nil
}

func toTitleCase(s string) string {
	return cases.Title(language.English).String(s)
}
