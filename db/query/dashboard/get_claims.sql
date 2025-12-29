-- name: GetClaims :many
-- Get paginated unique claims by claim_id only (matching dashboard API logic)
-- Returns one row per unique claim_id with aggregated values from all charge lines
-- OPTIMIZED: Filter -> Aggregates -> Combined Filters -> Paginate unique claim_ids -> Detailed Aggregates
--
-- $1::int = year (0 = all years)
-- $2::int = month (0 = all months)
-- $3::text = category (totalBilled = claims with payments, outstanding = claims without payments, totalClaims/totalExpected = all claims)
-- $4::boolean = exclude_peer (exclude claims with CPT code H0038)
-- $5::text = search (optional search term for claim_id)
-- $6::boolean = peer_only (include only claims with CPT code H0038)
-- $7::text = claim_type filter (empty = all types, inpatient, outpatient, peer)
-- $8::int = limit
-- $9::int = offset
-- $10::text = claims_expected_filter ('all', 'claimsGreater', 'expectedGreater')
-- $11::text = payment_status_filter ('all', 'overpaid', 'underpaid', 'fullyPaid')
WITH
    date_range AS (
        SELECT
            CASE
                WHEN $1::int = 0 THEN NULL
                ELSE DATE_TRUNC(
                    'year',
                    MAKE_DATE($1::int, 1, 1)
                )
            END as year_start,
            CASE
                WHEN $1::int = 0 THEN NULL
                ELSE DATE_TRUNC(
                    'year',
                    MAKE_DATE($1::int, 1, 1)
                ) + INTERVAL '1 year'
            END as year_end,
            CASE
                WHEN $2::int = 0 THEN NULL
                ELSE $2::int
            END as month_filter
    ),
    filtered_claims AS (
        -- Get all claim charge lines with filters applied early - optimized for index usage
        SELECT
            nc.claim_id,
            nc.charge_id,
            nc.cpt_code,
            nc.modifiers,
            nc.revenue_code,
            nc.claim_type,
            nc.claim_entered_date,
            nc.claim_from_date,
            nc.claim_to_date,
            nc.patient_id,
            nc.patient_name,
            nc.primary_payer,
            nc.secondary_payer,
            nc.tertiary_payer,
            nc.units,
            nc.facility_pos_code,
            nc.charge_pos_code,
            nc.expected_amount,
            nc.amount,
            EXTRACT(
                MONTH
                FROM nc.claim_from_date
            )::numeric as month_number,
            TO_CHAR(nc.claim_from_date, 'Month') as month_name,
            CASE
                WHEN nc.cpt_code = 'H0038' THEN 'peer'
                WHEN (
                    nc.revenue_code IN ('0126', '0128')
                    OR UPPER(nc.cpt_code) IN (
                        'H2019',
                        'H0017',
                        'H0019',
                        '99235',
                        'H0010',
                        'H2036'
                    )
                )
                AND nc.cpt_code != 'H0038' THEN 'inpatient'
                ELSE 'outpatient'
            END as determined_claim_type
        FROM collaboratemd.new_claim nc
            CROSS JOIN date_range dr
        WHERE (
                dr.year_start IS NULL
                OR (
                    nc.claim_from_date >= dr.year_start
                    AND nc.claim_from_date < dr.year_end
                )
            )
            AND (
                dr.month_filter IS NULL
                OR EXTRACT(
                    MONTH
                    FROM nc.claim_from_date
                ) = dr.month_filter
            )
            AND (
                $5::text IS NULL
                OR $5::text = ''
                OR nc.claim_id::text ILIKE '%' || $5::text || '%'
            )
            AND (
                $7::text IS NULL
                OR $7::text = ''
                OR CASE
                    WHEN nc.cpt_code = 'H0038' THEN LOWER($7::text) = 'peer'
                    WHEN (
                        nc.revenue_code IN ('0126', '0128')
                        OR UPPER(nc.cpt_code) IN (
                            'H2019',
                            'H0017',
                            'H0019',
                            '99235',
                            'H0010',
                            'H2036'
                        )
                    )
                    AND nc.cpt_code != 'H0038' THEN LOWER($7::text) = 'inpatient'
                    ELSE LOWER($7::text) = 'outpatient'
                END
            )
            AND NOT (
                $4::boolean = true
                AND nc.cpt_code = 'H0038'
            )
            AND (
                (
                    $6::boolean IS NULL
                    OR $6::boolean = false
                )
                OR (
                    $6::boolean = true
                    AND nc.cpt_code = 'H0038'
                )
            )
    ),
    unique_claim_ids AS (
        -- Same logic as before to get unique candidate claim_ids
        SELECT DISTINCT
            claim_id
        FROM (
                SELECT DISTINCT
                    nc.claim_id
                FROM collaboratemd.new_claim nc
                    CROSS JOIN date_range dr_all
                WHERE
                    $3::text = 'outstanding'
                    AND (
                        dr_all.year_start IS NULL
                        OR (
                            nc.claim_from_date >= dr_all.year_start
                            AND nc.claim_from_date < dr_all.year_end
                        )
                    )
                    AND (
                        dr_all.month_filter IS NULL
                        OR EXTRACT(
                            MONTH
                            FROM nc.claim_from_date
                        ) = dr_all.month_filter
                    )
                    AND (
                        $5::text IS NULL
                        OR $5::text = ''
                        OR nc.claim_id::text ILIKE '%' || $5::text || '%'
                    )
                    AND NOT (
                        $4::boolean = true
                        AND nc.cpt_code = 'H0038'
                    )
                    AND (
                        (
                            $6::boolean IS NULL
                            OR $6::boolean = false
                        )
                        OR (
                            $6::boolean = true
                            AND nc.cpt_code = 'H0038'
                        )
                    )
                    AND EXISTS (
                        SELECT 1
                        FROM collaboratemd.new_claim nc2
                        WHERE
                            nc2.claim_id = nc.claim_id
                            AND (
                                dr_all.year_start IS NULL
                                OR (
                                    nc2.claim_from_date >= dr_all.year_start
                                    AND nc2.claim_from_date < dr_all.year_end
                                )
                            )
                            AND (
                                dr_all.month_filter IS NULL
                                OR EXTRACT(
                                    MONTH
                                    FROM nc2.claim_from_date
                                ) = dr_all.month_filter
                            )
                        GROUP BY
                            nc2.claim_id
                        HAVING (
                                COALESCE(SUM(nc2.expected_amount), 0) - COALESCE(
                                    (
                                        SELECT SUM(p_all.insurance_paid_a)
                                        FROM collaboratemd.new_payment p_all
                                        WHERE
                                            p_all.claim_id = nc2.claim_id
                                    ), 0
                                )
                            ) > 0
                        LIMIT 1
                    )
                UNION
                SELECT DISTINCT
                    fc.claim_id
                FROM filtered_claims fc
                WHERE (
                        $3::text IS NULL
                        OR $3::text = ''
                        OR $3::text NOT IN ('totalBilled', 'outstanding')
                    )
                    OR (
                        $3::text = 'totalBilled'
                        AND EXISTS (
                            SELECT 1
                            FROM collaboratemd.new_payment p
                            WHERE
                                p.claim_id = fc.claim_id
                            LIMIT 1
                        )
                    )
            ) all_claim_ids
        WHERE (
                LOWER($7::text) != 'outpatient'
                OR NOT EXISTS (
                    SELECT 1
                    FROM collaboratemd.new_claim nc_check
                    WHERE
                        nc_check.claim_id = all_claim_ids.claim_id
                        AND (
                            (
                                nc_check.revenue_code IN ('0126', '0128')
                                OR UPPER(nc_check.cpt_code) IN (
                                    'H2019',
                                    'H0017',
                                    'H0019',
                                    '99235',
                                    'H0010',
                                    'H2036'
                                )
                            )
                            OR nc_check.cpt_code = 'H0038'
                        )
                )
            )
            AND (
                LOWER($7::text) != 'peer'
                OR NOT EXISTS (
                    SELECT 1
                    FROM collaboratemd.new_claim nc_check
                    WHERE
                        nc_check.claim_id = all_claim_ids.claim_id
                        AND (
                            (
                                nc_check.revenue_code IN ('0126', '0128')
                                OR UPPER(nc_check.cpt_code) IN (
                                    'H2019',
                                    'H0017',
                                    'H0019',
                                    '99235',
                                    'H0010',
                                    'H2036'
                                )
                            )
                            AND nc_check.cpt_code != 'H0038'
                        )
                )
            )
    ),
    aggregated_claims_for_filtering AS (
        -- Aggregate amounts for ALL unique claim IDs to apply combined filters (Claims vs Expected, Payment Status)
        -- This ensures we filter BEFORE pagination
        SELECT
            nc.claim_id,
            SUM(COALESCE(nc.amount, 0))::numeric as total_claims_sum,
            SUM(
                COALESCE(nc.expected_amount, 0)
            )::numeric as total_expected_sum,
            SUM(
                COALESCE(
                    NULLIF(TRIM(nc.units), '')::numeric,
                    0
                )
            )::numeric as total_units_sum,
            COUNT(
                DISTINCT COALESCE(nc.charge_id, '')
            )::int as total_charges,
            COALESCE(
                (
                    SELECT SUM(p.insurance_paid_a)
                    FROM collaboratemd.new_payment p
                    WHERE
                        p.claim_id = nc.claim_id
                ),
                0
            )::numeric as total_payment_sum
        FROM collaboratemd.new_claim nc
            CROSS JOIN date_range dr
        WHERE
            nc.claim_id IN (
                SELECT claim_id
                FROM unique_claim_ids
            )
            AND (
                dr.year_start IS NULL
                OR (
                    nc.claim_from_date >= dr.year_start
                    AND nc.claim_from_date < dr.year_end
                )
            )
            AND (
                dr.month_filter IS NULL
                OR EXTRACT(
                    MONTH
                    FROM nc.claim_from_date
                ) = dr.month_filter
            )
        GROUP BY
            nc.claim_id
    ),
    filtered_unique_claim_ids AS (
        -- Apply the combined filters ($10, $11) on the aggregated data
        SELECT acf.claim_id, acf.total_claims_sum, acf.total_expected_sum, acf.total_units_sum, acf.total_charges, acf.total_payment_sum
        FROM
            aggregated_claims_for_filtering acf
        WHERE
            -- Claims vs Expected filter ($10)
            (
                $10::text IS NULL
                OR $10::text = ''
                OR $10::text = 'all'
                OR (
                    $10::text = 'claimsGreater'
                    AND acf.total_claims_sum > acf.total_expected_sum
                )
                OR (
                    $10::text = 'expectedGreater'
                    AND acf.total_expected_sum > acf.total_claims_sum
                )
            )
            -- Payment status filter ($11)
            AND (
                $11::text IS NULL
                OR $11::text = ''
                OR $11::text = 'all'
                OR (
                    $11::text = 'overpaid'
                    AND (
                        (
                            acf.total_expected_sum - acf.total_payment_sum
                        ) < 0
                    )
                )
                OR (
                    $11::text = 'underpaid'
                    AND (
                        (
                            acf.total_expected_sum - acf.total_payment_sum
                        ) > 0
                    )
                )
                OR (
                    $11::text = 'fullyPaid'
                    AND ABS(
                        acf.total_expected_sum - acf.total_payment_sum
                    ) = 0
                )
            )
    ),
    paginated_claim_ids AS (
        -- Paginate the already filtered unique claim_ids
        SELECT fuci.claim_id
        FROM
            filtered_unique_claim_ids fuci
            INNER JOIN filtered_claims fc ON fuci.claim_id = fc.claim_id
        GROUP BY
            fuci.claim_id
        ORDER BY MAX(fc.claim_from_date) ASC, fuci.claim_id
        LIMIT $8::int
        OFFSET
            $9::int
    ),
    all_charges_aggregations AS (
        -- Detailed aggregates only for the paginated subset (using pre-calculated results where possible)
        SELECT
            fuci.claim_id,
            fuci.total_claims_sum as total_claims_sum_all,
            fuci.total_expected_sum as total_expected_sum_all,
            fuci.total_units_sum as total_units_sum_all,
            fuci.total_charges as total_charges_all,
            fuci.total_payment_sum as total_payment_sum_all
        FROM filtered_unique_claim_ids fuci
        WHERE
            fuci.claim_id IN (
                SELECT claim_id
                FROM paginated_claim_ids
            )
    ),
    claim_aggregations AS (
        SELECT
            pci.claim_id,
            MIN(
                CASE
                    WHEN fc.determined_claim_type = 'inpatient' THEN 1
                    WHEN fc.determined_claim_type = 'peer' THEN 2
                    ELSE 3
                END
            ) as priority,
            (
                ARRAY_AGG(
                    fc.claim_from_date
                    ORDER BY fc.claim_from_date DESC
                )
            ) [1] as claim_from_date,
            (
                ARRAY_AGG(
                    fc.claim_to_date
                    ORDER BY fc.claim_from_date DESC
                )
            ) [1] as claim_to_date,
            (
                ARRAY_AGG(
                    fc.claim_entered_date
                    ORDER BY fc.claim_from_date DESC
                )
            ) [1] as claim_entered_date,
            (
                ARRAY_AGG(
                    fc.patient_id
                    ORDER BY fc.claim_from_date DESC
                )
            ) [1] as patient_id,
            (
                ARRAY_AGG(
                    fc.patient_name
                    ORDER BY fc.claim_from_date DESC
                )
            ) [1] as patient_name,
            (
                ARRAY_AGG(
                    fc.claim_type
                    ORDER BY fc.claim_from_date DESC
                )
            ) [1] as type_of_service,
            (
                ARRAY_AGG(
                    fc.primary_payer
                    ORDER BY fc.claim_from_date DESC
                )
            ) [1] as primary_payer,
            (
                ARRAY_AGG(
                    fc.secondary_payer
                    ORDER BY fc.claim_from_date DESC
                )
            ) [1] as secondary_payer,
            (
                ARRAY_AGG(
                    fc.tertiary_payer
                    ORDER BY fc.claim_from_date DESC
                )
            ) [1] as tertiary_payer,
            (
                ARRAY_AGG(
                    fc.month_number
                    ORDER BY fc.claim_from_date DESC
                )
            ) [1] as month_number,
            (
                ARRAY_AGG(
                    fc.month_name
                    ORDER BY fc.claim_from_date DESC
                )
            ) [1] as month_name,
            (
                ARRAY_AGG(
                    fc.charge_id
                    ORDER BY fc.claim_from_date DESC
                ) FILTER (
                    WHERE
                        fc.charge_id IS NOT NULL
                        AND fc.charge_id != ''
                )
            ) [1] as first_charge_id,
            COALESCE(
                ARRAY_AGG(DISTINCT fc.cpt_code) FILTER (
                    WHERE
                        fc.cpt_code IS NOT NULL
                        AND fc.cpt_code != ''
                ),
                ARRAY[]::text []
            ) as cpt_codes,
            (
                SELECT fc2.modifiers
                FROM filtered_claims fc2
                WHERE
                    fc2.claim_id = pci.claim_id
                    AND fc2.modifiers IS NOT NULL
                    AND array_length(fc2.modifiers, 1) > 0
                ORDER BY fc2.claim_from_date DESC
                LIMIT 1
            ) as modifiers
        FROM
            paginated_claim_ids pci
            INNER JOIN filtered_claims fc ON pci.claim_id = fc.claim_id
        GROUP BY
            pci.claim_id
    ),
    payment_totals AS (
        SELECT
            p.claim_id,
            SUM(p.insurance_paid_a)::numeric as total_payment_sum,
            EXTRACT(
                MONTH
                FROM p.payment_entered_date
            ) as payment_month,
            SUM(
                CASE
                    WHEN p.cpt_code != 'H0038'
                    OR p.cpt_code IS NULL THEN p.insurance_paid_a
                    ELSE 0
                END
            ) as non_peer_billed,
            SUM(
                CASE
                    WHEN p.cpt_code = 'H0038' THEN p.insurance_paid_a
                    ELSE 0
                END
            ) as peer_billed
        FROM collaboratemd.new_payment p
            INNER JOIN paginated_claim_ids pci ON p.claim_id = pci.claim_id
        GROUP BY
            p.claim_id,
            EXTRACT(
                MONTH
                FROM p.payment_entered_date
            )
    ),
    payment_totals_by_claim AS (
        SELECT claim_id, SUM(total_payment_sum)::numeric as total_payment_sum
        FROM payment_totals
        GROUP BY
            claim_id
    ),
    payment_totals_by_month AS (
        SELECT
            claim_id,
            payment_month,
            non_peer_billed,
            peer_billed
        FROM payment_totals
    ),
    check_counts AS (
        SELECT p.claim_id, COUNT(DISTINCT p.payment_check) FILTER (
                WHERE
                    p.payment_check IS NOT NULL
                    AND p.payment_check <> ''
            )::int AS total_check_number
        FROM collaboratemd.new_payment p
            INNER JOIN paginated_claim_ids pci ON p.claim_id = pci.claim_id
        GROUP BY
            p.claim_id
    )
