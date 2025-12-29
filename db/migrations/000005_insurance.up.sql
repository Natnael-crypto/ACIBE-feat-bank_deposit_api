CREATE TABLE collaboratemd.insurance_check (
    id TEXT PRIMARY KEY,
    check_payer TEXT NOT NULL,
    check_date DATE NOT NULL,
    deposit_date DATE NOT NULL,
    check_number TEXT NOT NULL,
    check_amount NUMERIC(18, 2) NOT NULL,
    check_applied_amount NUMERIC(18, 2) NOT NULL,
    provider_adjustment_amount NUMERIC(18, 2) NOT NULL,
    check_remaining_amount NUMERIC(18, 2) NOT NULL,
    check_committed BOOLEAN NOT NULL
)