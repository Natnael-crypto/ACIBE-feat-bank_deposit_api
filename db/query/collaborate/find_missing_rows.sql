-- Find claim/charge IDs that exist in new_claim but are missing from the main query results
-- This will help identify which 500 rows are missing

-- First, get all claim/charge IDs from new_claim
WITH all_claims AS (
    SELECT DISTINCT claim_id, charge_id
    FROM collaboratemd.new_claim
),
query_results AS (
    SELECT DISTINCT ON (nc.claim_id, nc.charge_id)
        nc.claim_id,
        nc.charge_id
    FROM collaboratemd.new_claim nc
    LEFT JOIN collaboratemd.out_patient_rate opr 
        ON nc.cpt_code = ANY(opr.codes)
        AND (opr.modifiers = '{}' OR opr.modifiers <@ nc.modifiers)
    LEFT JOIN LATERAL (
        SELECT period
        FROM jsonb_array_elements(opr.rate_periods) AS period
        WHERE 
            NULLIF(period->>'start_date', '') IS NOT NULL
            AND nc.claim_from_date >= CASE 
                WHEN (NULLIF(period->>'start_date', '') ~ '^[0-9]{4}-[0-9]{1,2}-[0-9]{1,2}$') THEN
                    (NULLIF(period->>'start_date', ''))::DATE
                WHEN (NULLIF(period->>'start_date', '') ~ '/[0-9]{4}$') THEN
                    TO_DATE(NULLIF(period->>'start_date', ''), 'MM/DD/YYYY')
                ELSE
                    TO_DATE(NULLIF(period->>'start_date', ''), 'MM/DD/YY')
            END
            AND (
                NULLIF(period->>'end_date', '') IS NULL 
                OR NULLIF(period->>'end_date', '') = ''
                OR nc.claim_from_date <= CASE 
                    WHEN (NULLIF(period->>'end_date', '') ~ '^[0-9]{4}-[0-9]{1,2}-[0-9]{1,2}$') THEN
                        (NULLIF(period->>'end_date', ''))::DATE
                    WHEN (NULLIF(period->>'end_date', '') ~ '/[0-9]{4}$') THEN
                        TO_DATE(NULLIF(period->>'end_date', ''), 'MM/DD/YYYY')
                    ELSE
                        TO_DATE(NULLIF(period->>'end_date', ''), 'MM/DD/YY')
                END
            )
        ORDER BY 
            CASE WHEN NULLIF(period->>'end_date', '') IS NOT NULL AND NULLIF(period->>'end_date', '') != '' THEN 0 ELSE 1 END,
            CASE 
                WHEN (NULLIF(period->>'start_date', '') ~ '^[0-9]{4}-[0-9]{1,2}-[0-9]{1,2}$') THEN
                    (NULLIF(period->>'start_date', ''))::DATE
                WHEN (NULLIF(period->>'start_date', '') ~ '/[0-9]{4}$') THEN
                    TO_DATE(NULLIF(period->>'start_date', ''), 'MM/DD/YYYY')
                ELSE
                    TO_DATE(NULLIF(period->>'start_date', ''), 'MM/DD/YY')
            END DESC
        LIMIT 1
    ) AS rp ON TRUE
    WHERE 
        TRIM(nc.revenue_code) IN ('0126', '0128', '0136', '0138', '126', '128', '136', '138')
        OR rp.period IS NOT NULL
    ORDER BY 
        nc.claim_id,
        nc.charge_id,
        CASE WHEN opr.modifiers = '{}' THEN 1 ELSE 0 END,
        array_length(opr.modifiers, 1) DESC NULLS LAST,
        opr.id
)
SELECT DISTINCT
    nc.cpt_code,
    COUNT(*) as claim_count
FROM all_claims ac
LEFT JOIN query_results qr ON ac.claim_id = qr.claim_id AND ac.charge_id = qr.charge_id
INNER JOIN collaboratemd.new_claim nc ON ac.claim_id = nc.claim_id AND ac.charge_id = nc.charge_id
WHERE qr.claim_id IS NULL
GROUP BY nc.cpt_code
ORDER BY claim_count DESC, nc.cpt_code;

