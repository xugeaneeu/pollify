package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	moderation "xugeaneeu/pollify/internal/moderation/core"
	"xugeaneeu/pollify/internal/pkg/apierror"
	"xugeaneeu/pollify/internal/platform/auth"
)

type Service interface {
	Create(ctx context.Context, actorID string, input moderation.CreateReportInput) (moderation.Report, error)
	Get(ctx context.Context, reportID string) (moderation.Report, error)
	List(ctx context.Context, filter moderation.ListFilter) ([]moderation.Report, error)
	Review(ctx context.Context, input moderation.ReviewReportInput) (moderation.ReviewOutcome, error)
}

type Counter interface {
	Count(ctx context.Context, filter moderation.ListFilter) (int, error)
}

type Handler struct {
	service Service
	counter Counter
}

func NewHandler(service Service, counter Counter) *Handler {
	return &Handler{service: service, counter: counter}
}

func (h *Handler) CreateReport(w http.ResponseWriter, r *http.Request) {
	actor, ok := auth.AuthenticatedUserFromContext(r.Context())
	if !ok {
		apierror.Write(w, apierror.New(http.StatusUnauthorized, apierror.CodeUnauthorized, "authentication required"))
		return
	}

	pollID := chi.URLParam(r, "pollId")
	var req createReportRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		apierror.Write(w, err)
		return
	}
	comment := ""
	if req.Comment != nil {
		comment = *req.Comment
	}

	report, err := h.service.Create(r.Context(), actor.UserID, moderation.CreateReportInput{
		PollID:  pollID,
		Reason:  req.Reason,
		Comment: comment,
	})
	if err != nil {
		apierror.Write(w, mapError(err))
		return
	}

	apierror.WriteJSON(w, http.StatusCreated, toReportResponse(report))
}

func (h *Handler) ListReportsByPoll(w http.ResponseWriter, r *http.Request) {
	pollID := chi.URLParam(r, "pollId")
	filter, errResp := h.parseListFilter(r)
	if errResp != nil {
		apierror.Write(w, errResp)
		return
	}
	filter.PollID = pollID
	h.respondWithList(w, r.Context(), filter)
}

func (h *Handler) ListReports(w http.ResponseWriter, r *http.Request) {
	filter, errResp := h.parseListFilter(r)
	if errResp != nil {
		apierror.Write(w, errResp)
		return
	}
	if v := strings.TrimSpace(r.URL.Query().Get("poll_id")); v != "" {
		filter.PollID = v
	}
	if v := strings.TrimSpace(r.URL.Query().Get("created_by")); v != "" {
		filter.CreatedBy = v
	}
	h.respondWithList(w, r.Context(), filter)
}

func (h *Handler) GetReport(w http.ResponseWriter, r *http.Request) {
	reportID := chi.URLParam(r, "reportId")
	report, err := h.service.Get(r.Context(), reportID)
	if err != nil {
		apierror.Write(w, mapError(err))
		return
	}
	apierror.WriteJSON(w, http.StatusOK, toReportResponse(report))
}

func (h *Handler) SubmitReview(w http.ResponseWriter, r *http.Request) {
	actor, ok := auth.AuthenticatedUserFromContext(r.Context())
	if !ok {
		apierror.Write(w, apierror.New(http.StatusUnauthorized, apierror.CodeUnauthorized, "authentication required"))
		return
	}

	reportID := chi.URLParam(r, "reportId")
	var req createReviewRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		apierror.Write(w, err)
		return
	}
	comment := ""
	if req.Comment != nil {
		comment = *req.Comment
	}

	outcome, err := h.service.Review(r.Context(), moderation.ReviewReportInput{
		ReportID: reportID,
		Decision: moderation.ReviewDecision(strings.ToUpper(strings.TrimSpace(req.Decision))),
		Comment:  comment,
		Actor:    actor,
	})
	if err != nil {
		apierror.Write(w, mapError(err))
		return
	}

	apierror.WriteJSON(w, http.StatusCreated, toReviewResponse(outcome))
}

func (h *Handler) respondWithList(w http.ResponseWriter, ctx context.Context, filter moderation.ListFilter) {
	items, err := h.service.List(ctx, filter)
	if err != nil {
		apierror.Write(w, mapError(err))
		return
	}
	total, err := h.counter.Count(ctx, filter)
	if err != nil {
		apierror.Write(w, mapError(err))
		return
	}
	out := make([]reportResponse, 0, len(items))
	for _, rep := range items {
		out = append(out, toReportResponse(rep))
	}
	totalPages := 0
	if filter.Limit > 0 {
		totalPages = (total + filter.Limit - 1) / filter.Limit
	}
	apierror.WriteJSON(w, http.StatusOK, reportsListResponse{
		Items: out,
		Pagination: paginationMetaDTO{
			Page:       maxInt(filter.Page, 1),
			Limit:      filter.Limit,
			TotalItems: total,
			TotalPages: totalPages,
		},
	})
}

func (h *Handler) parseListFilter(r *http.Request) (moderation.ListFilter, *apierror.Error) {
	q := r.URL.Query()
	filter := moderation.ListFilter{Page: 1, Limit: 20}

	if v := strings.TrimSpace(q.Get("status")); v != "" {
		s := moderation.ReportStatus(strings.ToUpper(v))
		switch s {
		case moderation.ReportStatusOpen, moderation.ReportStatusInReview, moderation.ReportStatusResolved, moderation.ReportStatusRejected:
			filter.Status = &s
		default:
			return moderation.ListFilter{}, apierror.WithField(http.StatusUnprocessableEntity, apierror.CodeValidationError, "status", "unknown status value")
		}
	}
	if v := strings.TrimSpace(q.Get("page")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return moderation.ListFilter{}, apierror.WithField(http.StatusUnprocessableEntity, apierror.CodeValidationError, "page", "must be a positive integer")
		}
		filter.Page = n
	}
	if v := strings.TrimSpace(q.Get("limit")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 100 {
			return moderation.ListFilter{}, apierror.WithField(http.StatusUnprocessableEntity, apierror.CodeValidationError, "limit", "must be between 1 and 100")
		}
		filter.Limit = n
	}
	filter.Sort = strings.TrimSpace(q.Get("sort"))
	return filter, nil
}

func decodeJSON(body io.ReadCloser, dst any) *apierror.Error {
	dec := json.NewDecoder(body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return apierror.New(http.StatusUnprocessableEntity, apierror.CodeValidationError, "request body is required")
		}
		return apierror.New(http.StatusUnprocessableEntity, apierror.CodeValidationError, "invalid request body")
	}
	return nil
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
