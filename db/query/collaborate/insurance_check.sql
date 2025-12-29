-- name: CreateInsuranceChecks :exec
INSERT INTO collaboratemd.insurance_check (
    id,
    check_payer,
    check_date,
    deposit_date,
    check_number,
    check_amount,
    check_applied_amount,
    provider_adjustment_amount,
    check_remaining_amount,
    check_committed
) VALUES (
    unnest($1::text[]), -- id
    unnest($2::text[]), -- check_payer
    unnest($3::timestamptz[]), -- check_date
    unnest($4::timestamptz[]), -- deposit_date
    unnest($5::text[]), -- check_number
    unnest($6::numeric[]), -- check_amount
    unnest($7::numeric[]), -- check_applied_amount
    unnest($8::numeric[]), -- provider_adjustment_amount
    unnest($9::numeric[]), -- check_remaining_amount
    unnest($10::boolean[]) -- check_committed
) ON CONFLICT (id) DO NOTHING;

