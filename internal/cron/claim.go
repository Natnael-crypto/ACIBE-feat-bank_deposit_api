package cronjobs

import (
	"context"
	"log"
)

func (c *CronJobs) AddClaimsJobs() {
	_, err := c.cron.AddFunc(everydaySchedule, safeCronJob("add claims", func() {
		c.claimSvc.SyncNewClaims(context.Background())
	}))
	if err != nil {
		log.Fatalf("Error adding cron job: %v", err)
	}
}
