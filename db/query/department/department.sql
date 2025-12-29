-- name: CreateDepartment :one
INSERT INTO department.department (
    name, description, resources
) VALUES (
    $1, $2, $3
)
RETURNING id, name, description, resources, created_at, updated_at;

-- name: GetDepartmentById :one
SELECT
    id, name, description, resources, created_at, updated_at
FROM
    department.department
WHERE
    id = $1;

-- name: GetAllDepartments :many
SELECT
    id, name, description, resources, created_at, updated_at,
    COUNT(*) OVER() AS count
FROM
    department.department
WHERE
    ($3::text IS NULL OR $3::text = '' OR
     name ILIKE '%' || $3::text || '%' OR
     description ILIKE '%' || $3::text || '%')
ORDER BY 
    CASE WHEN $3::text IS NOT NULL AND $3::text != '' THEN name END ASC,
    CASE WHEN $3::text IS NULL OR $3::text = '' THEN created_at END DESC
LIMIT $1 OFFSET $2;

-- name: UpdateDepartment :one
UPDATE department.department
SET
    name = COALESCE(sqlc.narg('name'), name),
    description = COALESCE(sqlc.narg('description'), description),
    resources = COALESCE(sqlc.narg('resources'), resources),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, name, description, resources, created_at, updated_at;

-- name: DeleteDepartment :exec
DELETE FROM department.department
WHERE id = $1;

-- name: AddUserToDepartment :exec
INSERT INTO department.user_department (user_id, department_id)
VALUES ($1, $2)
ON CONFLICT (user_id, department_id) DO NOTHING;

-- name: RemoveUserFromDepartment :exec
DELETE FROM department.user_department
WHERE user_id = $1 AND department_id = $2;

-- name: GetUserDepartments :many
SELECT
    d.id, d.name, d.description, d.resources, d.created_at, d.updated_at
FROM
    department.department d
INNER JOIN department.user_department ud ON d.id = ud.department_id
WHERE
    ud.user_id = $1;

-- name: GetDepartmentUsers :many
SELECT
    u.id, u.fname, u.lname, u.email, u.account_status, u.email_verified, u.role, u.profile_picture_url, u.verified_at, u.google_id, u.microsoft_id, u.created_at, u.updated_at
FROM
    account.user u
INNER JOIN department.user_department ud ON u.id = ud.user_id
WHERE
    ud.department_id = $1;

-- name: ReplaceUserDepartments :exec
DELETE FROM department.user_department
WHERE user_id = $1;

