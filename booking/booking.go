package booking

import (
	"context"
	"encore.app/booking/db"
	"encore.dev/beta/errs"
	"encore.dev/storage/sqldb"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"slices"
	"time"
)

var (
	bookingDB = sqldb.NewDatabase("booking", sqldb.DatabaseConfig{
		Migrations: "./db/migrations",
	})

	pgxdb = sqldb.Driver[*pgxpool.Pool](bookingDB)
	query = db.New(pgxdb)
)

type Booking struct {
	ID    int64     `json:"id"`
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	Email string    `json:"email"`
}

type BookParams struct {
	Start time.Time `json:"start"`
	Email string    `encore:"sensitive"`
}

//encore:api public method=POST path=/booking
func Book(ctx context.Context, p *BookParams) error {

	now := time.Now()
	if p.Start.Before(now) {
		return errs.Wrap(&errs.Error{Code: errs.InvalidArgument}, "start time must be in the future")
	}

	tx, err := pgxdb.Begin(ctx)
	if err != nil {
		return errs.Wrap(&errs.Error{Code: errs.Unavailable, Message: err.Error()}, "failed to start transaction")
	}
	// Committed explicitly on success
	defer tx.Rollback(context.Background())

	// Get the bookings for this day.
	startOfDay := time.Date(p.Start.Year(), p.Start.Month(), p.Start.Day(), 0, 0, 0, 0, p.Start.Location())
	bookings, err := listBookingBetween(ctx, startOfDay, startOfDay.AddDate(0, 0, 1))
	if err != nil {
		return errs.Wrap(&errs.Error{Code: errs.Unavailable, Message: err.Error()}, "failed to list bookings")
	}
	// Is this slot bookable?
	slot := BookableSlot{Start: p.Start, End: p.Start.Add(DefaultBookingDuration)}
	if len(filterBookableSlots(ctx, []BookableSlot{slot}, now, bookings)) == 0 {
		return errs.Wrap(&errs.Error{Code: errs.Unavailable}, "slot is unavailable")
	}

	_, err = query.InsertBooking(ctx, db.InsertBookingParams{
		StartTime: pgtype.Timestamp{Time: p.Start, Valid: true},
		EndTime:   pgtype.Timestamp{Time: p.Start.Add(DefaultBookingDuration), Valid: true},
		Email:     p.Email,
	})
	if err != nil {
		return errs.Wrap(&errs.Error{Code: errs.Unavailable, Message: err.Error()}, "failed to insert booking")
	}
	err = tx.Commit(ctx)
	if err != nil {
		return errs.Wrap(&errs.Error{Code: errs.Unavailable, Message: err.Error()}, "failed to commit transaction")
	}
	return nil
}

type ListBookingResponse struct {
	Bookings []*Booking `json:"bookings"`
}

//encore:api auth method=GET path=/booking
func ListBookings(ctx context.Context) (*ListBookingResponse, error) {
	rows, err := query.ListBookings(ctx)
	if err != nil {
		return nil, err
	}

	var bookings []*Booking
	for _, row := range rows {
		bookings = append(bookings, &Booking{
			ID:    row.ID,
			Start: row.StartTime.Time,
			End:   row.EndTime.Time,
		})
	}
	return &ListBookingResponse{Bookings: bookings}, nil
}

//encore:api auth method=DELETE path=/booking/:id
func DeleteBooking(ctx context.Context, id int64) error {
	return query.DeleteBooking(ctx, id)
}

func listBookingBetween(ctx context.Context, start, end time.Time) ([]*Booking, error) {
	rows, err := query.ListBookingBetween(ctx, db.ListBookingBetweenParams{
		StartTime: pgtype.Timestamp{Time: start, Valid: true},
		EndTime:   pgtype.Timestamp{Time: end, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	var bookings []*Booking
	for _, row := range rows {
		bookings = append(bookings, &Booking{
			ID:    row.ID,
			Start: row.StartTime.Time,
			End:   row.EndTime.Time,
			Email: row.Email,
		})
	}
	return bookings, nil
}

func filterBookableSlots(_ context.Context, slots []BookableSlot, now time.Time, bookings []*Booking) []BookableSlot {
	// Remove slots which the start time has already passed
	slots = slices.DeleteFunc(slots, func(s BookableSlot) bool {
		// Has the slot already passed?
		if s.Start.Before(now) {
			return true
		}

		// Is there a booking that overlaps with this slot?
		for _, b := range bookings {
			if b.Start.Before(s.End) && b.End.After(s.Start) {
				return true
			}
		}

		return false
	})
	return nil
}
