package dashboard

import (
	"context"

	dbgen "github.com/aci/backend/gen/db"
)

func (r *repositoryImpl) GetDashboardPaymentsPaginated(ctx context.Context, year int32, month int32, excludePeer bool, peerOnly bool, search string, limit int32, offset int32, claimType string, claimsExpectedFilter *string, paymentStatusFilter *string) ([]dbgen.GetDashboardPaymentsPaginatedRow, error) {
	claimsExpectedFilterVal := ""
	if claimsExpectedFilter != nil {
		claimsExpectedFilterVal = *claimsExpectedFilter
	}

	paymentStatusFilterVal := ""
	if paymentStatusFilter != nil {
		paymentStatusFilterVal = *paymentStatusFilter
	}

	rows, err := r.store.Queries().GetDashboardPaymentsPaginated(ctx, dbgen.GetDashboardPaymentsPaginatedParams{
		Column1:  year,
		Column2:  month,
		Column3:  excludePeer,
		Column4:  peerOnly,
		Column5:  search,
		Column6:  limit,
		Column7:  offset,
		Column8:  claimType,
		Column9:  claimsExpectedFilterVal,
		Column10: paymentStatusFilterVal,
	})
	if err != nil {
		return nil, err
	}
	
	return rows, nil
}

func (r *repositoryImpl) GetDashboardPaymentsCount(ctx context.Context, year int32, month int32, excludePeer bool, peerOnly bool, search string, claimType string, claimsExpectedFilter *string, paymentStatusFilter *string) (int64, error) {
	claimsExpectedFilterVal := ""
	if claimsExpectedFilter != nil {
		claimsExpectedFilterVal = *claimsExpectedFilter
	}

	paymentStatusFilterVal := ""
	if paymentStatusFilter != nil {
		paymentStatusFilterVal = *paymentStatusFilter
	}

	count, err := r.store.Queries().GetDashboardPaymentsCount(ctx, dbgen.GetDashboardPaymentsCountParams{
		Column1: year,
		Column2: month,
		Column3: excludePeer,
		Column4: peerOnly,
		Column5: search,
		Column6: claimType,
		Column7: claimsExpectedFilterVal,
		Column8: paymentStatusFilterVal,
	})
	if err != nil {
		return 0, err
	}
	
	return count, nil
}

func (r *repositoryImpl) GetPaymentsByCheckNumberPaginated(ctx context.Context, checkNumber string, year int32, limit int32, offset int32, claimsExpectedFilter *string, paymentStatusFilter *string, search *string) ([]dbgen.GetPaymentsByCheckNumberRow, error) {
	var claimsFilter, paymentFilter, searchStr string
	if claimsExpectedFilter != nil {
		claimsFilter = *claimsExpectedFilter
	}
	if paymentStatusFilter != nil {
		paymentFilter = *paymentStatusFilter
	}
	if search != nil {
		searchStr = *search
	}
	
	rows, err := r.store.Queries().GetPaymentsByCheckNumber(ctx, dbgen.GetPaymentsByCheckNumberParams{
		Column1: checkNumber,
		Column2: year,
		Column3: limit,
		Column4: offset,
		Column5: claimsFilter,
		Column6: paymentFilter,
		Column7: searchStr,
	})
	if err != nil {
		return nil, err
	}
	
	return rows, nil
}

func (r *repositoryImpl) GetPaymentsByCheckNumberCount(ctx context.Context, checkNumber string, year int32, claimsExpectedFilter *string, paymentStatusFilter *string, search *string) (int64, error) {
	var claimsFilter, paymentFilter, searchStr string
	if claimsExpectedFilter != nil {
		claimsFilter = *claimsExpectedFilter
	}
	if paymentStatusFilter != nil {
		paymentFilter = *paymentStatusFilter
	}
	if search != nil {
		searchStr = *search
	}
	
	count, err := r.store.Queries().GetPaymentsByCheckNumberCount(ctx, dbgen.GetPaymentsByCheckNumberCountParams{
		Column1: checkNumber,
		Column2: year,
		Column3: claimsFilter,
		Column4: paymentFilter,
		Column5: searchStr,
	})
	if err != nil {
		return 0, err
	}
	
	return count, nil
}

func (r *repositoryImpl) GetPaymentsByClaimAndCharge(ctx context.Context, params dbgen.GetPaymentsByClaimAndChargeParams) ([]dbgen.GetPaymentsByClaimAndChargeRow, error) {
	return r.store.Queries().GetPaymentsByClaimAndCharge(ctx, params)
}

func (r *repositoryImpl) GetPaymentChecksByClaimAndCharge(ctx context.Context, params dbgen.GetPaymentChecksByClaimAndChargeParams) ([]dbgen.GetPaymentChecksByClaimAndChargeRow, error) {
	return r.store.Queries().GetPaymentChecksByClaimAndCharge(ctx, params)
}

func (r *repositoryImpl) GetPaymentChecksByClaimAndChargeCount(ctx context.Context, params dbgen.GetPaymentChecksByClaimAndChargeCountParams) (int64, error) {
	count, err := r.store.Queries().GetPaymentChecksByClaimAndChargeCount(ctx, params)
	if err != nil {
		return 0, err
	}
	return count, nil
}

