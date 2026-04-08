-- Add slot_date column to bookings table to track the actual date of a booking
-- This is needed for recurring schedules which don't have a fixed date
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS slot_date DATE;

-- Update existing bookings to have a default slot_date based on created_at
UPDATE bookings SET slot_date = DATE(created_at) WHERE slot_date IS NULL;
