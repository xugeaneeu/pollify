package postgres

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	moderation "xugeaneeu/pollify/internal/moderation/core"
	platformpg "xugeaneeu/pollify/internal/platform/postgres"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, report moderation.Report) (moderation.Report, error) {
	created, err := scanReportRow(r.db.QueryRow(ctx, `
		INSERT INTO reports (poll_id, created_by, reason, comment, status, approval_count, rejection_count, resolution, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, poll_id, created_by, reason, comment, status, approval_count, rejection_count, resolution, created_at
	`, report.PollID, report.CreatedBy, report.Reason, report.Comment, string(report.Status), report.ApprovalCount, report.RejectionCount, report.Resolution, report.CreatedAt))
	if err != nil {
		return moderation.Report{}, err
	}

	return created, nil
}

func (r *Repository) GetByID(ctx context.Context, reportID string) (moderation.Report, error) {
	report, err := scanReportRow(r.db.QueryRow(ctx, `
		SELECT id, poll_id, created_by, reason, comment, status, approval_count, rejection_count, resolution, created_at
		FROM reports
		WHERE id = $1
	`, strings.TrimSpace(reportID)))
	if err != nil {
		return moderation.Report{}, err
	}

	report.Reviews, err = listReviews(ctx, r.db, report.ID)
	if err != nil {
		return moderation.Report{}, err
	}

	return report, nil
}

func (r *Repository) List(ctx context.Context, filter moderation.ListFilter) ([]moderation.Report, error) {
	args := []any{}
	query := strings.Builder{}
	query.WriteString(`
		SELECT id, poll_id, created_by, reason, comment, status, approval_count, rejection_count, resolution, created_at
		FROM reports
		WHERE 1=1
	`)

	if filter.Status != nil {
		args = append(args, string(*filter.Status))
		query.WriteString(` AND status = $` + strconv.Itoa(len(args)))
	}
	if filter.PollID != "" {
		args = append(args, strings.TrimSpace(filter.PollID))
		query.WriteString(` AND poll_id = $` + strconv.Itoa(len(args)))
	}
	if filter.CreatedBy != "" {
		args = append(args, strings.TrimSpace(filter.CreatedBy))
		query.WriteString(` AND created_by = $` + strconv.Itoa(len(args)))
	}

	switch filter.Sort {
	case "created_at_asc":
		query.WriteString(` ORDER BY created_at ASC`)
	default:
		query.WriteString(` ORDER BY created_at DESC`)
	}

	limit := maxValue(filter.Limit, 20)
	offset := 0
	if filter.Page > 1 {
		offset = (filter.Page - 1) * limit
	}
	args = append(args, limit, offset)
	query.WriteString(` LIMIT $` + strconv.Itoa(len(args)-1) + ` OFFSET $` + strconv.Itoa(len(args)))

	rows, err := r.db.Query(ctx, query.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]moderation.Report, 0)
	reportIDs := make([]string, 0)
	for rows.Next() {
		report, err := scanReportRows(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, report)
		reportIDs = append(reportIDs, report.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return items, nil
	}

	reviewsByReport, err := listReviewsByReportIDs(ctx, r.db, reportIDs)
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i].Reviews = reviewsByReport[items[i].ID]
	}

	return items, nil
}

