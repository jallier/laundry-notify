package laundryNotify

import "time"

type Event struct {
	Id         int
	Type       string
	StartedAt  time.Time
	FinishedAt time.Time
}

func (e *Event) Validate() error {
	if e.Type == "" {
		return Errorf(EINVALID, "Event type required.")
	}

	if e.StartedAt.IsZero() {
		return Errorf(EINVALID, "Event start time required.")
	}

	if e.FinishedAt.IsZero() {
		return Errorf(EINVALID, "Event finish time required.")
	}

	return nil
}
