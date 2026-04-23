package moderation

import (
	"context"
	"strings"

	users "xugeaneeu/pollify/internal/users/core"
)

type Service struct {
	repo   Repository
	polls  PollReader
	hider  PollVisibilityWriter
	clock  Clock
	quorum int
}

type CreateReportInput struct {
	PollID  string
	Reason  string
	Comment string
}

type ReviewReportInput struct {
	ReportID string
	Decision ReviewDecision
	Comment  string
	Actor    users.AuthenticatedUser
}

func NewService(repo Repository, polls PollReader, hider PollVisibilityWriter, clock Clock, quorum int) (*Service, error) {
	if repo == nil {
		return nil, ErrRepositoryUnbound
	}
	if polls == nil {
		return nil, ErrPollReaderUnbound
	}
	if hider == nil {
		return nil, ErrPollWriterUnbound
	}
	if clock == nil {
		return nil, ErrClockUnbound
	}
	if quorum <= 0 {
		quorum = DefaultQuorum
	}

	return &Service{repo: repo, polls: polls, hider: hider, clock: clock, quorum: quorum}, nil
}

func (s *Service) Create(ctx context.Context, actorID string, input CreateReportInput) (Report, error) {
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return Report{}, ValidationError{Field: "actor_id", Message: "must not be empty"}
	}
	if strings.TrimSpace(input.PollID) == "" {
		return Report{}, ValidationError{Field: "poll_id", Message: "must not be empty"}
	}
	if strings.TrimSpace(input.Reason) == "" {
		return Report{}, ValidationError{Field: "reason", Message: "must not be empty"}
	}

	if _, err := s.polls.GetByID(ctx, strings.TrimSpace(input.PollID)); err != nil {
		return Report{}, err
	}

	exists, err := s.repo.HasActiveReport(ctx, strings.TrimSpace(input.PollID), actorID)
	if err != nil {
		return Report{}, err
	}
	if exists {
		return Report{}, ErrReportAlreadyExists
	}

	report := Report{
		PollID:    strings.TrimSpace(input.PollID),
		CreatedBy: actorID,
		Reason:    strings.TrimSpace(input.Reason),
		Comment:   strings.TrimSpace(input.Comment),
		Status:    ReportStatusOpen,
		CreatedAt: s.clock.Now(),
	}

	return s.repo.Create(ctx, report)
}

func (s *Service) Get(ctx context.Context, reportID string) (Report, error) {
	return s.repo.GetByID(ctx, strings.TrimSpace(reportID))
}

func (s *Service) List(ctx context.Context, filter ListFilter) ([]Report, error) {
	return s.repo.List(ctx, filter)
}

func (s *Service) Review(ctx context.Context, input ReviewReportInput) (ReviewOutcome, error) {
	if input.Actor.Role != users.RoleAdmin {
		return ReviewOutcome{}, ErrAdminRequired
	}
	if strings.TrimSpace(input.Actor.UserID) == "" {
		return ReviewOutcome{}, ValidationError{Field: "actor_id", Message: "must not be empty"}
	}
	if input.Decision != ReviewDecisionApprove && input.Decision != ReviewDecisionReject {
		return ReviewOutcome{}, ValidationError{Field: "decision", Message: "must be APPROVE or REJECT"}
	}

	report, err := s.repo.GetByID(ctx, strings.TrimSpace(input.ReportID))
	if err != nil {
		return ReviewOutcome{}, err
	}
	if report.CreatedBy == input.Actor.UserID {
		return ReviewOutcome{}, ErrSelfReviewForbidden
	}
	if report.Status == ReportStatusResolved || report.Status == ReportStatusRejected {
		return ReviewOutcome{}, ErrReportAlreadyResolved
	}
	for _, review := range report.Reviews {
		if review.AdminID == input.Actor.UserID {
			return ReviewOutcome{}, ErrReportReviewAlreadyExists
		}
	}

	review := ReportReview{
		ReportID:  report.ID,
		AdminID:   input.Actor.UserID,
		Decision:  input.Decision,
		Comment:   strings.TrimSpace(input.Comment),
		CreatedAt: s.clock.Now(),
	}

	updated := report
	updated.Reviews = append(append([]ReportReview(nil), report.Reviews...), review)
	quorumState := EvaluateQuorum(updated.Reviews, s.quorum)
	updated.ApprovalCount = quorumState.ApprovalCount
	updated.RejectionCount = quorumState.RejectionCount
	updated.Status = quorumState.Status
	updated.Resolution = quorumState.Resolution

	if quorumState.Reached && quorumState.FinalDecision == ReviewDecisionApprove {
		if err := s.hider.Hide(ctx, updated.PollID); err != nil {
			return ReviewOutcome{}, err
		}
	}

	updated, err = s.repo.SaveReview(ctx, review, updated)
	if err != nil {
		return ReviewOutcome{}, err
	}

	return ReviewOutcome{Review: review, Report: updated}, nil
}
