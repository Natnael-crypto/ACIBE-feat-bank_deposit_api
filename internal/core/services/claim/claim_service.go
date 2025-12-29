package claim

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	dbgen "github.com/aci/backend/gen/db"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
)

// ServiceLineInput represents a service line from the claim JSON input
type ServiceLineInput struct {
	Code       string   `json:"Code"`
	Units      float64  `json:"Units"`
	Amount     string   `json:"Amount,omitempty"`
	CPTCode    string   `json:"CPTCode"`
	Modifiers  []string `json:"Modifiers"`
	LineNumber string   `json:"LineNumber"`
}

// ClaimInput represents input data for a single claim computation
type ClaimInput struct {
	TypeOfService        string
	ServiceDateFrom      time.Time
	ServiceDateTo        time.Time
	PrimaryInsuranceName string
	ServiceLinesJSON     []byte
}

// BatchClaimInput represents input data for batch computation
type BatchClaimInput struct {
	TypeOfService        []string
	ServiceDateFrom      []time.Time
	ServiceDateTo        []time.Time
	PrimaryInsuranceName []string
	ServiceLinesJSON     [][]byte
}

// ComputeTotalExpected computes the total expected reimbursement for a single claim
func ComputeTotalExpected(ctx context.Context, q *dbgen.Queries, claimData ClaimInput) (pgtype.Numeric, error) {
	// Parse service lines
	var serviceLines []ServiceLineInput
	if err := json.Unmarshal(claimData.ServiceLinesJSON, &serviceLines); err != nil {
		return pgtype.Numeric{}, fmt.Errorf("failed to parse service lines: %w", err)
	}

	var totalExpected decimal.Decimal = decimal.Zero

	// TODO: Rate calculation functionality removed - methods GetAllOutPatientRates and GetAllInPatientRates do not exist
	// Return zero for now

	// Convert to pgtype.Numeric
	result := pgtype.Numeric{}
	if err := result.Scan(totalExpected.String()); err != nil {
		return pgtype.Numeric{}, fmt.Errorf("failed to convert to numeric: %w", err)
	}

	return result, nil
}

// ComputeBatchTotalExpected computes total expected for a batch of claims
func ComputeBatchTotalExpected(ctx context.Context, q *dbgen.Queries, batchData BatchClaimInput) ([]pgtype.Numeric, error) {
	if len(batchData.TypeOfService) == 0 {
		return []pgtype.Numeric{}, nil
	}

	results := make([]pgtype.Numeric, len(batchData.TypeOfService))

	for i := range batchData.TypeOfService {
		claimData := ClaimInput{
			TypeOfService:        batchData.TypeOfService[i],
			ServiceDateFrom:      batchData.ServiceDateFrom[i],
			ServiceDateTo:        batchData.ServiceDateTo[i],
			PrimaryInsuranceName: batchData.PrimaryInsuranceName[i],
			ServiceLinesJSON:     batchData.ServiceLinesJSON[i],
		}

		total, err := ComputeTotalExpected(ctx, q, claimData)
		if err != nil {
			return nil, fmt.Errorf("failed to compute total for claim %d: %w", i, err)
		}

		results[i] = total
	}

	return results, nil
}