SELECT
    ca.claim_id,
    CASE
        WHEN ca.priority = 1 THEN 'inpatient'
        WHEN ca.priority = 2 THEN 'peer'
        ELSE 'outpatient'
    END as claim_type,
    0::numeric as total_charge,
    COALESCE(
        CASE
            WHEN ca.priority = 2 THEN COALESCE(pt.peer_billed, 0)
            ELSE COALESCE(pt.non_peer_billed, 0)
        END,
        0
    )::numeric as total_billed,
    ca.claim_from_date as claim_submission_date,
    ca.claim_from_date as service_date_from,
    ca.claim_to_date as service_date_to,
    ca.patient_id,
    ca.patient_name,
    ca.type_of_service,
    COALESCE(
        (ca.primary_payer ->> 'Name')::text,
        ''
    ) as primary_payer_name,
    COALESCE(
        (ca.primary_payer ->> 'Id')::text,
        ''
    ) as primary_payer_id,
    COALESCE(
        (ca.secondary_payer ->> 'Name')::text,
        ''
    ) as secondary_payer_name,
    COALESCE(
        (ca.secondary_payer ->> 'Id')::text,
        ''
    ) as secondary_payer_id,
    COALESCE(aca.total_charges_all, 0)::int as charge_count,
    ca.cpt_codes as cpt_codes,
    CASE
        WHEN ca.modifiers IS NULL THEN ARRAY[]::text []
        ELSE ca.modifiers
    END as modifiers,
    NULL::text as units,
    NULL::text as facility_pos_code,
    NULL::text as charge_pos_code,
    COALESCE(
        (ca.tertiary_payer ->> 'Name')::text,
        ''
    ) as tertiary_payer_name,
    COALESCE(
        (ca.tertiary_payer ->> 'Id')::text,
        ''
    ) as tertiary_payer_id,
    COALESCE(aca.total_expected_sum_all, 0)::numeric as total_expected,
    COALESCE(aca.total_claims_sum_all, 0)::numeric as total_claims,
    COALESCE(aca.total_expected_sum_all, 0)::numeric as total_expected_agg,
    COALESCE(aca.total_payment_sum_all, 0)::numeric as total_payment,
    COALESCE(aca.total_units_sum_all, 0)::numeric as total_units,
    (
        COALESCE(aca.total_expected_sum_all, 0) - COALESCE(aca.total_payment_sum_all, 0)
    )::numeric as outstanding_amount,
    ca.month_number,
    ca.month_name,
    COALESCE(aca.total_charges_all, 0)::int as total_charges,
    ca.claim_entered_date as claim_entered_date,
    COALESCE(cc.total_check_number, 0)::int as total_check_number
