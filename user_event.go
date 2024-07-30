package laundryNotify

import "time"

type UserEvent struct {
	Id        int
	UserId    int
	EventId   int
	CreatedAt time.Time
}

func (u *UserEvent) Validate() error {
	if u.EventId <= 0 {
		return Errorf(EINVALID, "Event ID required.")
	}

	if u.UserId <= 0 {
		return Errorf(EINVALID, "User ID required.")
	}

	if u.CreatedAt.IsZero() {
		return Errorf(EINVALID, "UserEvent creation time required.")
	}

	return nil
}
