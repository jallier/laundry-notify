package sqlite

import (
	"context"
	laundryNotify "jallier/laundry-notify"
	"strings"
)

// Ensure service implements interface.
var _ laundryNotify.UserEventService = (*UserEventService)(nil)

type UserEventService struct {
	db *DB
}

func NewUserEventService(db *DB) *UserEventService {
	return &UserEventService{db: db}
}

func (s *UserEventService) FindUserEventById(ctx context.Context, id int) (*laundryNotify.UserEvent, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	event, err := findUserEventById(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	return event, nil
}

func (s *UserEventService) CreateUserEvent(ctx context.Context, userEvent *laundryNotify.UserEvent) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := createUserEvent(ctx, tx, userEvent); err != nil {
		return err
	}

	return tx.Commit()
}

func createUserEvent(ctx context.Context, tx *Tx, userEvent *laundryNotify.UserEvent) error {
	if err := userEvent.Validate(); err != nil {
		return err
	}

	_, err := tx.ExecContext(
		ctx,
		`
		INSERT INTO user_events (user_id, event_id, created_at) 
		VALUES (?, ?, ?)
		`,
		userEvent.UserId, userEvent.EventId, userEvent.CreatedAt,
	)
	return err
}

func findUserEventById(ctx context.Context, tx *Tx, id int) (*laundryNotify.UserEvent, error) {
	a, _, err := findUserEvents(ctx, tx, laundryNotify.UserEventFilter{Id: &id})
	if err != nil {
		return nil, err
	}
	if len(a) == 0 {
		return nil, laundryNotify.Errorf(laundryNotify.ENOTFOUND, "UserEvent not found: %d", id)
	}
	return a[0], nil
}

func findUserEvents(ctx context.Context, tx *Tx, filter laundryNotify.UserEventFilter) (_ []*laundryNotify.UserEvent, n int, err error) {
	// Build WHERE clause
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := filter.Id; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT 
			id,
			user_id,
			event_id,
			created_at,
			COUNT(*) OVER()
		FROM user_events
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY created_at DESC
		`+FormatLimitOffset(1, 0), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var a []*laundryNotify.UserEvent
	for rows.Next() {
		var o laundryNotify.UserEvent
		if err := rows.Scan(
			&o.Id,
			&o.UserId,
			&o.EventId,
			&o.CreatedAt,
			&n,
		); err != nil {
			return nil, 0, err
		}
		a = append(a, &o)
	}
	return a, len(a), nil
}
