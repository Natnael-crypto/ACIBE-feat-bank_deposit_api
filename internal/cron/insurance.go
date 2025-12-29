package cronjobs

import (
	"context"
	"log"
)

func (c *CronJobs) AddInsuranceChecksJobs() {
	_, err := c.cron.AddFunc(everydaySchedule, safeCronJob("add insurance checks", func() {
		c.insuranceSvc.SyncInsuranceChecks(context.Background())
	}))
	if err != nil {
		log.Fatalf("Error adding cron job: %v", err)
	}
}
