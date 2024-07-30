package sqlite

import (
	"context"
	laundryNotify "jallier/laundry-notify"
	"strings"
)

// Ensure service implements interface.
var _ laundryNotify.EventService = (*EventService)(nil)

type EventService struct {
	db *DB
}

func NewEventService(db *DB) *EventService {
	return &EventService{db: db}
}

func (s *EventService) FindEventById(ctx context.Context, id int) (*laundryNotify.Event, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	event, err := findEventById(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	return event, nil
}

func (s *EventService) FindMostRecentEvent(ctx context.Context, userId int) (*laundryNotify.Event, error) {
	return nil, nil
}

func (s *EventService) CreateEvent(ctx context.Context, event *laundryNotify.Event) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := createEvent(ctx, tx, event); err != nil {
		return err
	}

	return tx.Commit()
}

func createEvent(ctx context.Context, tx *Tx, event *laundryNotify.Event) error {
	if err := event.Validate(); err != nil {
		return err
	}

	_, err := tx.ExecContext(
		ctx,
		`
		INSERT INTO events (type, started_at, finished_at) 
		VALUES (?, ?, ?)
		`,
		event.Type, event.StartedAt, event.FinishedAt,
	)
	return err
}

func findEventById(ctx context.Context, tx *Tx, id int) (*laundryNotify.Event, error) {
	a, _, err := findEvents(ctx, tx, laundryNotify.EventFilter{Id: &id})
	if err != nil {
		return nil, err
	}
	if len(a) == 0 {
		return nil, laundryNotify.Errorf(laundryNotify.ENOTFOUND, "Event not found: %d", id)
	}
	return a[0], nil
}

func findEvents(ctx context.Context, tx *Tx, filter laundryNotify.EventFilter) (_ []*laundryNotify.Event, n int, err error) {
	// Build WHERE clause
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := filter.Id; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT 
			id, 
			type, 
			started_at,
			finished_at,
			COUNT(*) OVER()
		FROM events
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY id
		`+FormatLimitOffset(filter.Limit, filter.Offset),
		args...,
	)
	if err != nil {
		return nil, n, err
	}
	defer rows.Close()

	events := make([]*laundryNotify.Event, 0)
	for rows.Next() {
		var event laundryNotify.Event
		if err := rows.Scan(
			&event.Id,
			&event.Type,
			(*NullTime)(&event.StartedAt),
			(*NullTime)(&event.FinishedAt),
			&n,
		); err != nil {
			return nil, n, err
		}
		events = append(events, &event)
	}
	if err = rows.Err(); err != nil {
		return nil, n, err
	}

	return events, n, nil
}
