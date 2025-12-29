-- name: StoreToken :exec
INSERT INTO account.token (
    token, user_id, email, purpose, used, revoked, expires_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
);
-- name: GetToken :one
SELECT
    token, user_id, email, purpose, used, revoked, expires_at, created_at
FROM
    account.token
WHERE
    token = $1;
-- -- name: MarkLastTokenAsRevoked :exec
-- UPDATE account.token
-- SET revoked = TRUE
-- WHERE token = (
--     SELECT token
--     FROM account.token
--     WHERE user_id = $1 AND revoked = FALSE
--     ORDER BY created_at DESC
--     LIMIT 1
-- );
-- name: MarkTokenAsUsed :exec
UPDATE account.token
SET used = TRUE
WHERE token = $1;
-- name: MarkTokenAsRevoked :exec
UPDATE account.token
SET revoked = TRUE
WHERE token = $1;
-- name: MarkUserTokensAsRevoked :exec
UPDATE account.token
SET revoked = TRUE
WHERE user_id = $1 AND revoked = FALSE;

-- name: GetTokenByPurpose :one
SELECT
    token, user_id, email, purpose, used, revoked, expires_at, created_at
FROM
    account.token
WHERE
    token = $1 AND purpose = $2;

-- name: DeleteUserTokensByPurpose :exec
DELETE FROM account.token
WHERE user_id = $1 AND purpose = $2;