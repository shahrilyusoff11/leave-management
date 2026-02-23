package services

import (
	"fmt"
	"leave-management-system/internal/models"
	"strings"

	"gopkg.in/gomail.v2"
)

type EmailService struct {
	dialer *gomail.Dialer
	from   string
}

func NewEmailService(host string, port int, username, password, from string) *EmailService {
	dialer := gomail.NewDialer(host, port, username, password)
	return &EmailService{
		dialer: dialer,
		from:   from,
	}
}

func (es *EmailService) SendLeaveRequestNotification(request *models.LeaveRequest, manager *models.User) error {
	subject := fmt.Sprintf("New Leave Request from %s %s",
		request.User.FirstName, request.User.LastName)

	body := fmt.Sprintf(`
	Dear %s %s,
	
	A new leave request has been submitted for your approval:
	
	Employee: %s %s
	Leave Type: %s
	Dates: %s to %s
	Duration: %.1f days
	Reason: %s
	
	Please log in to the Leave Management System to review and take action.
	
	Regards,
	Leave Management System
	`,
		manager.FirstName, manager.LastName,
		request.User.FirstName, request.User.LastName,
		strings.Title(string(request.LeaveType)),
		request.StartDate.Format("January 2, 2006"),
		request.EndDate.Format("January 2, 2006"),
		request.DurationDays,
		request.Reason)

	return es.sendEmail(manager.Email, subject, body)
}

func (es *EmailService) SendApprovalNotification(request *models.LeaveRequest) error {
	subject := "Your Leave Request has been Approved"

	approverName := "System"
	if request.Approver != nil {
		approverName = fmt.Sprintf("%s %s", request.Approver.FirstName, request.Approver.LastName)
	}

	body := fmt.Sprintf(`
	Dear %s %s,
	
	Your leave request has been approved:
	
	Leave Type: %s
	Dates: %s to %s
	Duration: %.1f days
	Approved by: %s
	Approval Date: %s
	
	Regards,
	Leave Management System
	`,
		request.User.FirstName, request.User.LastName,
		strings.Title(string(request.LeaveType)),
		request.StartDate.Format("January 2, 2006"),
		request.EndDate.Format("January 2, 2006"),
		request.DurationDays,
		approverName,
		request.ApprovedAt.Format("January 2, 2006"))

	return es.sendEmail(request.User.Email, subject, body)
}

func (es *EmailService) SendEscalationNotification(request *models.LeaveRequest) error {
	subject := "Leave Request Escalated - Action Required"

	managerName := "System / No Manager Assigned"
	if request.User.Manager != nil {
		managerName = fmt.Sprintf("%s %s", request.User.Manager.FirstName, request.User.Manager.LastName)
	}

	body := fmt.Sprintf(`
	Attention HR Team,
	
	A leave request has been escalated due to no response from the manager:
	
	Employee: %s %s
	Manager: %s
	Leave Type: %s
	Dates: %s to %s
	Duration: %.1f days
	Submitted: %s
	
	Please log in to the Leave Management System to review and take action.
	
	Regards,
	Leave Management System
	`,
		request.User.FirstName, request.User.LastName,
		managerName,
		strings.Title(string(request.LeaveType)),
		request.StartDate.Format("January 2, 2006"),
		request.EndDate.Format("January 2, 2006"),
		request.DurationDays,
		request.CreatedAt.Format("January 2, 2006"))

	// Send to HR team (would need to fetch HR emails)
	// For now, using a placeholder
	return es.sendEmail("hr@company.com", subject, body)
}

func (es *EmailService) SendReminderNotification(request *models.LeaveRequest) error {
	subject := fmt.Sprintf("Reminder: Pending Leave Request for %s %s",
		request.User.FirstName, request.User.LastName)

	body := fmt.Sprintf(`
	Dear Manager,
	
	This is a reminder that you have a pending leave request waiting for your approval:
	
	Employee: %s %s
	Leave Type: %s
	Dates: %s to %s
	Duration: %.1f days
	
	Please log in to the Leave Management System to take action.
	
	Regards,
	Leave Management System
	`,
		request.User.FirstName, request.User.LastName,
		request.LeaveType, // strings.Title might be better but just using raw for now
		request.StartDate.Format("January 2, 2006"),
		request.EndDate.Format("January 2, 2006"),
		request.DurationDays)

	// Assuming manager email is available in request or needs fetching.
	// The request struct in cron_jobs might not have Manager loaded unless Preload was used.
	// In cron_jobs.go checkEscalatedRequests usage, Preload is used in GetPendingRequestsOlderThan?
	// I added Preload in leave_service_methods.go: GetPendingRequestsOlderThan DOES Preload("User").Preload("User.Manager").
	// But SendReminderNotification uses request.User.Manager.Email?
	// Note: in previous cron_jobs Code, usage was just passing request.

	// If User.Manager is nil, we should handle it.
	targetEmail := "manager@company.com"
	if request.User.Manager != nil {
		targetEmail = request.User.Manager.Email
	} else if request.User.ManagerID != nil {
		// ManagerID exists but struct not loaded? Protocol was Preload("User.Manager").
		// If Preload worked, Manager should be non-nil.
	}

	return es.sendEmail(targetEmail, subject, body)
}

func (es *EmailService) sendEmail(to, subject, body string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", es.from)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", body)

	return es.dialer.DialAndSend(m)
}

func (es *EmailService) SendRejectionNotification(request *models.LeaveRequest) error {
	subject := "Update on your Leave Request"

	body := fmt.Sprintf(`
	Dear %s %s,
	
	Your leave request has been rejected.
	
	Leave Type: %s
	Dates: %s to %s
	Reason for Rejection: %s
	
	Please contact your manager or HR for more details found in the system.
	
	Regards,
	Leave Management System
	`,
		request.User.FirstName, request.User.LastName,
		strings.Title(string(request.LeaveType)),
		request.StartDate.Format("January 2, 2006"),
		request.EndDate.Format("January 2, 2006"),
		request.RejectionReason)

	return es.sendEmail(request.User.Email, subject, body)
}

func (es *EmailService) SendActionRequiredEmail(user models.User, request *models.LeaveRequest, stepName string) error {
	subject := fmt.Sprintf("Action Required: Leave Request from %s %s",
		request.User.FirstName, request.User.LastName)

	body := fmt.Sprintf(`
	Dear %s %s,
	
	A leave request requires your attention (%s):
	
	Employee: %s %s
	Leave Type: %s
	Dates: %s to %s
	Duration: %.1f days
	Reason: %s
	
	Please log in to the Leave Management System to review and take action.
	
	Regards,
	Leave Management System
	`,
		user.FirstName, user.LastName,
		stepName,
		request.User.FirstName, request.User.LastName,
		strings.Title(string(request.LeaveType)),
		request.StartDate.Format("January 2, 2006"),
		request.EndDate.Format("January 2, 2006"),
		request.DurationDays,
		request.Reason)

	return es.sendEmail(user.Email, subject, body)
}
