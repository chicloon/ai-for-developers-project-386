package slots

import (
	"fmt"
	"time"

	"call-booking/internal/models"
)

// GenerateSlots generates available time slots from schedules and bookings
func GenerateSlots(schedules []models.Schedule, bookings []models.Booking, date string) []models.Slot {
	// Parse the date to get day of week
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return []models.Slot{}
	}
	dayOfWeek := int32(t.Weekday()) // 0=Sunday

	// Filter schedules for this date
	var matchingSchedules []models.Schedule
	for _, schedule := range schedules {
		// Skip blocked schedules
		if schedule.IsBlocked {
			continue
		}

		// Check if schedule matches the date
		if schedule.Type == "recurring" && schedule.DayOfWeek != nil && *schedule.DayOfWeek == dayOfWeek {
			matchingSchedules = append(matchingSchedules, schedule)
		}
		if schedule.Type == "one-time" && schedule.Date != nil && *schedule.Date == date {
			matchingSchedules = append(matchingSchedules, schedule)
		}
	}

	// Default to 9:00-18:00 if no schedules
	if len(matchingSchedules) == 0 {
		defaultSchedule := models.Schedule{
			Type:      "recurring",
			DayOfWeek: &dayOfWeek,
			StartTime: "09:00",
			EndTime:   "18:00",
		}
		matchingSchedules = append(matchingSchedules, defaultSchedule)
	}

	// Generate slots from schedules
	var slots []models.Slot
	for _, schedule := range matchingSchedules {
		slots = append(slots, generateSlotsFromSchedule(schedule, date)...)
	}

	// Mark booked slots
	for i := range slots {
		for _, booking := range bookings {
			if booking.Status != "active" {
				continue
			}
			if isSlotBooked(slots[i], booking, date) {
				slots[i].IsBooked = true
				break
			}
		}
	}

	return slots
}

func generateSlotsFromSchedule(schedule models.Schedule, date string) []models.Slot {
	var slots []models.Slot

	current := schedule.StartTime
	for current < schedule.EndTime {
		end := addMinutes(current, 30)
		if end > schedule.EndTime {
			break
		}
		slot := models.Slot{
			ID:        fmt.Sprintf("%s_%s_%s", schedule.ID, date, current),
			Date:      date,
			StartTime: current,
			EndTime:   end,
			IsBooked:  false,
		}
		slots = append(slots, slot)
		current = end
	}

	return slots
}

func addMinutes(t string, minutes int) string {
	parsed, _ := time.Parse("15:04", t)
	result := parsed.Add(time.Duration(minutes) * time.Minute)
	return result.Format("15:04")
}

func isSlotBooked(slot models.Slot, booking models.Booking, date string) bool {
	return booking.Date == date && booking.StartTime == slot.StartTime
}
