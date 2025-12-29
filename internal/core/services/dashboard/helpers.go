package dashboard

import (
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
)

// Helper function to create a float64 pointer
func floatPtr(f float64) *float64 {
	return &f
}

// Helper function to format date to US format (MM/DD/YYYY)
func formatDate(date pgtype.Date) string {
	if !date.Valid {
		return ""
	}
	// Format as MM/DD/YYYY (US date format)
	return date.Time.Format("01/02/2006")
}

// Helper function to convert pgtype.Numeric to decimal.Decimal
func numericToDecimal(n pgtype.Numeric) decimal.Decimal {
	if !n.Valid {
		return decimal.Zero
	}
	return decimal.NewFromBigInt(n.Int, int32(n.Exp))
}

// Helper function to convert interface{} to string
func interfaceToString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// Helper function to format date from interface{}
func formatDateInterface(dateInterface interface{}) string {
	switch v := dateInterface.(type) {
	case pgtype.Date:
		return formatDate(v)
	case time.Time:
		return v.Format("01/02/2006") // MM/DD/YYYY
	case string:
		if t, err := time.Parse("2006-01-02", v); err == nil {
			return t.Format("01/02/2006")
		} else if _, err := time.Parse("01/02/2006", v); err == nil {
			return v
		}
		return v
	default:
		return ""
	}
}

