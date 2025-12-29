-- name: CreateSession :exec
INSERT INTO account.user_session (
    token,
    user_id,
    user_agent,
    ip_address,
    revoked,
    revoked_at,
    expires_at,
    role 
) 
VALUES (
    $1, 
    $2, 
    $3, 
    $4,
    $5, 
    $6, 
    $7,
    $8
);

-- name: GetSession :one
SELECT
    us.id AS session_id,
    us.token,
    us.user_id,
    us.user_agent,
    us.role ,
    us.ip_address,
    us.revoked,
    us.revoked_at,
    us.expires_at,
    us.created_at AS session_created_at
FROM account.user_session us
WHERE us.token = $1
GROUP BY us.id;

-- name: GetSessionsByUserId :many
SELECT 
    id,
    token,
    user_id,
    role,
    user_agent,
    ip_address,
    revoked,
    revoked_at,
    expires_at,
    created_at,
    COUNT(*) OVER() as total
FROM account.user_session 
WHERE user_id = $1 AND revoked = FALSE AND expires_at > CURRENT_TIMESTAMP
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: RevokeSession :one
UPDATE account.user_session
SET revoked = TRUE, revoked_at = CURRENT_TIMESTAMP
WHERE id = $1 
  AND user_id = $2
RETURNING id;

-- name: RevokeAllSessions :many
UPDATE account.user_session
SET revoked = TRUE, revoked_at = CURRENT_TIMESTAMP
WHERE user_id = $1 
  AND expires_at > CURRENT_TIMESTAMP
  AND revoked = FALSE
  AND  token != $2
RETURNING id;

-- name: RevokeSessionByToken :exec
UPDATE account.user_session
SET revoked = TRUE, revoked_at = CURRENT_TIMESTAMP
WHERE token = $1 
  AND revoked = FALSE
  AND expires_at > CURRENT_TIMESTAMP;