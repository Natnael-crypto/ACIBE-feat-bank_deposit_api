package main

import (
	"context"
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
	cfg, err := config.NewConfig()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	conn, close, err := repository.OpenDB(cfg.DbUrl)
	if err != nil {
		panic(err)
	}
	defer close()
	store := repository.NewStore(conn)
	// rate, err :=
	// 	store.GetOutPatientRate(context.Background(), "90853", "")
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println(rate)
	// return
	collabApi := collaborate.NewAPI(cfg.Collaborate.Username, cfg.Collaborate.Password, store)
	claimSvc := claim.NewService(store, collabApi, slog.Default())
	claimSvc.SyncNewClaims(context.Background())
	paymentSvc := payment.NewService(store, collabApi, slog.Default())
	paymentSvc.SyncNewPayments(context.Background())
	insuranceSvc := insurance.NewService(store, collabApi, slog.Default())
	insuranceSvc.SyncInsuranceChecks(context.Background())
	return
}
