package repository

import (
	"context"
	"time"

	dbgen "github.com/aci/backend/gen/db"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type Store struct {
	queries *dbgen.Queries
	conn    *pgxpool.Pool
}

func NewStore(conn *pgxpool.Pool) *Store {

	queries := dbgen.New(conn)
	return &Store{
		queries: queries,
		conn:    conn,
	}
}

// GetConn returns the database connection pool
func (s *Store) GetConn() *pgxpool.Pool {
	return s.conn
}

// Queries returns the database queries instance
func (s *Store) Queries() *dbgen.Queries {
	return s.queries
}

func OpenDB(url string) (*pgxpool.Pool, func(), error) {
	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, func() {}, err
	}

	config.MaxConns = 80
	config.MinConns = 20
	config.MaxConnIdleTime = time.Minute
	config.MaxConnLifetime = 60 * time.Minute
	config.HealthCheckPeriod = 1 * time.Minute

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, func() {}, err
	}

	if err := conn.Ping(ctx); err != nil {
		return nil, func() {}, err
	}

	return conn, func() {
		conn.Close()
	}, nil
}
func toNullableTime(t pgtype.Timestamptz) *time.Time {
	if t.Valid {
		return &t.Time
	}
	return nil
}

func NumericToDecimal(n pgtype.Numeric) decimal.Decimal {
	if !n.Valid {
		return decimal.Zero
	}
	return decimal.NewFromBigInt(n.Int, int32(n.Exp))
}
func DecimalToNumeric(d decimal.Decimal) pgtype.Numeric {
	return pgtype.Numeric{
		Int:   d.Coefficient(),
		Exp:   d.Exponent(),
		Valid: true,
	}
}

func toDate(date time.Time) pgtype.Date {
	// only year month day
	formattedDate := date.Format("2006-01-02")
	parsedDate, err := time.ParseInLocation("2006-01-02", formattedDate, time.UTC)
	if err != nil {
		return pgtype.Date{Valid: false}
	}
	return pgtype.Date{Time: parsedDate, Valid: true}
}

// dateToTimestamptz converts pgtype.Date to pgtype.Timestamptz
func dateToTimestamptz(d pgtype.Date) pgtype.Timestamptz {
	if !d.Valid {
		return pgtype.Timestamptz{Valid: false}
	}
	// Convert date to timestamp at midnight UTC
	timestamp := time.Date(d.Time.Year(), d.Time.Month(), d.Time.Day(), 0, 0, 0, 0, time.UTC)
	return pgtype.Timestamptz{Time: timestamp, Valid: true}
}
