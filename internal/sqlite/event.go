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

func (s *EventService) FindMostRecentEvent(ctx context.Context, eventType string) (*laundryNotify.Event, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	event, err := findMostRecentEvent(ctx, tx, eventType)
	if err != nil {
		return nil, err
	}
	return event, nil
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

func (s *EventService) UpdateEvent(ctx context.Context, id int, upd laundryNotify.EventUpdate) (*laundryNotify.Event, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	event, err := updateEvent(ctx, tx, id, upd)
	if err != nil {
		return event, err
	}

	return event, tx.Commit()
}

func updateEvent(ctx context.Context, tx *Tx, id int, upd laundryNotify.EventUpdate) (*laundryNotify.Event, error) {
	event, err := findEventById(ctx, tx, id)
	if err != nil {
		return event, err
	}

	if v := upd.FinishedAt; v.Valid {
		event.FinishedAt = v
	}

	if err := event.Validate(); err != nil {
		return event, err
	}

	_, err = tx.ExecContext(
		ctx,
		`
		UPDATE events
		SET finished_at = ?
		WHERE id = ?
		`,
		(*NullTime)(&event.FinishedAt),
		event.Id,
	)
	if err != nil {
		return event, err
	}

	return event, nil
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
	events, _, err := findEvents(ctx, tx, laundryNotify.EventFilter{Id: &id})
	if err != nil {
		return nil, err
	}
	if len(events) == 0 {
		return nil, laundryNotify.Errorf(laundryNotify.ENOTFOUND, "Event not found: %d", id)
	}
	return events[0], nil
}

func findMostRecentEvent(ctx context.Context, tx *Tx, eventType string) (*laundryNotify.Event, error) {
	events, _, err := findEvents(
		ctx,
		tx,
		laundryNotify.EventFilter{
			Type:    &eventType,
			OrderBy: []string{"started_at DESC"},
		},
	)
	if err != nil {
		return nil, err
	}
	if len(events) == 0 {
		return nil, nil
	}
	return events[0], nil
}

func findEvents(ctx context.Context, tx *Tx, filter laundryNotify.EventFilter) (_ []*laundryNotify.Event, n int, err error) {
	// Build WHERE clause
	where, whereArgs := []string{"1 = 1"}, []interface{}{}
	if v := filter.Id; v != nil {
		where, whereArgs = append(where, "id = ?"), append(whereArgs, *v)
	} else if v := filter.Type; v != nil {
		where, whereArgs = append(where, "type = ?"), append(whereArgs, *v)
	} else if v := filter.StartedAt; !v.IsZero() {
		where, whereArgs = append(where, "started_at = ?"), append(whereArgs, &v)
	} else if v := filter.FinishedAt; !v.IsZero() {
		where, whereArgs = append(where, "finished_at = ?"), append(whereArgs, &v)
	}

	// Build ORDER BY clause
	orderBy := []string{"id"}
	if filter.OrderBy != nil {
		orderBy = filter.OrderBy
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
		`+FormatOrderBy(orderBy)+`
		`+FormatLimitOffset(filter.Limit, filter.Offset),
		whereArgs...,
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
