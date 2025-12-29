package dashboard

import (
	"context"

	dbgen "github.com/aci/backend/gen/db"
)

func (r *repositoryImpl) GetClaims(ctx context.Context, year *int32, month *int32, category string, claimType *string, excludePeer bool, search *string, peerOnly bool, limit int32, offset int32, claimsExpectedFilter *string, paymentStatusFilter *string) ([]dbgen.GetClaimsRow, error) {
	yearVal := int32(0)
	if year != nil {
		yearVal = *year
	}

	monthVal := int32(0)
	if month != nil {
		monthVal = *month
	}

	claimTypeVal := ""
	if claimType != nil {
		claimTypeVal = *claimType
	}

	searchVal := ""
	if search != nil {
		searchVal = *search
	}

	claimsExpectedFilterVal := ""
	if claimsExpectedFilter != nil {
		claimsExpectedFilterVal = *claimsExpectedFilter
	}

	paymentStatusFilterVal := ""
	if paymentStatusFilter != nil {
		paymentStatusFilterVal = *paymentStatusFilter
	}

	// Use the generated query from get_claims.sql
	rows, err := r.store.Queries().GetClaims(ctx, dbgen.GetClaimsParams{
		Column1:  yearVal,
		Column2:  monthVal,
		Column3:  category,
		Column4:  excludePeer,
		Column5:  searchVal,
		Column6:  peerOnly,
		Column7:  claimTypeVal,
		Column8:  limit,
		Column9:  offset,
		Column10: claimsExpectedFilterVal,
		Column11: paymentStatusFilterVal,
	})
	if err != nil {
		return nil, err
	}

	return rows, nil
}

func (r *repositoryImpl) GetDashboardClaimsCount(ctx context.Context, year *int32, month *int32, category string, claimType *string, excludePeer bool, search *string, peerOnly bool, claimsExpectedFilter *string, paymentStatusFilter *string) (dbgen.GetClaimsCountRow, error) {
	yearVal := int32(0)
	if year != nil {
		yearVal = *year
	}

	monthVal := int32(0)
	if month != nil {
		monthVal = *month
	}

	claimTypeVal := ""
	if claimType != nil {
		claimTypeVal = *claimType
	}

	searchVal := ""
	if search != nil {
		searchVal = *search
	}

	claimsExpectedFilterVal := ""
	if claimsExpectedFilter != nil {
		claimsExpectedFilterVal = *claimsExpectedFilter
	}

	paymentStatusFilterVal := ""
	if paymentStatusFilter != nil {
		paymentStatusFilterVal = *paymentStatusFilter
	}

	// Use the generated query from get_claims.sql
	row, err := r.store.Queries().GetClaimsCount(ctx, dbgen.GetClaimsCountParams{
		Column1: yearVal,
		Column2: monthVal,
		Column3: claimTypeVal,
		Column4: excludePeer,
		Column5: searchVal,
		Column6: peerOnly,
		Column7: category,
		Column8: claimsExpectedFilterVal,
		Column9: paymentStatusFilterVal,
	})
	if err != nil {
		return dbgen.GetClaimsCountRow{}, err
	}

	return row, nil
}

func (r *repositoryImpl) GetClaimCharges(ctx context.Context, claimID string, year *int32, month *int32) ([]dbgen.GetClaimChargesRow, error) {
	yearVal := int32(0)
	if year != nil {
		yearVal = *year
	}

	monthVal := int32(0)
	if month != nil {
		monthVal = *month
	}

	rows, err := r.store.Queries().GetClaimCharges(ctx, dbgen.GetClaimChargesParams{
		Column1: claimID,
		Column2: yearVal,
		Column3: monthVal,
	})
	if err != nil {
		return nil, err
	}

	return rows, nil
}

func (r *repositoryImpl) GetClaimById(ctx context.Context, claimID string) ([]dbgen.GetClaimByIdRow, error) {
	rows, err := r.store.Queries().GetClaimById(ctx, claimID)
	if err != nil {
		return nil, err
	}

	return rows, nil
}
