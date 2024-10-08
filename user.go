package laundryNotify

import (
	"context"
	"database/sql"
)

type User struct {
	Id        int
	Name      string
	CreatedAt sql.NullTime
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
	FindMostRecentUsers(ctx context.Context, name string) ([]*User, int, error)
	FindUserByName(ctx context.Context, name string) (*User, error)
	CreateUser(ctx context.Context, user *User) error
}
