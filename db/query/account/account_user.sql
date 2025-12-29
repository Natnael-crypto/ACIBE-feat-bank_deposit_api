-- name: CreateUser :one
INSERT INTO account.user (
  fname, lname, email, user_password, account_status, email_verified, role, verified_at, profile_picture_url, google_id, microsoft_id
) VALUES (
     $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
)
RETURNING id, fname, lname, email, user_password, account_status, email_verified, role, profile_picture_url, verified_at, google_id, microsoft_id, created_at, updated_at;
-- name: GetUserByEmail :one
SELECT
    id, fname, lname, email, user_password, account_status, email_verified, role, profile_picture_url, verified_at, google_id, microsoft_id, created_at, updated_at
FROM
    account.user
WHERE
    email = $1;

-- name: GetUserById :one
SELECT
    id, fname, lname, email, user_password, account_status, email_verified, role, profile_picture_url, verified_at, google_id, microsoft_id, created_at, updated_at
FROM
    account.user
WHERE
    id = $1;

-- name: GetUserByGoogleID :one
SELECT
    id, fname, lname, email, user_password, account_status, email_verified, role, profile_picture_url, verified_at, google_id, microsoft_id, created_at, updated_at
FROM
    account.user
WHERE
    google_id = $1;

-- name: GetUserByMicrosoftID :one
SELECT
    id, fname, lname, email, user_password, account_status, email_verified, role, profile_picture_url, verified_at, google_id, microsoft_id, created_at, updated_at
FROM
    account.user
WHERE
    microsoft_id = $1;

-- name: UpdateUser :one
UPDATE account.user
SET
    fname          = COALESCE(sqlc.narg('fname'), fname),
    lname          = COALESCE(sqlc.narg('lname'), lname),
    email           = COALESCE(sqlc.narg('email'), email),
    account_status = COALESCE(sqlc.narg('account_status'), account_status),
    email_verified = COALESCE(sqlc.narg('email_verified'), email_verified),
    role           = COALESCE(sqlc.narg('role'), role),
    profile_picture_url = COALESCE(sqlc.narg('profile_picture_url'), profile_picture_url),
    verified_at    = COALESCE(sqlc.narg('verified_at'), verified_at),
    google_id      = COALESCE(sqlc.narg('google_id'), google_id),
    microsoft_id   = COALESCE(sqlc.narg('microsoft_id'), microsoft_id),
    updated_at     = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, fname, lname, email, user_password, account_status, email_verified, role, profile_picture_url, verified_at, google_id, microsoft_id, created_at, updated_at;

-- name: UpdateOAuthIDs :one
UPDATE account.user
SET
    google_id    = COALESCE(sqlc.narg('google_id'), google_id),
    microsoft_id = COALESCE(sqlc.narg('microsoft_id'), microsoft_id),
    updated_at   = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, fname, lname, email, user_password, account_status, email_verified, role, profile_picture_url, verified_at, google_id, microsoft_id, created_at, updated_at;

-- name: UpdatePassword :one
UPDATE account.user
SET user_password = $2, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, fname, lname, email, user_password, account_status, email_verified, role, profile_picture_url, verified_at, google_id, microsoft_id, created_at, updated_at;


-- name: GetAllUsers :many
SELECT
    id, fname, lname,
     email, user_password,
      account_status, email_verified,
       role, profile_picture_url, verified_at, google_id, microsoft_id, created_at, 
       COUNT(*) OVER() AS count, updated_at
FROM
    account.user
ORDER BY 
    CASE WHEN $1 = 'asc' THEN created_at END ASC,
    CASE WHEN $1 = 'desc' THEN created_at END DESC
LIMIT $2 OFFSET $3;

-- name: SearchUsers :many
SELECT
    id, fname, lname, email, user_password, account_status, email_verified, role, profile_picture_url, verified_at, google_id, microsoft_id, created_at, updated_at,
    COUNT(*) OVER() AS count
FROM
    account.user
WHERE
    fname ILIKE '%' || $1 || '%' OR
    lname ILIKE '%' || $1 || '%' OR
    email ILIKE '%' || $1 || '%' OR
    role ILIKE '%' || $1 || '%'
ORDER BY fname ASC, lname ASC
LIMIT $2 OFFSET $3;