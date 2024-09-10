-- name: InsertBooking :one
INSERT INTO booking(start_time, end_time, email)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListBookingBetween :many
SELECT * FROM booking
WHERE start_time >= $1 AND end_time <= $2;

-- name: ListBookings :many
SELECT * FROM booking;

-- name: DeleteBooking :exec
DELETE FROM booking WHERE id = $1;