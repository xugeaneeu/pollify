package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"xugeaneeu/pollify/internal/pkg/apierror"
	"xugeaneeu/pollify/internal/platform/auth"
	polls "xugeaneeu/pollify/internal/polls/core"
)

type Service interface {
	Create(ctx context.Context, input polls.CreatePollInput) (polls.Poll, error)
	Get(ctx context.Context, pollID string) (polls.Poll, error)
	List(ctx context.Context, filter polls.ListFilter) ([]polls.Poll, error)
	Update(ctx context.Context, actorID, pollID string, input polls.UpdatePollInput) (polls.Poll, error)
}

type ParticipationStats interface {
	Count(ctx context.Context, pollID string) (int, error)
	HasVoted(ctx context.Context, pollID, userID string) (bool, error)
}

type Counter interface {
	Count(ctx context.Context, filter polls.ListFilter) (int, error)
}

type Clock interface {
	Now() time.Time
}

type Handler struct {
	service Service
	stats   ParticipationStats
	counter Counter
	clock   Clock
}

func NewHandler(service Service, stats ParticipationStats, counter Counter, clock Clock) *Handler {
	return &Handler{service: service, stats: stats, counter: counter, clock: clock}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Get("/{pollId}", h.get)
	r.Patch("/{pollId}", h.update)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	actor, ok := auth.AuthenticatedUserFromContext(r.Context())
	if !ok {
		apierror.Write(w, apierror.New(http.StatusUnauthorized, apierror.CodeUnauthorized, "authentication required"))
		return
	}

	filter, err := parseListFilter(r, actor.UserID)
	if err != nil {
		apierror.Write(w, err)
		return
	}

	items, listErr := h.service.List(r.Context(), filter)
	if listErr != nil {
		apierror.Write(w, mapError(listErr))
		return
	}

	total, countErr := h.counter.Count(r.Context(), filter)
	if countErr != nil {
		apierror.Write(w, mapError(countErr))
		return
	}

	now := h.clock.Now()
	summaries := make([]pollSummaryDTO, 0, len(items))
	for _, poll := range items {
		summary, err := h.participationFor(r.Context(), poll.ID, actor.UserID)
		if err != nil {
			apierror.Write(w, mapError(err))
			return
		}
		summaries = append(summaries, toSummaryDTO(poll, poll.Status(now), summary))
	}

	totalPages := 0
	if filter.Limit > 0 {
		totalPages = (total + filter.Limit - 1) / filter.Limit
	}
	apierror.WriteJSON(w, http.StatusOK, pollsListResponseDTO{
		Items: summaries,
		Pagination: paginationMetaDTO{
			Page:       maxInt(filter.Page, 1),
			Limit:      filter.Limit,
			TotalItems: total,
			TotalPages: totalPages,
		},
	})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	actor, ok := auth.AuthenticatedUserFromContext(r.Context())
	if !ok {
		apierror.Write(w, apierror.New(http.StatusUnauthorized, apierror.CodeUnauthorized, "authentication required"))
		return
	}

	var req createPollRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		apierror.Write(w, err)
		return
	}

	options := make([]polls.CreatePollOptionInput, 0, len(req.Options))
	for _, opt := range req.Options {
		options = append(options, polls.CreatePollOptionInput{Text: opt.Text})
	}

	settings := polls.PollSettings{
		IsAnonymous:       req.Settings.IsAnonymous,
		IsMultipleChoice:  req.Settings.IsMultipleChoice,
		AllowCustomAnswer: req.Settings.AllowCustomAnswer,
		StartAt:           req.Settings.StartAt,
		EndAt:             req.Settings.EndAt,
	}
	if req.Settings.MaxChoices != nil {
		settings.MaxChoices = *req.Settings.MaxChoices
	}

	description := ""
	if req.Description != nil {
		description = *req.Description
	}

	poll, err := h.service.Create(r.Context(), polls.CreatePollInput{
		CreatedBy:   actor.UserID,
		Title:       req.Title,
		Description: description,
		Question:    req.Question,
		Options:     options,
		Settings:    settings,
	})
	if err != nil {
		apierror.Write(w, mapError(err))
		return
	}

	summary := participationSummaryDTO{}
	apierror.WriteJSON(w, http.StatusCreated, toDetailsDTO(poll, poll.Status(h.clock.Now()), summary))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	actor, ok := auth.AuthenticatedUserFromContext(r.Context())
	if !ok {
		apierror.Write(w, apierror.New(http.StatusUnauthorized, apierror.CodeUnauthorized, "authentication required"))
		return
	}
	pollID := chi.URLParam(r, "pollId")
	poll, err := h.service.Get(r.Context(), pollID)
	if err != nil {
		apierror.Write(w, mapError(err))
		return
	}
	summary, err := h.participationFor(r.Context(), poll.ID, actor.UserID)
	if err != nil {
		apierror.Write(w, mapError(err))
		return
	}
	apierror.WriteJSON(w, http.StatusOK, toDetailsDTO(poll, poll.Status(h.clock.Now()), summary))
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	actor, ok := auth.AuthenticatedUserFromContext(r.Context())
	if !ok {
		apierror.Write(w, apierror.New(http.StatusUnauthorized, apierror.CodeUnauthorized, "authentication required"))
		return
	}
	pollID := chi.URLParam(r, "pollId")

	var req updatePollRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		apierror.Write(w, err)
		return
	}

	poll, err := h.service.Update(r.Context(), actor.UserID, pollID, polls.UpdatePollInput{
		Title:       req.Title,
		Description: req.Description,
		StartAt:     req.StartAt,
		EndAt:       req.EndAt,
	})
	if err != nil {
		apierror.Write(w, mapError(err))
		return
	}
	summary, err := h.participationFor(r.Context(), poll.ID, actor.UserID)
	if err != nil {
		apierror.Write(w, mapError(err))
		return
	}
	apierror.WriteJSON(w, http.StatusOK, toDetailsDTO(poll, poll.Status(h.clock.Now()), summary))
}

