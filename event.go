package laundryNotify

import (
	"context"
	"database/sql"
	"time"
)

const WASHER_EVENT = "washer"
const DRYER_EVENT = "dryer"

type Event struct {
	Id         int
	Type       string
	StartedAt  sql.NullTime
	FinishedAt sql.NullTime
	User       *User
}

func (e *Event) Validate() error {
	if e.Type == "" {
		return Errorf(EINVALID, "Event type required.")
	}

	if !e.StartedAt.Valid || e.StartedAt.Time.IsZero() {
		return Errorf(EINVALID, "Event start time required.")
	}

	// if e.FinishedAt.IsZero() {
	// 	return Errorf(EINVALID, "Event finish time required.")
	// }

	return nil
}

// Represents a set of fields to update on an event
type EventUpdate struct {
	FinishedAt sql.NullTime
}

type EventFilter struct {
	Id         *int
	Type       *string
	StartedAt  time.Time
	FinishedAt time.Time
	Limit      int
	Offset     int
	OrderBy    []string
}

type EventService interface {
	FindEventById(ctx context.Context, userId int) (*Event, error)
	FindMostRecentEvent(ctx context.Context, eventType string) (*Event, error)
	CreateEvent(ctx context.Context, event *Event) error
	UpdateEvent(ctx context.Context, id int, update EventUpdate) (*Event, error)
}

type UserEventFilter struct {
	Id *int
}

type UserEventService interface {
	FindUserEventById(ctx context.Context, id int) (*UserEvent, error)
	FindUserNamesByEventId(ctx context.Context, eventId int) ([]string, error)
	FindByUserName(ctx context.Context, name string, eventType string) ([]*UserEvent, int, error)
	FindUpcomingUserEvents(ctx context.Context, eventType string) ([]*UserEvent, int, error)
	CreateUserEvent(ctx context.Context, userEvent *UserEvent) error
	UpdateUserEvent(ctx context.Context, id int, update UserEventUpdate) (*UserEvent, error)
}
