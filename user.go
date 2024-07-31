package laundryNotify

import (
	"context"
	"time"
)

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

type UserFilter struct {
	Id     *int
	Name   *string
	Limit  int
	Offset int
}

type UserService interface {
	FindUserById(ctx context.Context, id int) (*User, error)
}
