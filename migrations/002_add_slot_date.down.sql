-- Remove slot_date column from bookings table
ALTER TABLE bookings DROP COLUMN IF EXISTS slot_date;
