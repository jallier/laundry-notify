package sqlite

import (
	"context"
	"database/sql"
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

func (s *UserEventService) FindUserNamesByEventId(ctx context.Context, eventId int) ([]string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	events, err := findUserNamesEventByEventId(ctx, tx, eventId)
	if err != nil {
		return nil, err
	}
	return events, nil
}

func (s *UserEventService) FindByUserName(ctx context.Context, name string, eventType string) ([]*laundryNotify.UserEvent, int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, 0, err
	}
	defer tx.Rollback()

	events, n, err := findByUserName(ctx, tx, name, eventType)
	if err != nil {
		return nil, 0, err
	}

	return events, n, nil
}

func (s *UserEventService) FindUpcomingUserEvents(ctx context.Context, eventType string) ([]*laundryNotify.UserEvent, int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, 0, err
	}
	defer tx.Rollback()

	events, n, err := findUpcomingUserEvents(ctx, tx, eventType)
	if err != nil {
		return nil, 0, err
	}

	return events, n, nil
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

func (s *UserEventService) UpdateUserEvent(ctx context.Context, id int, update laundryNotify.UserEventUpdate) (*laundryNotify.UserEvent, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	event, err := updateUserEvent(ctx, tx, id, update)
	if err != nil {
		return nil, err
	}

	return event, tx.Commit()
}

func createUserEvent(ctx context.Context, tx *Tx, userEvent *laundryNotify.UserEvent) error {
	time := sql.NullTime{
		Time:  tx.now,
		Valid: true,
	}
	userEvent.CreatedAt = time

	if err := userEvent.Validate(); err != nil {
		return err
	}

	_, err := tx.ExecContext(
		ctx,
		`
		INSERT INTO user_events (user_id, event_id, created_at, type) 
		VALUES (?, ?, ?, ?)
		`,
		userEvent.UserId, userEvent.EventId, userEvent.CreatedAt, userEvent.Type,
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

func findByUserName(ctx context.Context, tx *Tx, name string, eventType string) ([]*laundryNotify.UserEvent, int, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT
			ue.id,
			ue.user_id,
			ue.event_id,
			ue.created_at,
			ue.type,
			COUNT(*) OVER()
		FROM user_events ue
		JOIN users u ON u.id = ue.user_id
		WHERE u.name = ?
			AND ue.type = ?
			AND (
				ue.event_id IS NULL
				OR ue.event_id = 0
			)
		ORDER BY ue.created_at DESC
		`+FormatLimitOffset(5, 0),
		name,
		eventType,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []*laundryNotify.UserEvent
	var n int
	for rows.Next() {
		var e laundryNotify.UserEvent
		if err := rows.Scan(
			&e.Id,
			&e.UserId,
			&e.EventId,
			&e.CreatedAt,
			&e.Type,
			&n,
		); err != nil {
			return nil, 0, err
		}
		events = append(events, &e)
	}
	return events, n, nil
}

func findUpcomingUserEvents(ctx context.Context, tx *Tx, eventType string) ([]*laundryNotify.UserEvent, int, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT
			ue.id,
			ue.user_id,
			ue.event_id,
			ue.created_at,
			ue.type,
			COUNT(*) OVER()
		FROM user_events ue
		WHERE ue.type = ?
		AND (
			ue.event_id IS NULL
			OR ue.event_id = 0
		)	
		ORDER BY ue.created_at DESC
		`+FormatLimitOffset(5, 0),
		eventType,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []*laundryNotify.UserEvent
	var n int
	for rows.Next() {
		var e laundryNotify.UserEvent
		if err := rows.Scan(
			&e.Id,
			&e.UserId,
			&e.EventId,
			&e.CreatedAt,
			&e.Type,
			&n,
		); err != nil {
			return nil, 0, err
		}
		events = append(events, &e)
	}
	return events, n, nil
}

func findUserNamesEventByEventId(ctx context.Context, tx *Tx, eventId int) ([]string, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT 
			u.name
		FROM 
			user_events ue
		JOIN 
			users u ON u.id = ue.user_id
		WHERE 
			ue.event_id = ?
		`, eventId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var name []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		name = append(name, n)
	}

	return name, nil
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
			type,
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
		var ue laundryNotify.UserEvent
		if err := rows.Scan(
			&ue.Id,
			&ue.UserId,
			&ue.EventId,
			&ue.CreatedAt,
			&ue.Type,
			&n,
		); err != nil {
			return nil, 0, err
		}
		a = append(a, &ue)
	}
	return a, len(a), nil
}

func updateUserEvent(ctx context.Context, tx *Tx, id int, update laundryNotify.UserEventUpdate) (*laundryNotify.UserEvent, error) {
	userEvent, err := findUserEventById(ctx, tx, id)
	if err != nil {
		return nil, err
	}

	if v := update.EventId; v > 0 {
		userEvent.EventId = v
	}

	if err = userEvent.Validate(); err != nil {
		return nil, err
	}

	_, err = tx.ExecContext(
		ctx,
		`
		UPDATE user_events
		SET event_id = ?
		WHERE id = ?
		`,
		userEvent.EventId,
		id,
	)
	if err != nil {
		return nil, err
	}

	return userEvent, nil
}
