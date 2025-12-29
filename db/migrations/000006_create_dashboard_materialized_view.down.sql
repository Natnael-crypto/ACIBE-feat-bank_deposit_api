-- Rollback: Drop materialized view and related functions

DROP FUNCTION IF EXISTS collaboratemd.get_dashboard_available_years();
DROP FUNCTION IF EXISTS collaboratemd.refresh_dashboard_annual_data();
DROP MATERIALIZED VIEW IF EXISTS collaboratemd.dashboard_annual_data;

