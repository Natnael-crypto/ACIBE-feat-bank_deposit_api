-- Create available_years table to store distinct years from claims
-- Years are extracted from claim_entered_date (when data was inserted into the table)
-- This eliminates the need to query all claims every time we need available years

CREATE TABLE IF NOT EXISTS collaboratemd.available_years (
    year INTEGER NOT NULL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create index for faster queries (though year is already the primary key)
CREATE INDEX IF NOT EXISTS idx_available_years_year ON collaboratemd.available_years(year DESC);

-- Function to insert or update available year
CREATE OR REPLACE FUNCTION collaboratemd.upsert_available_year(claim_year INTEGER)
RETURNS VOID AS $$
BEGIN
    INSERT INTO collaboratemd.available_years (year, created_at, updated_at)
    VALUES (claim_year, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
    ON CONFLICT (year) DO UPDATE
    SET updated_at = CURRENT_TIMESTAMP;
END;
$$ LANGUAGE plpgsql;

-- Trigger function to automatically update available_years when claims are inserted or updated
CREATE OR REPLACE FUNCTION collaboratemd.update_available_years_on_claim_change()
RETURNS TRIGGER AS $$
DECLARE
    claim_year INTEGER;
BEGIN
    -- Extract year from claim_entered_date (when the data was inserted into the table)
    IF NEW.claim_entered_date IS NOT NULL THEN
        claim_year := EXTRACT(YEAR FROM NEW.claim_entered_date)::INTEGER;
        
        -- Insert or update the year in available_years table
        PERFORM collaboratemd.upsert_available_year(claim_year);
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for INSERT operations
CREATE TRIGGER trigger_update_available_years_on_insert
AFTER INSERT ON collaboratemd.new_claim
FOR EACH ROW
EXECUTE FUNCTION collaboratemd.update_available_years_on_claim_change();

-- Create trigger for UPDATE operations (in case claim_entered_date changes)
CREATE TRIGGER trigger_update_available_years_on_update
AFTER UPDATE OF claim_entered_date ON collaboratemd.new_claim
FOR EACH ROW
WHEN (OLD.claim_entered_date IS DISTINCT FROM NEW.claim_entered_date)
EXECUTE FUNCTION collaboratemd.update_available_years_on_claim_change();

-- Function to backfill existing years from claims table
-- This can be run manually if needed: SELECT collaboratemd.backfill_available_years();
CREATE OR REPLACE FUNCTION collaboratemd.backfill_available_years()
RETURNS VOID AS $$
BEGIN
    INSERT INTO collaboratemd.available_years (year, created_at, updated_at)
    SELECT DISTINCT 
        EXTRACT(YEAR FROM claim_entered_date)::INTEGER as year,
        CURRENT_TIMESTAMP as created_at,
        CURRENT_TIMESTAMP as updated_at
    FROM collaboratemd.new_claim
    WHERE claim_entered_date IS NOT NULL
    ON CONFLICT (year) DO NOTHING;
END;
$$ LANGUAGE plpgsql;

