-- Remove slot_start_time column from bookings table
ALTER TABLE bookings DROP COLUMN IF EXISTS slot_start_time;
