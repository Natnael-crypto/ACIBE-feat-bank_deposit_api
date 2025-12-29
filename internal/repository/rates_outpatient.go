package repository

import (
	"context"
	"encoding/json"

	dbgen "github.com/aci/backend/gen/db"
	ratesdomain "github.com/aci/backend/internal/core/domain/rates"
	"github.com/jackc/pgx/v5/pgtype"
	// "github.com/jackc/pgx/pgtype"
)

func (s *Store) CreateOutPatientRate(ctx context.Context, rate ratesdomain.OutPatientRate) error {
	ratePeriods, err := json.Marshal(rate.RatePeriods)
	if err != nil {
		return err
	}
	_, err = s.queries.CreateOutPatientRate(ctx, dbgen.CreateOutPatientRateParams{
		Codes:       rate.Codes,
		Modifiers:   rate.Modifiers,
		Quantifiers: rate.Quantifiers,
		Type:        rate.Type,
		Description: rate.Description,
		RatePeriods: ratePeriods,
		Note: pgtype.Text{
			String: rate.Note,
			Valid:  true,
		},
		PlusIndividualCodes: rate.PlusIndividualCodes,
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *Store) GetOutPatientRate(ctx context.Context, code string, modifiers string) (ratesdomain.OutPatientRate, error) {
	dbRate, err := s.queries.GetOutPatientRate(ctx, dbgen.GetOutPatientRateParams{
		Column1: code,
		Column2: modifiers,
	})
	if err != nil {
		return ratesdomain.OutPatientRate{}, err
	}

	var ratePeriods []ratesdomain.RatePeriod
	if len(dbRate.RatePeriods) > 0 {
		if err := json.Unmarshal(dbRate.RatePeriods, &ratePeriods); err != nil {
			return ratesdomain.OutPatientRate{}, err
		}
	}

	note := ""
	if dbRate.Note.Valid {
		note = dbRate.Note.String
	}

	return ratesdomain.OutPatientRate{
		Codes:               dbRate.Codes,
		Type:                dbRate.Type,
		Quantifiers:         dbRate.Quantifiers,
		Modifiers:           dbRate.Modifiers,
		Description:         dbRate.Description,
		RatePeriods:         ratePeriods,
		PlusIndividualCodes: dbRate.PlusIndividualCodes,
		Note:                note,
	}, nil
}

func (s *Store) GetAllOutPatientRates(ctx context.Context) ([]ratesdomain.OutPatientRate, error) {
	dbRates, err := s.queries.GetAllOutPatientRates(ctx)
	if err != nil {
		return nil, err
	}

	var rates []ratesdomain.OutPatientRate
	for _, dbRate := range dbRates {
		var ratePeriods []ratesdomain.RatePeriod
		if len(dbRate.RatePeriods) > 0 {
			if err := json.Unmarshal(dbRate.RatePeriods, &ratePeriods); err != nil {
				return nil, err
			}
		}

		note := ""
		if dbRate.Note.Valid {
			note = dbRate.Note.String
		}

		rates = append(rates, ratesdomain.OutPatientRate{
			Codes:               dbRate.Codes,
			Type:                dbRate.Type,
			Quantifiers:         dbRate.Quantifiers,
			Modifiers:           dbRate.Modifiers,
			Description:         dbRate.Description,
			RatePeriods:         ratePeriods,
			PlusIndividualCodes: dbRate.PlusIndividualCodes,
			Note:                note,
		})
	}
	return rates, nil
}