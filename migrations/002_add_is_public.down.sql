-- Restore public constraint
ALTER TABLE visibility_groups DROP CONSTRAINT IF EXISTS visibility_groups_visibility_level_check;
ALTER TABLE visibility_groups ADD CONSTRAINT visibility_groups_visibility_level_check 
    CHECK (visibility_level IN ('family', 'work', 'friends', 'public'));

-- Recreate public groups from is_public flag
INSERT INTO visibility_groups (owner_id, name, visibility_level, created_at)
SELECT id, 'Публичная', 'public', NOW() FROM users WHERE is_public = true
ON CONFLICT DO NOTHING;

-- Remove column
ALTER TABLE users DROP COLUMN IF EXISTS is_public;
DROP INDEX IF EXISTS idx_users_is_public;
