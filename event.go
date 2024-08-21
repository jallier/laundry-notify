package laundryNotify

import (
	"database/sql"
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