func (r *Repository) Count(ctx context.Context, filter moderation.ListFilter) (int, error) {
	args := []any{}
	query := strings.Builder{}
	query.WriteString(`SELECT COUNT(*) FROM reports WHERE 1=1`)

	if filter.Status != nil {
		args = append(args, string(*filter.Status))
		query.WriteString(` AND status = $` + strconv.Itoa(len(args)))
	}
	if filter.PollID != "" {
		args = append(args, strings.TrimSpace(filter.PollID))
		query.WriteString(` AND poll_id = $` + strconv.Itoa(len(args)))
	}
	if filter.CreatedBy != "" {
		args = append(args, strings.TrimSpace(filter.CreatedBy))
		query.WriteString(` AND created_by = $` + strconv.Itoa(len(args)))
	}

	var count int
	if err := r.db.QueryRow(ctx, query.String(), args...).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (r *Repository) HasActiveReport(ctx context.Context, pollID string, userID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM reports
			WHERE poll_id = $1
			  AND created_by = $2
			  AND status IN ('OPEN', 'IN_REVIEW')
		)
	`, strings.TrimSpace(pollID), strings.TrimSpace(userID)).Scan(&exists)
	return exists, err
}

func (r *Repository) SaveReview(ctx context.Context, review moderation.ReportReview, report moderation.Report) (moderation.Report, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return moderation.Report{}, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		INSERT INTO report_reviews (report_id, admin_id, decision, comment, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, review.ReportID, review.AdminID, string(review.Decision), review.Comment, review.CreatedAt); err != nil {
		if code := platformpg.ErrorCode(err); code == "23505" {
			return moderation.Report{}, moderation.ErrReportReviewAlreadyExists
		}
		message := platformpg.ErrorMessage(err)
		if strings.Contains(message, "self review") {
			return moderation.Report{}, moderation.ErrSelfReviewForbidden
		}
		return moderation.Report{}, err
	}

	updated, err := scanReportRow(tx.QueryRow(ctx, `
		UPDATE reports
		SET status = $2,
			approval_count = $3,
			rejection_count = $4,
			resolution = $5
		WHERE id = $1
		RETURNING id, poll_id, created_by, reason, comment, status, approval_count, rejection_count, resolution, created_at
	`, report.ID, string(report.Status), report.ApprovalCount, report.RejectionCount, report.Resolution))
	if err != nil {
		return moderation.Report{}, err
	}

	updated.Reviews, err = listReviews(ctx, tx, updated.ID)
	if err != nil {
		return moderation.Report{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return moderation.Report{}, err
	}

	return updated, nil
}

type queryer interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type queryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func scanReportRow(row pgx.Row) (moderation.Report, error) {
	var report moderation.Report
	var status string
	err := row.Scan(&report.ID, &report.PollID, &report.CreatedBy, &report.Reason, &report.Comment, &status, &report.ApprovalCount, &report.RejectionCount, &report.Resolution, &report.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return moderation.Report{}, moderation.ErrReportNotFound
		}
		return moderation.Report{}, err
	}

	report.Status = moderation.ReportStatus(status)
	return report, nil
}

func scanReportRows(rows pgx.Rows) (moderation.Report, error) {
	var report moderation.Report
	var status string
	err := rows.Scan(&report.ID, &report.PollID, &report.CreatedBy, &report.Reason, &report.Comment, &status, &report.ApprovalCount, &report.RejectionCount, &report.Resolution, &report.CreatedAt)
	report.Status = moderation.ReportStatus(status)
	return report, err
}

func listReviews(ctx context.Context, db queryer, reportID string) ([]moderation.ReportReview, error) {
	items, err := listReviewsByReportIDs(ctx, db, []string{reportID})
	if err != nil {
		return nil, err
	}

	return items[reportID], nil
}

func listReviewsByReportIDs(ctx context.Context, db queryer, reportIDs []string) (map[string][]moderation.ReportReview, error) {
	rows, err := db.Query(ctx, `
		SELECT report_id, admin_id, decision, comment, created_at
		FROM report_reviews
		WHERE report_id = ANY($1::uuid[])
		ORDER BY created_at ASC, admin_id ASC
	`, reportIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]moderation.ReportReview, len(reportIDs))
	for _, reportID := range reportIDs {
		result[reportID] = []moderation.ReportReview{}
	}
	for rows.Next() {
		var review moderation.ReportReview
		var decision string
		if err := rows.Scan(&review.ReportID, &review.AdminID, &decision, &review.Comment, &review.CreatedAt); err != nil {
			return nil, err
		}
		review.Decision = moderation.ReviewDecision(decision)
		result[review.ReportID] = append(result[review.ReportID], review)
	}

	return result, rows.Err()
}

func maxValue(value int, fallback int) int {
	if value <= 0 {
		return fallback
	}

	return value
}
