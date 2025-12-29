-- Drop triggers
DROP TRIGGER IF EXISTS trigger_update_available_years_on_update ON collaboratemd.new_claim;
DROP TRIGGER IF EXISTS trigger_update_available_years_on_insert ON collaboratemd.new_claim;

-- Drop functions
DROP FUNCTION IF EXISTS collaboratemd.update_available_years_on_claim_change();
DROP FUNCTION IF EXISTS collaboratemd.upsert_available_year(INTEGER);
DROP FUNCTION IF EXISTS collaboratemd.backfill_available_years();

-- Drop table
DROP TABLE IF EXISTS collaboratemd.available_years;

