package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/aci/backend/internal/collaborate"
	"github.com/aci/backend/internal/config"
	"github.com/aci/backend/internal/core/services/claim"
	"github.com/aci/backend/internal/core/services/insurance"
	"github.com/aci/backend/internal/core/services/payment"
	"github.com/aci/backend/internal/repository"
)

func main() {
	// Define flags for which sync jobs to run
	syncClaims := flag.Bool("claims", false, "Sync new claims")
	syncPayments := flag.Bool("payments", false, "Sync new payments")
	syncInsurance := flag.Bool("insurance", false, "Sync insurance checks")
	all := flag.Bool("all", false, "Run all sync jobs (claims, payments, insurance)")
	
	flag.Parse()

	// If no flags are provided, show usage
	if !*syncClaims && !*syncPayments && !*syncInsurance && !*all {
		fmt.Println("Usage: go run cmd/sync/main.go [options]")
		fmt.Println("\nOptions:")
		fmt.Println("  -claims       Sync new claims to newclaims table")
		fmt.Println("  -payments     Sync new payments to new payment table")
		fmt.Println("  -insurance    Sync insurance checks to insurance check table")
		fmt.Println("  -all          Run all sync jobs (claims, payments, insurance)")
		fmt.Println("\nExamples:")
		fmt.Println("  go run cmd/sync/main.go -all")
		fmt.Println("  go run cmd/sync/main.go -claims")
		fmt.Println("  go run cmd/sync/main.go -claims -payments")
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.NewConfig()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	// Connect to database
	conn, close, err := repository.OpenDB(cfg.DbUrl)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer close()

	// Initialize logger
	logger := slog.Default()
	
	// Initialize store and collaborate API
	store := repository.NewStore(conn)
	collabApi := collaborate.NewAPI(cfg.Collaborate.Username, cfg.Collaborate.Password, store)

	// Initialize services
	claimSvc := claim.NewService(store, collabApi, logger)
	paymentSvc := payment.NewService(store, collabApi, logger)
	insuranceSvc := insurance.NewService(store, collabApi, logger)

	ctx := context.Background()

	// Determine what to run
	runClaims := *all || *syncClaims
	runPayments := *all || *syncPayments
	runInsurance := *all || *syncInsurance

	fmt.Println("Starting manual sync jobs...")
	fmt.Println("==================================")

	// Run sync jobs
	if runClaims {
		fmt.Println("\n[1/3] Syncing new claims...")
		claimSvc.SyncNewClaims(ctx)
		fmt.Println("✓ Claims sync completed")
	}

	if runPayments {
		fmt.Println("\n[2/3] Syncing new payments...")
		paymentSvc.SyncNewPayments(ctx)
		fmt.Println("✓ Payments sync completed")
	}

	if runInsurance {
		fmt.Println("\n[3/3] Syncing insurance checks...")
		insuranceSvc.SyncInsuranceChecks(ctx)
		fmt.Println("✓ Insurance checks sync completed")
	}

	fmt.Println("\n==================================")
	fmt.Println("All requested sync jobs completed!")
}
