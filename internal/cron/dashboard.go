package cronjobs

import (
	"context"
	"log"
)

// AddDashboardJobs registers cron jobs for dashboard materialized view refresh
func (c *CronJobs) AddDashboardJobs() {
	// Refresh dashboard materialized view daily at 2 AM UTC
	// This keeps the dashboard data up-to-date with the latest claims and payments
	_, err := c.cron.AddFunc(dailyAt2AM, safeCronJob("refresh-dashboard-materialized-view", func() {
		ctx := context.Background()
		
		// Get connection from store and execute the refresh function
		conn := c.store.GetConn()
		if conn == nil {
			log.Printf("Failed to refresh dashboard materialized view: store connection is nil")
			return
		}
		
		_, err := conn.Exec(ctx, "SELECT collaboratemd.refresh_dashboard_annual_data();")
		if err != nil {
			log.Printf("Failed to refresh dashboard materialized view: %v", err)
			return
		}
		
		log.Println("Dashboard materialized view refreshed successfully")
	}))
	if err != nil {
		log.Fatalf("Error adding dashboard cron job: %v", err)
	}
}

