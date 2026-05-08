package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"xugeaneeu/pollify/internal/pkg/apierror"
	"xugeaneeu/pollify/internal/platform/auth"
	voting "xugeaneeu/pollify/internal/voting/core"
)

type Service interface {
	SubmitVote(ctx context.Context, actorID string, input voting.SubmitVoteInput) (voting.VoteReceipt, error)
	Results(ctx context.Context, actorID string, pollID string, includeVoters bool) (voting.PollResults, error)
}

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/{pollId}/votes", h.submitVote)
	r.Get("/{pollId}/results", h.getResults)
}

func (h *Handler) submitVote(w http.ResponseWriter, r *http.Request) {
	actor, ok := auth.AuthenticatedUserFromContext(r.Context())
	if !ok {
		apierror.Write(w, apierror.New(http.StatusUnauthorized, apierror.CodeUnauthorized, "authentication required"))
		return
	}
	pollID := chi.URLParam(r, "pollId")

	var req createVoteRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		apierror.Write(w, err)
		return
	}

	receipt, err := h.service.SubmitVote(r.Context(), actor.UserID, voting.SubmitVoteInput{
		PollID:     pollID,
		OptionIDs:  req.OptionIDs,
		CustomText: req.CustomText,
	})
	if err != nil {
		apierror.Write(w, mapError(err))
		return
	}

	apierror.WriteJSON(w, http.StatusCreated, voteAcceptedResponse{
		PollID:                receipt.PollID,
		ParticipationRecorded: receipt.ParticipationRecorded,
	})
}

func (h *Handler) getResults(w http.ResponseWriter, r *http.Request) {
	actor, ok := auth.AuthenticatedUserFromContext(r.Context())
	if !ok {
		apierror.Write(w, apierror.New(http.StatusUnauthorized, apierror.CodeUnauthorized, "authentication required"))
		return
	}
	pollID := chi.URLParam(r, "pollId")
	includeVoters := false
	if v := strings.TrimSpace(r.URL.Query().Get("include_voters")); v != "" {
		switch strings.ToLower(v) {
		case "true", "1":
			includeVoters = true
		case "false", "0":
			includeVoters = false
		default:
			apierror.Write(w, apierror.WithField(http.StatusUnprocessableEntity, apierror.CodeValidationError, "include_voters", "must be true or false"))
			return
		}
	}

	results, err := h.service.Results(r.Context(), actor.UserID, pollID, includeVoters)
	if err != nil {
		apierror.Write(w, mapError(err))
		return
	}

	apierror.WriteJSON(w, http.StatusOK, toResultsDTO(results))
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
