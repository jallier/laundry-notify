package laundryNotify

import "time"

type User struct {
	Id        int
	Name      string
	CreatedAt time.Time
}

func (u *User) Validate() error {
	if u.Name == "" {
		return Errorf(EINVALID, "User name required.")
	}
	return nil
}
