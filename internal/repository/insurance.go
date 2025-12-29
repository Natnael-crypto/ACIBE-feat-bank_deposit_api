package repository

import (
	"context"
	"time"

	dbgen "github.com/aci/backend/gen/db"
	"github.com/aci/backend/internal/core/services/insurance"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Store) CreateInsuranceChecks(ctx context.Context, checks []insurance.InsuranceCheck) error {
	params := dbgen.CreateInsuranceChecksParams{
		Column1:  make([]string, len(checks)),             // id
		Column2:  make([]string, len(checks)),             // check_payer
		Column3:  make([]pgtype.Timestamptz, len(checks)), // check_date
		Column4:  make([]pgtype.Timestamptz, len(checks)), // deposit_date
		Column5:  make([]string, len(checks)),             // check_number
		Column6:  make([]pgtype.Numeric, len(checks)),     // check_amount
		Column7:  make([]pgtype.Numeric, len(checks)),     // check_applied_amount
		Column8:  make([]pgtype.Numeric, len(checks)),     // provider_adjustment_amount
		Column9:  make([]pgtype.Numeric, len(checks)),     // check_remaining_amount
		Column10: make([]bool, len(checks)),               // check_committed
	}

	for i, check := range checks {
		params.Column1[i] = check.Id
		params.Column2[i] = check.CheckPayer
		params.Column3[i] = toTimestamptz(check.CheckDate)
		params.Column4[i] = toTimestamptz(check.DepositDate)
		params.Column5[i] = check.CheckNumber
		params.Column6[i] = DecimalToNumeric(check.CheckAmount)
		params.Column7[i] = DecimalToNumeric(check.CheckAppliedAmount)
		params.Column8[i] = DecimalToNumeric(check.ProviderAdjustmentAmount)
		params.Column9[i] = DecimalToNumeric(check.CheckRemainingAmount)
		params.Column10[i] = check.CheckCommitted
	}

	return s.queries.CreateInsuranceChecks(ctx, params)
}

func toTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{
		Time:  t,
		Valid: !t.IsZero(),
	}
}
