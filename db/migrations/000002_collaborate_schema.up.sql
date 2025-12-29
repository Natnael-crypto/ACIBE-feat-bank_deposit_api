CREATE SCHEMA collaboratemd;

CREATE TABLE collaboratemd.out_patient_rate
(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    codes TEXT[] NOT NULL,
    modifiers TEXT[] NOT NULL DEFAULT '{}',
    quantifiers TEXT[] NOT NULL DEFAULT '{}',
    type TEXT NOT NULL,
    description TEXT NOT NULL,
    rate_periods JSONB NOT NULL,
    plus_individual_codes TEXT[] NOT NULL DEFAULT '{}',
    note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE
    collaboratemd.claim (
        claim_id VARCHAR(255) NOT NULL PRIMARY KEY,
        claim_frequency CHAR(1) NOT NULL,
        original_claim_number VARCHAR(255),
        total_charge DECIMAL(18, 2) NOT NULL,
        claim_submission_date DATE NOT NULL,
        service_date_from DATE NOT NULL,
        service_date_to DATE NOT NULL,
        patient_id VARCHAR(255) NOT NULL,
        patient_name VARCHAR(255) NOT NULL,
        service_lines JSONB NOT NULL,
        type_of_service VARCHAR(100) NOT NULL,
        primary_insurance_name VARCHAR(255) NOT NULL,
        primary_insurance_id VARCHAR(255) NOT NULL,
        secondary_insurance_name VARCHAR(255) NOT NULL,
        secondary_insurance_id VARCHAR(255) NOT NULL,
        raw_text TEXT NOT NULL
    );

CREATE TABLE collaboratemd.payment (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    claim_id VARCHAR(255) NOT NULL,
    patient_id VARCHAR(255) NOT NULL,
    total_claim_charge DECIMAL(18, 2) NOT NULL,
    total_paid_amount DECIMAL(18, 2) NOT NULL,
    claim_status VARCHAR(100) NOT NULL,
    insurance_name VARCHAR(255) NOT NULL,
    insurance_id VARCHAR(255) NOT NULL,
    raw_text TEXT NOT NULL,
    service_lines JSONB NOT NULL,
    payment_date DATE NOT NULL,
    posted_date DATE NOT NULL,
    charge_date DATE NOT NULL,
    bpr_date DATE NOT NULL,
    trn VARCHAR(255) NOT NULL,
    claim_level_adjustment JSONB NOT NULL,
    CONSTRAINT unique_claim_payment UNIQUE (claim_id, trn)
);





CREATE TABLE collaboratemd.new_payment (
    payment_received_date DATE NOT NULL,
    deposit_date DATE NOT NULL,
    payment_entered_date DATE NOT NULL,
    charge_from_date DATE NOT NULL,
    claim_id VARCHAR(255) NOT NULL,
    charge_id VARCHAR(255) NOT NULL,
    cpt_code VARCHAR(255) NOT NULL,
    modifiers TEXT[] NOT NULL DEFAULT '{}',
    insurance_paid_a DECIMAL(18, 2) NOT NULL,
    insurance_adjustment DECIMAL(18, 2) NOT NULL,
    charge_amount DECIMAL(18, 2) NOT NULL,
    credit_id VARCHAR(255) NOT NULL,
    payment_check VARCHAR(255) NOT NULL,
    remark TEXT NOT NULL,
    unpaid_reason TEXT NOT NULL,
    adjustment_reasons TEXT NOT NULL DEFAULT '{}',
    payer_id VARCHAR(255) NOT NULL,
    payment_name VARCHAR(255) NOT NULL,
    patient_id VARCHAR(255) NOT NULL,
    patient_name VARCHAR(255) NOT NULL,
    revenue_code VARCHAR(255) NOT NULL,
    is_refund BOOLEAN NOT NULL DEFAULT FALSE,
    is_provider_adjustment BOOLEAN NOT NULL DEFAULT FALSE,
    is_era_adjustment BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT unique_payment_entered_date_claim_id_charge_id_cpt_code_payment_check UNIQUE (payment_entered_date, claim_id,credit_id, charge_id, cpt_code, payment_check,is_refund,is_provider_adjustment,is_era_adjustment)
);
CREATE OR REPLACE FUNCTION update_payment_on_conflict()
RETURNS TRIGGER AS
$$
DECLARE
    existing collaboratemd.new_payment%ROWTYPE;
BEGIN
  
    SELECT *
    INTO existing
    FROM collaborate.payments
   WHERE payment_entered_date   = NEW.payment_entered_date
      AND claim_id               = NEW.claim_id
      AND charge_id              = NEW.charge_id
      AND cpt_code               = NEW.cpt_code
      AND payment_check          = NEW.payment_check
      AND is_refund              = NEW.is_refund
      AND is_provider_adjustment = NEW.is_provider_adjustment
      AND is_era_adjustment      = NEW.is_era_adjustment
      AND credit_id              = NEW.credit_id
    LIMIT 1;

    IF FOUND THEN
        
        
        IF NEW.deposit_date = '0001-01-01' THEN
            UPDATE collaborate.payments
            SET deposit_date = NEW.deposit_date
            WHERE id = existing.id;
        END IF;

       
        RETURN NULL;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

---------------------------

--------------------------------

CREATE TABLE collaboratemd.new_claim (
    claim_id VARCHAR(255) NOT NULL,
    charge_id VARCHAR(255) NOT NULL,
    total_claim DECIMAL(18, 2) NOT NULL,
    claim_entered_date DATE NOT NULL,
    claim_from_date DATE NOT NULL,
    claim_to_date DATE NOT NULL,
    claim_type VARCHAR(255) NOT NULL,
    primary_payer JSONB ,
    secondary_payer JSONB ,
    tertiary_payer JSONB ,
    patient_id VARCHAR(255) NOT NULL,
    patient_name VARCHAR(255) NOT NULL,
    cpt_code VARCHAR(255) NOT NULL,
    modifiers TEXT[] NOT NULL,
    amount DECIMAL(18, 2) NOT NULL,
    units VARCHAR(255) NOT NULL,
    revenue_code VARCHAR(255) NOT NULL,
    facility_pos_code VARCHAR(255) NOT NULL,
    charge_pos_code VARCHAR(255) NOT NULL,
    value_code_1 VARCHAR(255) NOT NULL,
    value_amount_1 INTEGER NOT NULL,
    claim_status VARCHAR(255) NOT NULL,
    expected_amount DECIMAL(18, 2) NOT NULL,
    PRIMARY KEY (claim_id,charge_id)
);
------------------------------------------------------------------------


