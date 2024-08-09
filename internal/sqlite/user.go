package sqlite

import (
	"context"
	laundryNotify "jallier/laundry-notify"
	"strings"

	"github.com/charmbracelet/log"
)

// Ensure service implements interface.
var _ laundryNotify.UserService = (*UserService)(nil)

// UserService represents a service for managing users.
type UserService struct {
	db *DB
}

// NewUserService returns a new instance of UserService.
func NewUserService(db *DB) *UserService {
	return &UserService{db: db}
}

// FindUserByID retrieves a user by ID.
// Returns ENOTFOUND if user does not exist.
func (s *UserService) FindUserById(ctx context.Context, id int) (*laundryNotify.User, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		log.Error("failed to begin transaction", "error", err)
		return nil, err
	}
	defer tx.Rollback()
	user, err := findUserById(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) FindMostRecentUser(ctx context.Context) (*laundryNotify.User, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	user, err := findMostRecentUser(ctx, tx)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) CreateUser(ctx context.Context, user *laundryNotify.User) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := createUser(ctx, tx, user); err != nil {
		return err
	}

	return tx.Commit()
}

func createUser(ctx context.Context, tx *Tx, user *laundryNotify.User) error {
	user.CreatedAt.Time = tx.now

	if err := user.Validate(); err != nil {
		return err
	}

	res, err := tx.ExecContext(ctx, `
		INSERT INTO users (name, created_at)
		VALUES (?, ?)
	`,
		user.Name,
		(*NullTime)(&user.CreatedAt),
	)
	if err != nil {
		return err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	user.Id = int(id)

	return nil
}

// findUserByID is a helper function to fetch a user by ID.
// Returns ENOTFOUND if user does not exist.
func findUserById(ctx context.Context, tx *Tx, id int) (*laundryNotify.User, error) {
	a, _, err := findUsers(ctx, tx, laundryNotify.UserFilter{Id: &id})
	if err != nil {
		return nil, err
	} else if len(a) == 0 {
		// return nil, laundryNotify.ErrNotFound
		return nil, err
	}
	return a[0], nil
}

// findUsers returns a list of users matching a filter. Also returns a count of
// total matching users which may differ if filter.Limit is set.
func findUsers(ctx context.Context, tx *Tx, filter laundryNotify.UserFilter) (_ []*laundryNotify.User, n int, err error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := filter.Id; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}
	if v := filter.Name; v != nil {
		where, args = append(where, "name = ?"), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT 
			id, 
			name, 
			created_at,
			COUNT(*) OVER()
		FROM users 
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY id
		`+FormatLimitOffset(filter.Limit, filter.Offset),
		args...,
	)
	if err != nil {
		return nil, n, err
	}
	defer rows.Close()

	users := make([]*laundryNotify.User, 0)
	for rows.Next() {
		var user laundryNotify.User
		if err := rows.Scan(
			&user.Id,
			&user.Name,
			(*NullTime)(&user.CreatedAt),
			&n,
		); err != nil {
			return nil, n, err
		}
		users = append(users, &user)
	}
	if err = rows.Err(); err != nil {
		return nil, n, err
	}

	return users, n, nil
}

func findMostRecentUser(ctx context.Context, tx *Tx) (*laundryNotify.User, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT 
			name 
		FROM users
		ORDER BY created_at DESC
		LIMIT 1
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]*laundryNotify.User, 1)
	for rows.Next() {
		var user laundryNotify.User
		if err := rows.Scan(
			&user.Name,
		); err != nil {
			return nil, err
		}
		users = append(users, &user)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	if len(users) == 0 {
		return nil, nil
	}

	return users[0], nil
}
