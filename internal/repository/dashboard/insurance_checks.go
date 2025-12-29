package dashboard

import (
	"context"

	dbgen "github.com/aci/backend/gen/db"
	"github.com/jackc/pgx/v5/pgtype"
)

func (r *repositoryImpl) GetInsuranceChecksPaginated(ctx context.Context, year int32, month int32, search string, limit int32, cursor string) ([]dbgen.GetInsuranceChecksPaginatedRow, error) {
	rows, err := r.store.Queries().GetInsuranceChecksPaginated(ctx, dbgen.GetInsuranceChecksPaginatedParams{
		Column1: year,
		Column2: month,
		Column3: search,
		Column4: limit,
		Column5: cursor,
	})
	if err != nil {
		return nil, err
	}
	
	return rows, nil
}

func (r *repositoryImpl) GetInsuranceChecksCount(ctx context.Context, year int32, month int32, search string) (int64, error) {
	count, err := r.store.Queries().GetInsuranceChecksCount(ctx, dbgen.GetInsuranceChecksCountParams{
		Column1: year,
		Column2: month,
		Column3: search,
	})
	if err != nil {
		return 0, err
	}
	
	return count, nil
}

func (r *repositoryImpl) GetInsuranceChecksTotalDeposit(ctx context.Context, year int32, month int32, search string) (pgtype.Numeric, error) {
	return r.store.Queries().GetInsuranceChecksTotalDeposit(ctx, dbgen.GetInsuranceChecksTotalDepositParams{
		Column1: year,
		Column2: month,
		Column3: search,
	})
}

