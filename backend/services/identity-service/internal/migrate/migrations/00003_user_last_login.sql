ALTER TABLE users ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMPTZ;

UPDATE users AS target
SET last_login_at = source.last_login_at
FROM (
	SELECT user_id, MAX(created_at) AS last_login_at
	FROM sessions
	GROUP BY user_id
) AS source
WHERE target.id = source.user_id
	AND (target.last_login_at IS NULL OR target.last_login_at < source.last_login_at);
