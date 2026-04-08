-- Add is_public flag to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_public BOOLEAN NOT NULL DEFAULT false;
CREATE INDEX IF NOT EXISTS idx_users_is_public ON users(is_public) WHERE is_public = true;

-- Migrate existing public groups to is_public flag
UPDATE users 
SET is_public = true 
WHERE EXISTS (
    SELECT 1 FROM visibility_groups 
    WHERE owner_id = users.id 
    AND visibility_level = 'public'
);

-- Remove public groups (they will be replaced by the flag)
DELETE FROM visibility_groups WHERE visibility_level = 'public';

-- Remove 'public' from valid values
ALTER TABLE visibility_groups DROP CONSTRAINT IF EXISTS visibility_groups_visibility_level_check;
ALTER TABLE visibility_groups ADD CONSTRAINT visibility_groups_visibility_level_check 
    CHECK (visibility_level IN ('family', 'work', 'friends'));
