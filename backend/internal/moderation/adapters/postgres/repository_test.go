package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	moderationpg "xugeaneeu/pollify/internal/moderation/adapters/postgres"
	moderation "xugeaneeu/pollify/internal/moderation/core"
	postgrestest "xugeaneeu/pollify/internal/platform/postgrestest"
)

func TestModerationRepositoryCreateAndReview(t *testing.T) {
	pool := postgrestest.OpenTestDB(t)
	postgrestest.SeedUser(t, pool, "11111111-1111-1111-1111-111111111111", "owner@example.com", "USER")
	postgrestest.SeedUser(t, pool, "99999999-9999-9999-9999-999999999999", "admin@example.com", "ADMIN")
	postgrestest.SeedPoll(t, pool, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "11111111-1111-1111-1111-111111111111", false)

	repo := moderationpg.NewRepository(pool)
	report, err := repo.Create(context.Background(), moderation.Report{
		PollID:    "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		CreatedBy: "11111111-1111-1111-1111-111111111111",
		Reason:    "spam",
		Comment:   "comment",
		Status:    moderation.ReportStatusOpen,
		CreatedAt: time.Now().UTC().Truncate(time.Second),
	})
	if err != nil {
		t.Fatalf("create report: %v", err)
	}

	loaded, err := repo.GetByID(context.Background(), report.ID)
	if err != nil {
		t.Fatalf("get report: %v", err)
	}
	if loaded.Reason != "spam" {
		t.Fatalf("unexpected reason: %q", loaded.Reason)
	}

	updated, err := repo.SaveReview(context.Background(), moderation.ReportReview{
		ReportID:  report.ID,
		AdminID:   "99999999-9999-9999-9999-999999999999",
		Decision:  moderation.ReviewDecisionApprove,
		Comment:   "looks valid",
		CreatedAt: time.Now().UTC().Truncate(time.Second),
	}, moderation.Report{
		ID:             report.ID,
		PollID:         report.PollID,
		CreatedBy:      report.CreatedBy,
		Reason:         report.Reason,
		Comment:        report.Comment,
		Status:         moderation.ReportStatusInReview,
		CreatedAt:      report.CreatedAt,
		ApprovalCount:  1,
		RejectionCount: 0,
		Resolution:     "",
	})
	if err != nil {
		t.Fatalf("save review: %v", err)
	}
	if len(updated.Reviews) != 1 {
		t.Fatalf("expected 1 review, got %d", len(updated.Reviews))
	}

	_, err = repo.SaveReview(context.Background(), moderation.ReportReview{
		ReportID:  report.ID,
		AdminID:   "99999999-9999-9999-9999-999999999999",
		Decision:  moderation.ReviewDecisionApprove,
		Comment:   "duplicate",
		CreatedAt: time.Now().UTC().Truncate(time.Second),
	}, updated)
	if !errors.Is(err, moderation.ErrReportReviewAlreadyExists) {
		t.Fatalf("expected ErrReportReviewAlreadyExists, got %v", err)
	}
}
