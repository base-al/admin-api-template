package jobs

import (
	"base/core/logger"
	"base/core/scheduler"

	"gorm.io/gorm"
)

// SetupScheduler registers all scheduled jobs with the cron scheduler
func SetupScheduler(db *gorm.DB, logger logger.Logger) *scheduler.CronScheduler {
	cronScheduler := scheduler.NewCronScheduler(logger)

	// Order Services
	// TODO: Temporarily disabled until we resolve dependency injection for jobs
	// orderService := orders.NewOrderService(db, emitter, storage, logger, invoiceService)

	// Initialize jobs
	// orderRenewalJob := NewOrderRenewalJob(db, logger, orderService)

	// Register Order Renewal Job - runs daily at 9:00 AM
	// TODO: Re-enable when dependency injection is resolved
	/*
		cronTask := &scheduler.CronTask{
			Name:        "order_renewal_check",
			Description: "Check for orders expiring in 5 days and create pending renewal orders",
			CronExpr:    "0 9 * * *", // Daily at 9:00 AM
			Handler: func(ctx context.Context) error {
				return orderRenewalJob.Execute(ctx)
			},
			Enabled: true,
		}

		err := cronScheduler.RegisterTask(cronTask)
		if err != nil {
			logger.Error("failed to register order renewal job")
		} else {
			logger.Info("registered order renewal job")
		}
	*/

	return cronScheduler
}
