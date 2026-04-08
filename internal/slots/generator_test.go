package slots

import (
	"testing"

	"call-booking/internal/models"
)

func ptrInt32(v int32) *int32 { return &v }
func strPtr(v string) *string  { return &v }

func TestGenerateSlotsFromSchedule(t *testing.T) {
	schedule := models.Schedule{
		Type:      "recurring",
		DayOfWeek: ptrInt32(1), // Monday
		StartTime: "10:00",
		EndTime:   "11:00",
		IsBlocked: false,
	}

	slots := generateSlotsFromSchedule(schedule, "2026-04-06") // Monday

	if len(slots) != 2 {
		t.Fatalf("expected 2 slots, got %d", len(slots))
	}
	if slots[0].StartTime != "10:00" || slots[0].EndTime != "10:30" {
		t.Fatalf("unexpected first slot: %+v", slots[0])
	}
	if slots[1].StartTime != "10:30" || slots[1].EndTime != "11:00" {
		t.Fatalf("unexpected second slot: %+v", slots[1])
	}
}

func TestGenerateSlots_MultipleSchedules(t *testing.T) {
	schedules := []models.Schedule{
		{
			Type:      "one-time",
			Date:      strPtr("2026-04-10"),
			StartTime: "10:00",
			EndTime:   "11:00",
			IsBlocked: false,
		},
		{
			Type:      "one-time",
			Date:      strPtr("2026-04-10"),
			StartTime: "14:00",
			EndTime:   "15:00",
			IsBlocked: false,
		},
	}

	slots := GenerateSlots(schedules, nil, "2026-04-10")

	if len(slots) != 4 {
		t.Fatalf("expected 4 slots, got %d", len(slots))
	}
}

func TestGenerateSlots_EmptyWhenNoSchedules(t *testing.T) {
	slots := GenerateSlots(nil, nil, "2026-04-06")
	// Default behavior: 9:00-18:00 = 18 slots (30 min intervals)
	if len(slots) != 18 {
		t.Fatalf("expected 18 slots (default 9-18), got %d", len(slots))
	}
}

func TestGenerateSlots_BlockedSchedule(t *testing.T) {
	schedule := models.Schedule{
		Type:      "recurring",
		DayOfWeek: ptrInt32(1),
		StartTime: "10:00",
		EndTime:   "11:00",
		IsBlocked: true,
	}

	slots := GenerateSlots([]models.Schedule{schedule}, nil, "2026-04-06")
	if len(slots) != 18 {
		// Should use default schedule (9-18) since blocked schedule is skipped
		t.Fatalf("expected 18 slots (blocked schedule should be skipped, using default), got %d", len(slots))
	}
}

func TestGenerateSlots_MarksBooked(t *testing.T) {
	schedule := models.Schedule{
		ID:        "sched-1",
		Type:      "recurring",
		DayOfWeek: ptrInt32(1),
		StartTime: "10:00",
		EndTime:   "11:00",
		IsBlocked: false,
	}
	bookings := []models.Booking{
		{
			Date:      "2026-04-06",
			StartTime: "10:00",
			Status:    "active",
		},
	}

	slots := GenerateSlots([]models.Schedule{schedule}, bookings, "2026-04-06")

	if len(slots) != 2 {
		t.Fatalf("expected 2 slots, got %d", len(slots))
	}
	if !slots[0].IsBooked {
		t.Error("expected first slot (10:00) to be marked as booked")
	}
	if slots[1].IsBooked {
		t.Error("expected second slot (10:30) to not be booked")
	}
}

func TestGenerateSlots_WrongDayRecurring(t *testing.T) {
	// Monday schedule on Tuesday
	schedule := models.Schedule{
		Type:      "recurring",
		DayOfWeek: ptrInt32(1), // Monday
		StartTime: "10:00",
		EndTime:   "11:00",
		IsBlocked: false,
	}

	// 2026-04-07 is Tuesday
	slots := GenerateSlots([]models.Schedule{schedule}, nil, "2026-04-07")

	// Should use default since the schedule doesn't match the day
	if len(slots) != 18 {
		t.Fatalf("expected 18 default slots (recurring on wrong day), got %d", len(slots))
	}
}

func TestGenerateSlots_OneTimeWrongDate(t *testing.T) {
	schedule := models.Schedule{
		Type:      "one-time",
		Date:      strPtr("2026-04-15"),
		StartTime: "10:00",
		EndTime:   "11:00",
		IsBlocked: false,
	}

	// Different date
	slots := GenerateSlots([]models.Schedule{schedule}, nil, "2026-04-16")

	// Should use default since the schedule doesn't match the date
	if len(slots) != 18 {
		t.Fatalf("expected 18 default slots (one-time on wrong date), got %d", len(slots))
	}
}

func TestAddMinutes(t *testing.T) {
	result := addMinutes("10:00", 30)
	if result != "10:30" {
		t.Fatalf("expected 10:30, got %s", result)
	}

	result = addMinutes("10:30", 30)
	if result != "11:00" {
		t.Fatalf("expected 11:00, got %s", result)
	}

	result = addMinutes("23:30", 30)
	if result != "00:00" {
		t.Fatalf("expected 00:00, got %s", result)
	}
}

func TestIsSlotBooked(t *testing.T) {
	slot := models.Slot{
		Date:      "2026-04-06",
		StartTime: "10:00",
	}
	booking := models.Booking{
		Date:      "2026-04-06",
		StartTime: "10:00",
		Status:    "active",
	}

	if !isSlotBooked(slot, booking, "2026-04-06") {
		t.Error("expected slot to be booked")
	}

	// Wrong date
	if isSlotBooked(slot, booking, "2026-04-07") {
		t.Error("expected slot not to be booked on different date")
	}

	// Wrong time
	wrongTimeBooking := models.Booking{
		Date:      "2026-04-06",
		StartTime: "10:30",
		Status:    "active",
	}
	if isSlotBooked(slot, wrongTimeBooking, "2026-04-06") {
		t.Error("expected slot not to be booked with wrong time")
	}
}

func TestIsSlotBooked_Cancelled(t *testing.T) {
	slot := models.Slot{
		Date:      "2026-04-06",
		StartTime: "10:00",
	}
	// Note: isSlotBooked doesn't check status - that's handled in GenerateSlots
	// This test verifies that isSlotBooked matches based on date/time regardless of status
	booking := models.Booking{
		Date:      "2026-04-06",
		StartTime: "10:00",
		Status:    "cancelled",
	}

	// isSlotBooked only checks date/time match, not status
	if !isSlotBooked(slot, booking, "2026-04-06") {
		t.Error("expected isSlotBooked to match cancelled booking (status checked in GenerateSlots)")
	}
}
