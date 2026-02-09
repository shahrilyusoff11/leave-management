package cron

import (
	"fmt"
	"leave-management-system/internal/services"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

type CronJobs struct {
	leaveService    *services.LeaveService
	emailService    *services.EmailService
	workflowService *services.WorkflowService
	logger          *zap.Logger
	cron            *cron.Cron
}

func NewCronJobs(leaveService *services.LeaveService,
	emailService *services.EmailService, workflowService *services.WorkflowService, logger *zap.Logger) *CronJobs {
	return &CronJobs{
		leaveService:    leaveService,
		emailService:    emailService,
		workflowService: workflowService,
		logger:          logger,
		cron:            cron.New(cron.WithSeconds()),
	}
}

func (cj *CronJobs) Start() error {
	// Run every day at midnight to check for escalated requests
	_, err := cj.cron.AddFunc("0 0 0 * * *", cj.checkEscalatedRequests)
	if err != nil {
		return fmt.Errorf("failed to add escalated requests job: %w", err)
	}

	// Run every day at 1 AM to send reminder emails
	_, err = cj.cron.AddFunc("0 0 1 * * *", cj.sendReminderEmails)
	if err != nil {
		return fmt.Errorf("failed to add reminder emails job: %w", err)
	}

	// Run on December 31st at 11:59 PM for year-end processing
	_, err = cj.cron.AddFunc("0 59 23 31 12 *", cj.processYearEnd)
	if err != nil {
		return fmt.Errorf("failed to add year-end job: %w", err)
	}

	// Run every hour to check for workflow timeouts
	_, err = cj.cron.AddFunc("0 0 * * * *", cj.processWorkflowTimeouts)
	if err != nil {
		return fmt.Errorf("failed to add workflow timeout job: %w", err)
	}

	cj.cron.Start()
	cj.logger.Info("Cron jobs started")

	return nil
}

func (cj *CronJobs) Stop() {
	cj.cron.Stop()
	cj.logger.Info("Cron jobs stopped")
}

func (cj *CronJobs) checkEscalatedRequests() {
	cj.logger.Info("Checking for escalated leave requests")

	// Get pending requests older than 7 days
	sevenDaysAgo := time.Now().AddDate(0, 0, -7)

	requests, err := cj.leaveService.GetPendingRequestsOlderThan(sevenDaysAgo)
	if err != nil {
		cj.logger.Error("Failed to get pending requests", zap.Error(err))
		return
	}

	for _, request := range requests {
		// Escalate the request
		if err := cj.leaveService.EscalateRequest(request.ID); err != nil {
			cj.logger.Error("Failed to escalate request",
				zap.String("request_id", request.ID.String()),
				zap.Error(err))
			continue
		}

		// Send notification to HR
		cj.emailService.SendEscalationNotification(&request)

		cj.logger.Info("Request escalated",
			zap.String("request_id", request.ID.String()),
			zap.String("user_id", request.UserID.String()))
	}
}

func (cj *CronJobs) sendReminderEmails() {
	cj.logger.Info("Sending reminder emails for pending requests")

	// Get pending requests that need reminder (e.g., pending for 3 days)
	threeDaysAgo := time.Now().AddDate(0, 0, -3)

	requests, err := cj.leaveService.GetPendingRequestsOlderThan(threeDaysAgo)
	if err != nil {
		cj.logger.Error("Failed to get pending requests for reminders", zap.Error(err))
		return
	}

	for _, request := range requests {
		// Send reminder to manager
		cj.emailService.SendReminderNotification(&request)
	}
}

func (cj *CronJobs) processYearEnd() {
	cj.logger.Info("Starting year-end processing")

	if err := cj.leaveService.ProcessYearEndCarryForward(); err != nil {
		cj.logger.Error("Failed to process year-end carry forward", zap.Error(err))
		return
	}

	// Archive old records (older than 7 years)
	sevenYearsAgo := time.Now().AddDate(-7, 0, 0)
	if err := cj.leaveService.ArchiveOldRecords(sevenYearsAgo); err != nil {
		cj.logger.Error("Failed to archive old records", zap.Error(err))
		return
	}

	cj.logger.Info("Year-end processing completed")
}

// Manual trigger functions for testing
func (cj *CronJobs) TriggerEscalationCheck() {
	cj.checkEscalatedRequests()
}

func (cj *CronJobs) TriggerYearEndProcess() {
	cj.processYearEnd()
}

// processWorkflowTimeouts checks for overdue workflow steps and applies timeout actions
func (cj *CronJobs) processWorkflowTimeouts() {
	cj.logger.Info("Processing workflow timeouts")

	if cj.workflowService == nil {
		cj.logger.Warn("Workflow service not available, skipping timeout processing")
		return
	}

	processedIDs, err := cj.workflowService.ProcessTimeouts()
	if err != nil {
		cj.logger.Error("Failed to process workflow timeouts", zap.Error(err))
		return
	}

	if len(processedIDs) > 0 {
		cj.logger.Info("Workflow timeouts processed",
			zap.Int("count", len(processedIDs)),
			zap.Any("request_ids", processedIDs))
	} else {
		cj.logger.Debug("No workflow timeouts to process")
	}
}

// TriggerWorkflowTimeouts allows manual triggering of workflow timeout processing
func (cj *CronJobs) TriggerWorkflowTimeouts() {
	cj.processWorkflowTimeouts()
}
