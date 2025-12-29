package ratesdomain

type OutPatientRate struct {
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
type RatePeriod struct {
	StartDate   string  `json:"start_date"`
	EndDate     string  `json:"end_date"`
	OnSiteRate  float64 `json:"on_site_rate"`
	OffSiteRate float64 `json:"off_site_rate"`
	Note        string  `json:"note"`
}
