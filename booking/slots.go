// Package booking Service booking keeps track of bookable slots in the calendar
package booking

import (
	"context"
	"errors"
	"time"
)

const DefaultBookingDuration = 1 * time.Hour

type BookableSlot struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type SlotsParams struct{}

type SlotsResponse struct {
	Slots []BookableSlot `json:"slots"`
}

//encore:api public method=GET path=/slots/:from
func GetBookableSlots(ctx context.Context, from string) (*SlotsResponse, error) {
	fromDate, err := time.Parse("2006-01-02", from)
	if err != nil {
		return nil, err
	}

	availabilityResp, err := GetAvailability(ctx)
	if err != nil {
		return nil, err
	}
	availability := availabilityResp.Availability

	const numDays = 7

	var slots []BookableSlot
	for i := 0; i < numDays; i++ {
		date := fromDate.AddDate(0, 0, i)
		weekday := int(date.Weekday())
		if len(availability) <= weekday {
			break
		}
		daySlots, err := bookableSlotsFromDay(date, &availability[weekday])
		if err != nil {
			return nil, err
		}
		slots = append(slots, daySlots...)

	}
	// Get bookings for next 7 days.
	activeBookings, err := listBookingBetween(ctx, fromDate, fromDate.AddDate(0, 0, numDays))
	if err != nil {
		return nil, err
	}

	slots = filterBookableSlots(ctx, slots, time.Now(), activeBookings)

	return &SlotsResponse{Slots: slots}, nil
}

func bookableSlotsFromDay(date time.Time, avail *Availability) ([]BookableSlot, error) {
	if avail.Start == nil || avail.End == nil {
		return nil, nil
	}

	availStartTime, err1 := strToTime(avail.Start)
	availEndTime, err2 := strToTime(avail.End)
	if err := errors.Join(err1, err2); err != nil {
		return nil, err
	}

	availStart := date.Add(time.Duration(availStartTime.Microseconds) * time.Microsecond)
	availEnd := date.Add(time.Duration(availEndTime.Microseconds) * time.Microsecond)

	// Compute the bookable slots in this day, based on availability.
	var slots []BookableSlot
	start := availStart
	for {
		end := start.Add(DefaultBookingDuration)
		if start.After(availEnd) {
			break
		}
		slots = append(slots, BookableSlot{
			Start: start,
			End:   end,
		})
		start = end
	}
	return slots, nil
}
