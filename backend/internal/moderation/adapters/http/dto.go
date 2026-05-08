package http

import (
	"time"

	moderation "xugeaneeu/pollify/internal/moderation/core"
)

type createReportRequest struct {
	Reason  string  `json:"reason"`
	Comment *string `json:"comment,omitempty"`
}

type createReviewRequest struct {
	Decision string  `json:"decision"`
	Comment  *string `json:"comment,omitempty"`
}

type reportReviewDTO struct {
	AdminID   string    `json:"admin_id"`
	Decision  string    `json:"decision"`
	Comment   *string   `json:"comment"`
	CreatedAt time.Time `json:"created_at"`
}

type reportResponse struct {
	ID             string            `json:"id"`
	PollID         string            `json:"poll_id"`
	CreatedBy      string            `json:"created_by"`
	Reason         string            `json:"reason"`
	Comment        *string           `json:"comment"`
	Status         string            `json:"status"`
	CreatedAt      time.Time         `json:"created_at"`
	ApprovalCount  int               `json:"approval_count"`
	RejectionCount int               `json:"rejection_count"`
	Resolution     *string           `json:"resolution"`
	Reviews        []reportReviewDTO `json:"reviews"`
}

type reviewResponse struct {
	Review reportReviewDTO `json:"review"`
	Report reportResponse  `json:"report"`
}

type paginationMetaDTO struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}

type reportsListResponse struct {
	Items      []reportResponse  `json:"items"`
	Pagination paginationMetaDTO `json:"pagination"`
}

func toReportResponse(r moderation.Report) reportResponse {
	resp := reportResponse{
		ID:             r.ID,
		PollID:         r.PollID,
		CreatedBy:      r.CreatedBy,
		Reason:         r.Reason,
		Status:         string(r.Status),
		CreatedAt:      r.CreatedAt.UTC(),
		ApprovalCount:  r.ApprovalCount,
		RejectionCount: r.RejectionCount,
		Reviews:        toReviewDTOs(r.Reviews),
	}
	if r.Comment != "" {
		c := r.Comment
		resp.Comment = &c
	}
	if r.Resolution != "" {
		res := r.Resolution
		resp.Resolution = &res
	}
	return resp
}

func toReviewDTOs(reviews []moderation.ReportReview) []reportReviewDTO {
	out := make([]reportReviewDTO, 0, len(reviews))
	for _, rv := range reviews {
		dto := reportReviewDTO{
			AdminID:   rv.AdminID,
			Decision:  string(rv.Decision),
			CreatedAt: rv.CreatedAt.UTC(),
		}
		if rv.Comment != "" {
			c := rv.Comment
			dto.Comment = &c
		}
		out = append(out, dto)
	}
	return out
}

func toReviewResponse(out moderation.ReviewOutcome) reviewResponse {
	dto := reportReviewDTO{
		AdminID:   out.Review.AdminID,
		Decision:  string(out.Review.Decision),
		CreatedAt: out.Review.CreatedAt.UTC(),
	}
	if out.Review.Comment != "" {
		c := out.Review.Comment
		dto.Comment = &c
	}
	return reviewResponse{Review: dto, Report: toReportResponse(out.Report)}
}
