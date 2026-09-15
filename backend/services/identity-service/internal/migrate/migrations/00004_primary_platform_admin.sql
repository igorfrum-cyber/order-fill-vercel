ALTER TABLE users ADD COLUMN IF NOT EXISTS is_primary_admin BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE users
SET is_primary_admin = TRUE
WHERE id = (
	SELECT id
	FROM users
	WHERE role = 'platform_admin'
	ORDER BY created_at, id
	LIMIT 1
)
AND NOT EXISTS (SELECT 1 FROM users WHERE is_primary_admin);

CREATE UNIQUE INDEX IF NOT EXISTS users_one_primary_admin_uidx
	ON users (is_primary_admin)
	WHERE is_primary_admin;
