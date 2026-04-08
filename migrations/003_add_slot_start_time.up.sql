-- Add slot_start_time column to bookings table
-- This stores the specific time slot that was booked
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS slot_start_time TIME;

-- Update existing bookings with default slot_start_time from schedule
UPDATE bookings
SET slot_start_time = s.start_time
FROM schedules s
WHERE bookings.schedule_id = s.id
  AND bookings.slot_start_time IS NULL;
