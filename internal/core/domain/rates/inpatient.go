package ratesdomain

type InPatientRate struct {
	Codes               []string     `json:"codes"`
	Type                string       `json:"type"`
	Quantifiers         []string     `json:"quantifiers"`
	Modifiers           []string     `json:"modifiers"`
	Description         string       `json:"description"`
	APGWeight           float64      `json:"apg_weight,omitempty"`
	RatePeriods         []RatePeriod `json:"rate_periods"`
	PlusIndividualCodes []string     `json:"plus_individual_codes,omitempty"`
	Note                string       `json:"note"`
}

