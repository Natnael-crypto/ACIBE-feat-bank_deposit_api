
-- name: CreateOutPatientRate :one
INSERT INTO collaboratemd.out_patient_rate (codes, modifiers, quantifiers, type, description, rate_periods, plus_individual_codes, note)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id;


-- name: GetOutPatientRate :one
SELECT * FROM collaboratemd.out_patient_rate
WHERE $1::text = ANY(codes) -- code is in the codes array
AND (
    -- Case 1: When modifier is provided (not empty/null)
    ($2::text IS NOT NULL AND $2::text != ''
    AND  $2::text = ANY(modifiers))
    OR
    -- Case 2: When no modifier is provided, look for code-only rates
    (($2::text IS NULL OR $2::text = '')
     AND modifiers = '{}'::text[])
)
LIMIT 1;


-- name: GetAllOutPatientRates :many
SELECT * FROM collaboratemd.out_patient_rate;