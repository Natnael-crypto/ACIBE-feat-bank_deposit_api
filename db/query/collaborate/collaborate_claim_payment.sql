-- name: CreateClaim :exec
INSERT INTO collaboratemd.claim
    (
        claim_id, --1
        claim_frequency, --2
        original_claim_number, --3
        total_charge, --4
        claim_submission_date, --5
        service_date_from, --6
        service_date_to, --7
        patient_id, --8
        patient_name, --9
        service_lines, --10
        type_of_service, --11
        primary_insurance_name, --12
        primary_insurance_id, --13
        secondary_insurance_name, --14
        secondary_insurance_id, --15
        raw_text --16
    )
VALUES 
    (
    unnest($1::text[]), -- claim_id
    unnest($2::text[]), -- claim_frequency
    unnest($3::text[]), -- original_claim_number
    unnest($4::numeric[]), -- total_charge
    unnest($5::date[]), -- claim_submission_date
    unnest($6::date[]), -- service_date_from
    unnest($7::date[]), -- service_date_to
    unnest($8::text[]), -- patient_id
    unnest($9::text[]), -- patient_name
    unnest($10::jsonb[]), -- service_lines
    unnest($11::text[]), -- type_of_service
    unnest($12::text[]), -- primary_insurance_name
    unnest($13::text[]), -- primary_insurance_id
    unnest($14::text[]), -- secondary_insurance_name
    unnest($15::text[]), -- secondary_insurance_id
    unnest($16::text[]) -- raw_text
    );

-- name: CreatePayment :exec
INSERT INTO collaboratemd.payment (
    claim_id,
    patient_id,
    total_claim_charge, 
    total_paid_amount, 
    claim_status, 
    insurance_name, 
    insurance_id, 
    raw_text, 
    service_lines, 
    payment_date, 
    posted_date, 
    charge_date,
    bpr_date,
    trn,
    claim_level_adjustment
    )
VALUES 
    (
    unnest($1::text[]), -- claim_id
    unnest($2::text[]), -- patient_id
    unnest($3::numeric[]), -- total_claim_charge
    unnest($4::numeric[]), -- total_paid_amount
    unnest($5::text[]), -- claim_status
    unnest($6::text[]), -- insurance_name
    unnest($7::text[]), -- insurance_id
    unnest($8::text[]), -- raw_text
    unnest($9::jsonb[]), -- service_lines
    unnest($10::date[]), -- payment_date
    unnest($11::date[]), -- posted_date
    unnest($12::date[]), -- charge_date
    unnest($13::date[]), -- bpr_date
    unnest($14::text[]), -- trn
    unnest($15::jsonb[]) -- claim_level_adjustment
    ) ON CONFLICT (claim_id, trn) DO NOTHING;





-- name: CreateNewPayment :exec
INSERT INTO collaboratemd.new_payment (
    payment_received_date,
    deposit_date,
    payment_entered_date,
    charge_from_date,
    claim_id,
    charge_id,
    cpt_code,
    modifiers,
    insurance_paid_a,
    insurance_adjustment,
    charge_amount,
    credit_id,
    payment_check,
    remark,
    unpaid_reason,
    payer_id,
    payment_name,
    patient_id,
    patient_name,
    revenue_code,
    is_refund,
    is_provider_adjustment,
    is_era_adjustment
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8,
    $9,
    $10,
    $11,
    $12,
    $13,
    $14,
    $15,
    $16,
    $17,
    $18,
    $19,
    $20,
    $21,
    $22,
    $23
) 
ON CONFLICT (payment_entered_date, claim_id,credit_id, charge_id, cpt_code, payment_check,is_refund,is_provider_adjustment,is_era_adjustment) DO NOTHING;




-- name: CreateNewClaim :exec
INSERT INTO collaboratemd.new_claim (
    claim_id,
    charge_id,
    total_claim,
    claim_entered_date,
    claim_from_date,
    claim_to_date,
    claim_type,
    primary_payer,
    secondary_payer,
    tertiary_payer,
    patient_id,
    patient_name,
    cpt_code,
    modifiers,
    amount,
    units,
    revenue_code,
    facility_pos_code,
    charge_pos_code,
    value_code_1,
    value_amount_1,
    claim_status
) VALUES (
    $1::text[],
    $2::text[],
    $3::numeric[],
    $4::date[],
    $5::date[],
    $6::date[],
    $7::text[],
    $8::text[],
    $9::text[],
    $10::text[],
    $11::text[],
    $12::text[],
    $13::text[],
    $14::text[],
    $15::text[],
    $16::text[],
    $17::text[],
    $18::text[],
    $19::text[],
    $20::text[],
    $21::text[],
    $22::text[]
    ) ON CONFLICT (claim_id,charge_id) DO NOTHING;
   


-- name: CreateNewClaims :exec
WITH data AS (
    SELECT 
        unnest($1::text[]) AS claim_id,
        unnest($2::text[]) AS charge_id,
        unnest($3::numeric[]) AS total_claim,
        unnest($4::date[]) AS claim_entered_date,
        unnest($5::date[]) AS claim_from_date,
        unnest($6::date[]) AS claim_to_date,
        unnest($7::text[]) AS claim_type,
        unnest($8::jsonb[]) AS primary_payer,
        unnest($9::jsonb[]) AS secondary_payer,
        unnest($10::jsonb[]) AS tertiary_payer,
        unnest($11::text[]) AS patient_id,
        unnest($12::text[]) AS patient_name,
        unnest($13::text[]) AS cpt_code,
        unnest($14::jsonb[]) AS modifiers_jsonb,
        unnest($15::numeric[]) AS amount,
        unnest($16::text[]) AS units,
        unnest($17::text[]) AS revenue_code,
        unnest($18::text[]) AS facility_pos_code,
        unnest($19::text[]) AS charge_pos_code,
        unnest($20::text[]) AS value_code_1,
        unnest($21::integer[]) AS value_amount_1,
        unnest($22::text[]) AS claim_status,
        unnest($23::numeric[]) AS expected_amount
)
INSERT INTO collaboratemd.new_claim (
    claim_id,
    charge_id,
    total_claim,
    claim_entered_date,
    claim_from_date,
    claim_to_date,
    claim_type,
    primary_payer,
    secondary_payer,
    tertiary_payer,
    patient_id,
    patient_name,
    cpt_code,
    modifiers,
    amount,
    units,
    revenue_code,
    facility_pos_code,
    charge_pos_code,
    value_code_1,
    value_amount_1,
    claim_status,
    expected_amount
)
SELECT 
    claim_id,
    charge_id,
    total_claim,
    claim_entered_date,
    claim_from_date,
    claim_to_date,
    claim_type,
    primary_payer,
    secondary_payer,
    tertiary_payer,
    patient_id,
    patient_name,
    cpt_code,
    COALESCE(
        (SELECT array_agg(value::text) 
         FROM jsonb_array_elements_text(modifiers_jsonb) AS value),
        ARRAY[]::text[]
    ) AS modifiers,
    amount,
    units,
    revenue_code,
    facility_pos_code,
    charge_pos_code,
    value_code_1,
    value_amount_1,
    claim_status,
    expected_amount
FROM data
ON CONFLICT (claim_id,charge_id) DO NOTHING;
   