FROM
    claim_aggregations ca
    LEFT JOIN all_charges_aggregations aca ON ca.claim_id = aca.claim_id
    LEFT JOIN payment_totals_by_claim ptc ON ca.claim_id = ptc.claim_id
    LEFT JOIN payment_totals_by_month pt ON ca.claim_id = pt.claim_id
    AND ca.month_number = pt.payment_month
    LEFT JOIN check_counts cc ON ca.claim_id = cc.claim_id
ORDER BY ca.claim_from_date ASC, ca.claim_id;

-- name: GetClaimsCount :one
-- Get total count of DISTINCT claims (after all filters applied)
-- $1::int = year (0 = all years)
-- $2::int = month (0 = all months)
-- $3::text = claim_type filter (empty = all types)
-- $4::boolean = exclude_peer (exclude claims with CPT code H0038)
-- $5::text = search (optional search term for claim_id)
-- $6::boolean = peer_only (include only claims with CPT code H0038)
-- $7::text = category (totalBilled = claims with payments, outstanding = claims without payments, totalClaims/totalExpected = all claims)
-- $8::text = claims_expected_filter ('all', 'claimsGreater', 'expectedGreater')
-- $9::text = payment_status_filter ('all', 'overpaid', 'underpaid', 'fullyPaid')
WITH
    date_range AS (
        SELECT
            CASE
                WHEN $1::int = 0 THEN NULL
                ELSE DATE_TRUNC(
                    'year',
                    MAKE_DATE($1::int, 1, 1)
                )
            END as year_start,
            CASE
                WHEN $1::int = 0 THEN NULL
                ELSE DATE_TRUNC(
                    'year',
                    MAKE_DATE($1::int, 1, 1)
                ) + INTERVAL '1 year'
            END as year_end,
            CASE
                WHEN $2::int = 0 THEN NULL
                ELSE $2::int
            END as month_filter
    ),
    filtered_claims AS (
        SELECT nc.claim_id, nc.cpt_code, nc.revenue_code
        FROM collaboratemd.new_claim nc
            CROSS JOIN date_range dr
        WHERE (
                dr.year_start IS NULL
                OR (
                    nc.claim_from_date >= dr.year_start
                    AND nc.claim_from_date < dr.year_end
                )
            )
            AND (
                dr.month_filter IS NULL
                OR EXTRACT(
                    MONTH
                    FROM nc.claim_from_date
                ) = dr.month_filter
            )
            AND (
                $5::text IS NULL
                OR $5::text = ''
                OR nc.claim_id::text ILIKE '%' || $5::text || '%'
            )
            AND (
                $3::text IS NULL
                OR $3::text = ''
                OR CASE
                    WHEN nc.cpt_code = 'H0038' THEN LOWER($3::text) = 'peer'
                    WHEN (
                        nc.revenue_code IN ('0126', '0128')
                        OR UPPER(nc.cpt_code) IN (
                            'H2019',
                            'H0017',
                            'H0019',
                            '99235',
                            'H0010',
                            'H2036'
                        )
                    )
                    AND nc.cpt_code != 'H0038' THEN LOWER($3::text) = 'inpatient'
                    ELSE LOWER($3::text) = 'outpatient'
                END
            )
            AND NOT (
                $4::boolean = true
                AND nc.cpt_code = 'H0038'
            )
            AND (
                (
                    $6::boolean IS NULL
                    OR $6::boolean = false
                )
                OR (
                    $6::boolean = true
                    AND nc.cpt_code = 'H0038'
                )
            )
    ),
    unique_claim_ids AS (
        SELECT DISTINCT
            claim_id
        FROM (
                SELECT DISTINCT
                    nc.claim_id
                FROM collaboratemd.new_claim nc
                    CROSS JOIN date_range dr_all
                WHERE
                    $7::text = 'outstanding'
                    AND (
                        dr_all.year_start IS NULL
                        OR (
                            nc.claim_from_date >= dr_all.year_start
                            AND nc.claim_from_date < dr_all.year_end
                        )
                    )
                    AND (
                        dr_all.month_filter IS NULL
                        OR EXTRACT(
                            MONTH
                            FROM nc.claim_from_date
                        ) = dr_all.month_filter
                    )
                    AND (
                        $5::text IS NULL
                        OR $5::text = ''
                        OR nc.claim_id::text ILIKE '%' || $5::text || '%'
                    )
                    AND NOT (
                        $4::boolean = true
                        AND nc.cpt_code = 'H0038'
                    )
                    AND (
                        (
                            $6::boolean IS NULL
                            OR $6::boolean = false
                        )
                        OR (
                            $6::boolean = true
                            AND nc.cpt_code = 'H0038'
                        )
                    )
                    AND EXISTS (
                        SELECT 1
                        FROM collaboratemd.new_claim nc2
                        WHERE
                            nc2.claim_id = nc.claim_id
                            AND (
                                dr_all.year_start IS NULL
                                OR (
                                    nc2.claim_from_date >= dr_all.year_start
                                    AND nc2.claim_from_date < dr_all.year_end
                                )
                            )
                            AND (
                                dr_all.month_filter IS NULL
                                OR EXTRACT(
                                    MONTH
                                    FROM nc2.claim_from_date
                                ) = dr_all.month_filter
                            )
                        GROUP BY
                            nc2.claim_id
                        HAVING (
                                COALESCE(SUM(nc2.expected_amount), 0) - COALESCE(
                                    (
                                        SELECT SUM(p_all.insurance_paid_a)
                                        FROM collaboratemd.new_payment p_all
                                        WHERE
                                            p_all.claim_id = nc2.claim_id
                                    ), 0
                                )
                            ) > 0
                        LIMIT 1
                    )
                UNION
                SELECT DISTINCT
                    fc.claim_id
                FROM filtered_claims fc
                WHERE (
                        $7::text IS NULL
                        OR $7::text = ''
                        OR $7::text NOT IN ('totalBilled', 'outstanding')
                    )
                    OR (
                        $7::text = 'totalBilled'
                        AND EXISTS (
                            SELECT 1
                            FROM collaboratemd.new_payment p
                            WHERE
                                p.claim_id = fc.claim_id
                            LIMIT 1
                        )
                    )
            ) all_claim_ids
        WHERE (
                LOWER($3::text) != 'outpatient'
                OR NOT EXISTS (
                    SELECT 1
                    FROM collaboratemd.new_claim nc_check
                    WHERE
                        nc_check.claim_id = all_claim_ids.claim_id
                        AND (
                            (
                                nc_check.revenue_code IN ('0126', '0128')
                                OR UPPER(nc_check.cpt_code) IN (
                                    'H2019',
                                    'H0017',
                                    'H0019',
                                    '99235',
                                    'H0010',
                                    'H2036'
                                )
                            )
                            OR nc_check.cpt_code = 'H0038'
                        )
                )
            )
            AND (
                LOWER($3::text) != 'peer'
                OR NOT EXISTS (
                    SELECT 1
                    FROM collaboratemd.new_claim nc_check
                    WHERE
                        nc_check.claim_id = all_claim_ids.claim_id
                        AND (
                            (
                                nc_check.revenue_code IN ('0126', '0128')
                                OR UPPER(nc_check.cpt_code) IN (
                                    'H2019',
                                    'H0017',
                                    'H0019',
                                    '99235',
                                    'H0010',
                                    'H2036'
                                )
                            )
                            AND nc_check.cpt_code != 'H0038'
                        )
                )
            )
    ),
    aggregated_claims_for_filtering AS (
        SELECT
            nc.claim_id,
            SUM(COALESCE(nc.amount, 0))::numeric as total_claims_sum,
            SUM(
                COALESCE(nc.expected_amount, 0)
            )::numeric as total_expected_sum,
            SUM(
                COALESCE(
                    NULLIF(TRIM(nc.units), '')::numeric,
                    0
                )
            )::numeric as total_units_sum,
            COUNT(
                DISTINCT COALESCE(nc.charge_id, '')
            )::int as total_charges,
            COALESCE(
                (
                    SELECT SUM(p.insurance_paid_a)
                    FROM collaboratemd.new_payment p
                    WHERE
                        p.claim_id = nc.claim_id
                ),
                0
            )::numeric as total_payment_sum
        FROM collaboratemd.new_claim nc
            CROSS JOIN date_range dr
        WHERE
            nc.claim_id IN (
                SELECT claim_id
                FROM unique_claim_ids
            )
            AND (
                dr.year_start IS NULL
                OR (
                    nc.claim_from_date >= dr.year_start
                    AND nc.claim_from_date < dr.year_end
                )
            )
            AND (
                dr.month_filter IS NULL
                OR EXTRACT(
                    MONTH
                    FROM nc.claim_from_date
                ) = dr.month_filter
            )
        GROUP BY
            nc.claim_id
    ),
    final_filtered_claims AS (
        SELECT *
        FROM
            aggregated_claims_for_filtering acf
        WHERE
            -- Claims vs Expected filter ($8)
            (
                $8::text IS NULL
                OR $8::text = ''
                OR $8::text = 'all'
                OR (
                    $8::text = 'claimsGreater'
                    AND acf.total_claims_sum > acf.total_expected_sum
                )
                OR (
                    $8::text = 'expectedGreater'
                    AND acf.total_expected_sum > acf.total_claims_sum
                )
            )
            -- Payment status filter ($9)
            AND (
                $9::text IS NULL
                OR $9::text = ''
                OR $9::text = 'all'
                OR (
                    $9::text = 'overpaid'
                    AND (
                        (
                            acf.total_expected_sum - acf.total_payment_sum
                        ) < 0
                    )
                )
                OR (
                    $9::text = 'underpaid'
                    AND (
                        (
                            acf.total_expected_sum - acf.total_payment_sum
                        ) > 0
                    )
                )
                OR (
                    $9::text = 'fullyPaid'
                    AND ABS(
                        acf.total_expected_sum - acf.total_payment_sum
                    ) = 0
                )
            )
    )
SELECT
    COUNT(DISTINCT ffc.claim_id)::bigint as total_claims,
    COALESCE(SUM(ffc.total_units_sum), 0)::bigint as total_units,
    COALESCE(SUM(ffc.total_charges), 0)::bigint as total_charges,
    COALESCE(SUM(ffc.total_payment_sum), 0)::numeric as total_payment,
    COALESCE(
        SUM(ffc.total_expected_sum),
        0
    )::numeric as total_expected,
    COALESCE(SUM(ffc.total_claims_sum), 0)::numeric as total_amount,
    COALESCE(
        SUM(ffc.total_expected_sum) - SUM(ffc.total_payment_sum),
        0
    )::numeric as outstanding_amount,
    COALESCE(
        (
            SELECT COUNT(DISTINCT p.payment_check)
            FROM collaboratemd.new_payment p
            WHERE
                p.claim_id IN (
                    SELECT claim_id
                    FROM final_filtered_claims
                )
                AND p.payment_check IS NOT NULL
                AND p.payment_check != ''
        ),
        0
    )::bigint as total_check_number
FROM final_filtered_claims ffc;