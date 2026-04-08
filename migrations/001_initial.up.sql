-- Step 1: Create tables without foreign keys
CREATE TABLE IF NOT EXISTS users (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), email VARCHAR(255) UNIQUE NOT NULL, password_hash VARCHAR(255) NOT NULL, name VARCHAR(255) NOT NULL, created_at TIMESTAMP NOT NULL DEFAULT NOW(), updated_at TIMESTAMP NOT NULL DEFAULT NOW());
CREATE TABLE IF NOT EXISTS visibility_groups (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), owner_id UUID NOT NULL, name VARCHAR(100) NOT NULL, visibility_level VARCHAR(20) NOT NULL CHECK (visibility_level IN ('family', 'work', 'friends', 'public')), created_at TIMESTAMP NOT NULL DEFAULT NOW());
CREATE TABLE IF NOT EXISTS group_members (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), group_id UUID NOT NULL, member_id UUID NOT NULL, added_by UUID NOT NULL, added_at TIMESTAMP NOT NULL DEFAULT NOW(), UNIQUE(group_id, member_id));
CREATE TABLE IF NOT EXISTS schedules (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), user_id UUID NOT NULL, type VARCHAR(20) NOT NULL CHECK (type IN ('recurring', 'one-time')), day_of_week INT CHECK (day_of_week BETWEEN 0 AND 6), date DATE, start_time TIME NOT NULL, end_time TIME NOT NULL, is_blocked BOOLEAN DEFAULT FALSE, created_at TIMESTAMP NOT NULL DEFAULT NOW(), CHECK (end_time > start_time), CHECK ((type = 'recurring' AND day_of_week IS NOT NULL AND date IS NULL) OR (type = 'one-time' AND date IS NOT NULL AND day_of_week IS NULL)));
CREATE TABLE IF NOT EXISTS bookings (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), schedule_id UUID NOT NULL, booker_id UUID NOT NULL, owner_id UUID NOT NULL, status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'cancelled')), created_at TIMESTAMP NOT NULL DEFAULT NOW(), cancelled_at TIMESTAMP, cancelled_by UUID);

-- Step 2: Add foreign keys (idempotent with IF NOT EXISTS via DO blocks)
DO $$ BEGIN ALTER TABLE visibility_groups ADD CONSTRAINT fk_visibility_groups_owner FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE; EXCEPTION WHEN duplicate_table THEN NULL; WHEN duplicate_object THEN NULL; WHEN OTHERS THEN NULL; END $$;
DO $$ BEGIN ALTER TABLE group_members ADD CONSTRAINT fk_group_members_group FOREIGN KEY (group_id) REFERENCES visibility_groups(id) ON DELETE CASCADE; EXCEPTION WHEN duplicate_table THEN NULL; WHEN duplicate_object THEN NULL; WHEN OTHERS THEN NULL; END $$;
DO $$ BEGIN ALTER TABLE group_members ADD CONSTRAINT fk_group_members_member FOREIGN KEY (member_id) REFERENCES users(id) ON DELETE CASCADE; EXCEPTION WHEN duplicate_table THEN NULL; WHEN duplicate_object THEN NULL; WHEN OTHERS THEN NULL; END $$;
DO $$ BEGIN ALTER TABLE group_members ADD CONSTRAINT fk_group_members_added_by FOREIGN KEY (added_by) REFERENCES users(id) ON DELETE CASCADE; EXCEPTION WHEN duplicate_table THEN NULL; WHEN duplicate_object THEN NULL; WHEN OTHERS THEN NULL; END $$;
DO $$ BEGIN ALTER TABLE schedules ADD CONSTRAINT fk_schedules_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE; EXCEPTION WHEN duplicate_table THEN NULL; WHEN duplicate_object THEN NULL; WHEN OTHERS THEN NULL; END $$;
DO $$ BEGIN ALTER TABLE bookings ADD CONSTRAINT fk_bookings_schedule FOREIGN KEY (schedule_id) REFERENCES schedules(id) ON DELETE CASCADE; EXCEPTION WHEN duplicate_table THEN NULL; WHEN duplicate_object THEN NULL; WHEN OTHERS THEN NULL; END $$;
DO $$ BEGIN ALTER TABLE bookings ADD CONSTRAINT fk_bookings_booker FOREIGN KEY (booker_id) REFERENCES users(id); EXCEPTION WHEN duplicate_table THEN NULL; WHEN duplicate_object THEN NULL; WHEN OTHERS THEN NULL; END $$;
DO $$ BEGIN ALTER TABLE bookings ADD CONSTRAINT fk_bookings_owner FOREIGN KEY (owner_id) REFERENCES users(id); EXCEPTION WHEN duplicate_table THEN NULL; WHEN duplicate_object THEN NULL; WHEN OTHERS THEN NULL; END $$;
DO $$ BEGIN ALTER TABLE bookings ADD CONSTRAINT fk_bookings_cancelled_by FOREIGN KEY (cancelled_by) REFERENCES users(id) ON DELETE SET NULL; EXCEPTION WHEN duplicate_table THEN NULL; WHEN duplicate_object THEN NULL; WHEN OTHERS THEN NULL; END $$;

-- Step 3: Create indexes
CREATE INDEX IF NOT EXISTS idx_schedules_user_id ON schedules(user_id);
CREATE INDEX IF NOT EXISTS idx_schedules_user_date ON schedules(user_id, date);
CREATE INDEX IF NOT EXISTS idx_bookings_booker_id ON bookings(booker_id);
CREATE INDEX IF NOT EXISTS idx_bookings_owner_id ON bookings(owner_id);
CREATE INDEX IF NOT EXISTS idx_bookings_schedule_id ON bookings(schedule_id);
CREATE INDEX IF NOT EXISTS idx_group_members_member_id ON group_members(member_id);
CREATE INDEX IF NOT EXISTS idx_visibility_groups_owner_id ON visibility_groups(owner_id);
