package cronjobs

import (
	"context"
	"log"
)

func (c *CronJobs) AddNewPaymentsJobs() {
	_, err := c.cron.AddFunc(everydaySchedule, safeCronJob("add new payments", func() {
		c.paymentSvc.SyncNewPayments(context.Background())
	}))
	if err != nil {
		log.Fatalf("Error adding cron job: %v", err)
	}
}
