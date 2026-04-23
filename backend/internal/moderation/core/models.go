package moderation

import "time"

type ReportStatus string

const (
	ReportStatusOpen     ReportStatus = "OPEN"
	ReportStatusInReview ReportStatus = "IN_REVIEW"
	ReportStatusResolved ReportStatus = "RESOLVED"
	ReportStatusRejected ReportStatus = "REJECTED"
)

type ReviewDecision string

const (
	ReviewDecisionApprove ReviewDecision = "APPROVE"
	ReviewDecisionReject  ReviewDecision = "REJECT"
)

type ReportReview struct {
	ReportID  string
	AdminID   string
	Decision  ReviewDecision
	Comment   string
	CreatedAt time.Time
}

type Report struct {
	ID             string
	PollID         string
	CreatedBy      string
	Reason         string
	Comment        string
	Status         ReportStatus
	CreatedAt      time.Time
	ApprovalCount  int
	RejectionCount int
	Resolution     string
	Reviews        []ReportReview
}

type ListFilter struct {
	Status    *ReportStatus
	PollID    string
	CreatedBy string
	Page      int
	Limit     int
	Sort      string
}

type ReviewOutcome struct {
	Review ReportReview
	Report Report
}
