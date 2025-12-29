package insurance

import (
	"time"

	"github.com/shopspring/decimal"
)

// https://webapi.collaboratemd.com/v1/customer/10028881/reports/10079605/filter/10142096/run
type InsuranceCheck struct {
	Id                       string
	CheckPayer               string
	CheckDate                time.Time
	DepositDate              time.Time
	CheckNumber              string
	CheckAmount              decimal.Decimal
	CheckAppliedAmount       decimal.Decimal
	ProviderAdjustmentAmount decimal.Decimal
	CheckRemainingAmount     decimal.Decimal
	CheckCommitted           bool
}
