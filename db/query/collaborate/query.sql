
SELECT DISTINCT ON (nc.claim_id, nc.charge_id)
    opr.id AS rate_id,
    nc.revenue_code,
    nc.claim_id,
    nc.cpt_code,
    nc.claim_from_date,
    opr.description AS rate_description,
    nc.amount,
    CASE 
        WHEN TRIM(nc.revenue_code) IN ('0126', '0128', '0136', '0138', '126', '128', '136', '138') THEN
            500.00
        WHEN nc.value_amount_1 = 1080 AND rp.period IS NOT NULL THEN 
            (rp.period->>'off_site_rate')::DECIMAL(18, 2) * COALESCE(NULLIF(nc.units, '')::INTEGER, 1)
        WHEN rp.period IS NOT NULL THEN 
            (rp.period->>'on_site_rate')::DECIMAL(18, 2) * COALESCE(NULLIF(nc.units, '')::INTEGER, 1)
        ELSE 
            NULL
    END AS expected_amount
FROM collaboratemd.new_claim nc
LEFT JOIN collaboratemd.out_patient_rate opr 
    ON nc.cpt_code = ANY(opr.codes)
    AND (opr.modifiers = '{}' OR opr.modifiers <@ nc.modifiers)
LEFT JOIN LATERAL (
    SELECT period
    FROM jsonb_array_elements(opr.rate_periods) AS period
    WHERE 
        NULLIF(period->>'start_date', '') IS NOT NULL
        -- Parse start_date: support both ISO format (YYYY-MM-DD) and US format (M/D/YY or M/D/YYYY)
        AND nc.claim_from_date >= CASE 
            WHEN (NULLIF(period->>'start_date', '') ~ '^[0-9]{4}-[0-9]{1,2}-[0-9]{1,2}$') THEN
                (NULLIF(period->>'start_date', ''))::DATE
            WHEN (NULLIF(period->>'start_date', '') ~ '/[0-9]{4}$') THEN
                TO_DATE(NULLIF(period->>'start_date', ''), 'MM/DD/YYYY')
            ELSE
                TO_DATE(NULLIF(period->>'start_date', ''), 'MM/DD/YY')
        END
        -- Parse end_date and ensure claim date is within the period (start_date <= claim_date <= end_date)
        -- Handle empty end_date as "no end date" (matches all future dates)
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
    LIMIT 1  -- Only get the first matching rate period to avoid duplicates
) AS rp ON TRUE
WHERE 
    -- For revenue codes 0126, 0128, 0136, 0138, don't require rate match (they don't have CPT codes)
    TRIM(nc.revenue_code) IN ('0126', '0128', '0136', '0138', '126', '128', '136', '138')
    -- For other rows, require a matching rate period
    OR rp.period IS NOT NULL
ORDER BY 
    nc.claim_id,
    nc.charge_id,
    -- Prefer rates with specific modifiers over empty modifiers (more specific match)
    CASE WHEN opr.modifiers = '{}' THEN 1 ELSE 0 END,
    -- Prefer rates with more modifiers (more specific)
    array_length(opr.modifiers, 1) DESC NULLS LAST,
    opr.id;
--------------------------
--  CAPITAL DISTRICT PHYSICIANS HEALTH PLAN-CDPHP medicaid, detox rehab is zero
-- 
SELECT distinct
    claim_id,
    
    primary_payer->>'Name' as payer_name,
    primary_payer->>'Id' as payer_id,
  revenue_code,
    amount/units::int,
  amount,
  units,
  claim_from_date
   -- expected_amount
FROM collaboratemd.new_claim 
WHERE  primary_payer->>'Name' ='NEW YORK MEDICAID-DO NOT USE'
  -- and expected_amount=5555
  and(revenue_code='0128' or revenue_code='0126')
  order by payer_name,payer_id
LIMIT 1000;
