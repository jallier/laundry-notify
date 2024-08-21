package laundryNotify

import (
	"database/sql"
)

type UserEvent struct {
	Id        int
	UserId    int
	EventId   int
	CreatedAt sql.NullTime
	Type      string
}

func (u *UserEvent) Validate() error {
	if u.UserId <= 0 {
		return Errorf(EINVALID, "User ID required.")
	}

	if !u.CreatedAt.Valid {
		return Errorf(EINVALID, "UserEvent creation time required.")
	}

	if u.Type == "" {
		return Errorf(EINVALID, "UserEvent type required.")
	}

	return nil
}

type UserEventUpdate struct {
	EventId int
}
