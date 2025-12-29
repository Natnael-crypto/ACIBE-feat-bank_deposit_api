package cronjobs

import (
	"log"
	"runtime/debug"
	"time"

	"github.com/aci/backend/internal/core/services/claim"
	"github.com/aci/backend/internal/core/services/insurance"
	"github.com/aci/backend/internal/core/services/payment"
	"github.com/aci/backend/internal/repository"
	"github.com/robfig/cron/v3"
)

const thirtyMinSchedule = "*/15 * * * *"
const everydaySchedule = "0 0 * * *"
const every5Minutes = "*/5 * * * *"
const every10Minutes = "*/10 * * * *"
const everyHour = "0 * * * *"
const dailyAt2AM = "0 2 * * *" // Daily at 2:00 AM UTC

type CronJobs struct {
	cron         *cron.Cron
	claimSvc     *claim.Service
	paymentSvc   *payment.Service
	insuranceSvc *insurance.Service
	store        *repository.Store
}

func NewCronJobs(claimSvc *claim.Service, paymentSvc *payment.Service, insuranceSvc *insurance.Service, store *repository.Store) *CronJobs {
	location, err := time.LoadLocation("UTC")
	if err != nil {
		log.Fatalf("Failed to load location: %v", err)
	}
	newCron := cron.New(cron.WithLocation(location))
	return &CronJobs{
		cron:         newCron,
		claimSvc:     claimSvc,
		paymentSvc:   paymentSvc,
		insuranceSvc: insuranceSvc,
		store:        store,
	}
}

func (c *CronJobs) Start() {
	location, err := time.LoadLocation("UTC")
	if err != nil {
		log.Fatalf("Failed to load location: %v", err)
	}
	newCron := cron.New(cron.WithLocation(location))
	c.cron = newCron
	c.AddClaimsJobs()
	c.AddNewPaymentsJobs()
	c.AddInsuranceChecksJobs()
	c.AddDashboardJobs()
	c.cron.Start()

	log.Println("Cron jobs started")
}

func (c *CronJobs) Stop() {
	if c.cron != nil {
		c.cron.Stop()
		log.Println("Cron jobs stopped")
	}
}

func safeCronJob(name string, job func()) func() {
	return func() {
		log.Println("\nExecuting cron job", name)
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Recovered from panic in cron job %s: %v\n%s", name, r, string(debug.Stack()))
			}
		}()
		job()
	}
}