func (h *Handler) participationFor(ctx context.Context, pollID, userID string) (participationSummaryDTO, error) {
	count, err := h.stats.Count(ctx, pollID)
	if err != nil {
		return participationSummaryDTO{}, err
	}
	hasVoted := false
	if userID != "" {
		v, err := h.stats.HasVoted(ctx, pollID, userID)
		if err != nil {
			return participationSummaryDTO{}, err
		}
		hasVoted = v
	}
	return participationSummaryDTO{ParticipantsCount: count, HasVoted: hasVoted}, nil
}

func parseListFilter(r *http.Request, actorID string) (polls.ListFilter, *apierror.Error) {
	q := r.URL.Query()
	filter := polls.ListFilter{ActorID: actorID}

	if v := q.Get("status"); v != "" {
		switch polls.Status(v) {
		case polls.StatusScheduled, polls.StatusActive, polls.StatusCompleted, polls.StatusHidden:
			s := polls.Status(v)
			filter.Status = &s
		default:
			return polls.ListFilter{}, apierror.WithField(http.StatusUnprocessableEntity, apierror.CodeValidationError, "status", "unknown status value")
		}
	}
	if v := q.Get("creator_id"); v != "" {
		filter.CreatorID = strings.TrimSpace(v)
	}
	if v, ok, err := boolQuery(q, "is_anonymous"); err != nil {
		return polls.ListFilter{}, err
	} else if ok {
		filter.IsAnonymous = &v
	}
	if v, ok, err := boolQuery(q, "is_multiple_choice"); err != nil {
		return polls.ListFilter{}, err
	} else if ok {
		filter.IsMultipleChoice = &v
	}
	if v, ok, err := boolQuery(q, "allow_custom_answer"); err != nil {
		return polls.ListFilter{}, err
	} else if ok {
		filter.AllowCustomAnswer = &v
	}
	if v, ok, err := boolQuery(q, "available_for_voting"); err != nil {
		return polls.ListFilter{}, err
	} else if ok {
		filter.AvailableForVoting = &v
	}

	page, limit, err := parsePagination(q)
	if err != nil {
		return polls.ListFilter{}, err
	}
	filter.Page = page
	filter.Limit = limit
	filter.Sort = strings.TrimSpace(q.Get("sort"))

	return filter, nil
}

func parsePagination(q map[string][]string) (int, int, *apierror.Error) {
	page := 1
	limit := 20
	if v := getQuery(q, "page"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return 0, 0, apierror.WithField(http.StatusUnprocessableEntity, apierror.CodeValidationError, "page", "must be a positive integer")
		}
		page = n
	}
	if v := getQuery(q, "limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 100 {
			return 0, 0, apierror.WithField(http.StatusUnprocessableEntity, apierror.CodeValidationError, "limit", "must be between 1 and 100")
		}
		limit = n
	}
	return page, limit, nil
}

func boolQuery(q map[string][]string, key string) (bool, bool, *apierror.Error) {
	raw := getQuery(q, key)
	if raw == "" {
		return false, false, nil
	}
	switch strings.ToLower(raw) {
	case "true", "1":
		return true, true, nil
	case "false", "0":
		return false, true, nil
	}
	return false, false, apierror.WithField(http.StatusUnprocessableEntity, apierror.CodeValidationError, key, "must be true or false")
}

func getQuery(q map[string][]string, key string) string {
	if vs, ok := q[key]; ok && len(vs) > 0 {
		return strings.TrimSpace(vs[0])
	}
	return ""
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
