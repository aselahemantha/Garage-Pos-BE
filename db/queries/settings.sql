-- name: GetAllSettings :many
SELECT key, value, updated_at
FROM settings;

-- name: GetSetting :one
SELECT key, value, updated_at
FROM settings
WHERE key = $1;

-- name: UpsertSetting :exec
INSERT INTO settings (key, value, updated_at)
VALUES ($1, $2, NOW())
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value,
    updated_at = NOW();
