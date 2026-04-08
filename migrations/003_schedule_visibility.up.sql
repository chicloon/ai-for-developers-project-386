-- Create table linking schedules to visibility groups (many-to-many)
CREATE TABLE IF NOT EXISTS schedule_visibility_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    schedule_id UUID NOT NULL,
    group_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(schedule_id, group_id)
);

-- Add foreign keys
DO $$ BEGIN
    ALTER TABLE schedule_visibility_groups
    ADD CONSTRAINT fk_schedule_visibility_schedule
    FOREIGN KEY (schedule_id) REFERENCES schedules(id) ON DELETE CASCADE;
EXCEPTION
    WHEN duplicate_object THEN NULL;
    WHEN OTHERS THEN NULL;
END $$;

DO $$ BEGIN
    ALTER TABLE schedule_visibility_groups
    ADD CONSTRAINT fk_schedule_visibility_group
    FOREIGN KEY (group_id) REFERENCES visibility_groups(id) ON DELETE CASCADE;
EXCEPTION
    WHEN duplicate_object THEN NULL;
    WHEN OTHERS THEN NULL;
END $$;

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_schedule_visibility_schedule_id ON schedule_visibility_groups(schedule_id);
CREATE INDEX IF NOT EXISTS idx_schedule_visibility_group_id ON schedule_visibility_groups(group_id);
