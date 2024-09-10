package booking

import (
	"context"
	"encore.app/booking/db"
	"encore.dev/beta/errs"
	"encore.dev/storage/sqldb"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
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
	ID    int       `json:"id"`
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
	eb := errs.B()

	now := time.Now()
	if p.Start.After(now) {
		return eb.Code(errs.InvalidArgument).Msg("start time must be in the future").Err()
	}

	tx, err := pgxdb.Begin(ctx)
	if err != nil {
		return eb.Cause(err).Code(errs.Unavailable).Msg("failed to start transaction").Err()
	}
	// Committed explicitly on success
	defer tx.Rollback(context.Background())

	_, err = query.InsertBooking(ctx, db.InsertBookingParams{
		StartTime: pgtype.Timestamp{Time: p.Start, Valid: true},
		EndTime:   pgtype.Timestamp{Time: p.Start.Add(DefaultBookingDuration), Valid: true},
		Email:     p.Email,
	})
	if err != nil {
		return eb.Cause(err).Code(errs.Unavailable).Msg("failed to insert booking").Err()
	}
	err = tx.Commit(ctx)
	if err != nil {
		return eb.Cause(err).Code(errs.Unavailable).Msg("failed to commit transaction").Err()
	}
	return nil
}
