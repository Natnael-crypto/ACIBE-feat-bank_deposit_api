package dashboard

import (
	"context"
	"strings"

	dbgen "github.com/aci/backend/gen/db"
	"github.com/aci/backend/internal/repository"
	"github.com/aci/backend/internal/types/dashboard"
)

// BankDepositRepository interface for bank deposit operations
type BankDepositRepository interface {
	GetBankDepositData(ctx context.Context, year int32) ([]dashboard.BankDepositDataRow, error)
	GetBankDepositClaimsPaginated(ctx context.Context, year int32, month int32, search string, limit int32, cursor string) ([]dbgen.GetBankDepositClaimsPaginatedRow, error)
	GetBankDepositClaimsCount(ctx context.Context, year int32, month int32, search string) (int64, error)
	GetBankDepositAppliedPaymentsPaginated(ctx context.Context, year int32, month int32, search string, limit int32, cursor string) ([]dbgen.GetBankDepositAppliedPaymentsPaginatedRow, error)
	GetBankDepositAppliedPaymentsCount(ctx context.Context, year int32, month int32, search string) (int64, error)
}

// NewBankDepositRepository creates a new BankDepositRepository that wraps the dashboard repository
func NewBankDepositRepository(store *repository.Store) BankDepositRepository {
	return NewRepository(store)
}

// Bank deposit methods implementation
func (r *repositoryImpl) GetBankDepositData(ctx context.Context, year int32) ([]dashboard.BankDepositDataRow, error) {
	rows, err := r.store.Queries().GetBankDepositData(ctx, year)
	if err != nil {
		return nil, err
	}

	result := make([]dashboard.BankDepositDataRow, 0, len(rows))
	for _, row := range rows {
		// Type assert MonthName from interface{} to string
		monthName := ""
		if row.MonthName != nil {
			if str, ok := row.MonthName.(string); ok {
				monthName = strings.TrimSpace(str)
			}
		}
		
		result = append(result, dashboard.BankDepositDataRow{
			MonthNumber:         int(row.MonthNumber),
			MonthName:           monthName,
			TotalClaimsNumber:   row.TotalClaimsNumber,
			TotalClaimsAmount:   repository.NumericToDecimal(row.TotalClaimsAmount),
			TotalExpectedAmount: repository.NumericToDecimal(row.TotalExpectedAmount),
			TotalDepositAmount:  repository.NumericToDecimal(row.TotalDepositAmount),
			TotalAppliedAmount:  repository.NumericToDecimal(row.TotalAppliedAmount),
		})
	}

	return result, nil
}

func (r *repositoryImpl) GetBankDepositClaimsPaginated(ctx context.Context, year int32, month int32, search string, limit int32, cursor string) ([]dbgen.GetBankDepositClaimsPaginatedRow, error) {
	rows, err := r.store.Queries().GetBankDepositClaimsPaginated(ctx, dbgen.GetBankDepositClaimsPaginatedParams{
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

func (r *repositoryImpl) GetBankDepositClaimsCount(ctx context.Context, year int32, month int32, search string) (int64, error) {
	count, err := r.store.Queries().GetBankDepositClaimsCount(ctx, dbgen.GetBankDepositClaimsCountParams{
		Column1: year,
		Column2: month,
		Column3: search,
	})
	if err != nil {
		return 0, err
	}
	
	return count, nil
}

func (r *repositoryImpl) GetBankDepositAppliedPaymentsPaginated(ctx context.Context, year int32, month int32, search string, limit int32, cursor string) ([]dbgen.GetBankDepositAppliedPaymentsPaginatedRow, error) {
	rows, err := r.store.Queries().GetBankDepositAppliedPaymentsPaginated(ctx, dbgen.GetBankDepositAppliedPaymentsPaginatedParams{
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

func (r *repositoryImpl) GetBankDepositAppliedPaymentsCount(ctx context.Context, year int32, month int32, search string) (int64, error) {
	count, err := r.store.Queries().GetBankDepositAppliedPaymentsCount(ctx, dbgen.GetBankDepositAppliedPaymentsCountParams{
		Column1: year,
		Column2: month,
		Column3: search,
	})
	if err != nil {
		return 0, err
	}
	
	return count, nil
}

