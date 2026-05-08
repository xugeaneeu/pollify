package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"xugeaneeu/pollify/internal/pkg/apierror"
	"xugeaneeu/pollify/internal/platform/auth"
	users "xugeaneeu/pollify/internal/users/core"
)

type Service interface {
	Register(ctx context.Context, input users.RegisterInput) (users.AuthSession, error)
	Login(ctx context.Context, input users.LoginInput) (users.AuthSession, error)
	CurrentUser(ctx context.Context, actor users.AuthenticatedUser) (users.User, error)
}

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) PublicRoutes() chi.Router {
	r := chi.NewRouter()
	r.Post("/register", h.register)
	r.Post("/login", h.login)
	return r
}

func (h *Handler) ProtectedRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/me", h.me)
	return r
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		apierror.Write(w, err)
		return
	}

	displayName := ""
	if req.DisplayName != nil {
		displayName = *req.DisplayName
	}

	session, err := h.service.Register(r.Context(), users.RegisterInput{
		Email:       req.Email,
		Password:    req.Password,
		DisplayName: displayName,
	})
	if err != nil {
		apierror.Write(w, mapError(err))
		return
	}

	apierror.WriteJSON(w, http.StatusCreated, toSessionResponse(session))
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		apierror.Write(w, err)
		return
	}

	session, err := h.service.Login(r.Context(), users.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		apierror.Write(w, mapError(err))
		return
	}

	apierror.WriteJSON(w, http.StatusOK, toSessionResponse(session))
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	actor, ok := auth.AuthenticatedUserFromContext(r.Context())
	if !ok {
		apierror.Write(w, apierror.New(http.StatusUnauthorized, apierror.CodeUnauthorized, "authentication required"))
		return
	}

	user, err := h.service.CurrentUser(r.Context(), actor)
	if err != nil {
		apierror.Write(w, mapError(err))
		return
	}

	apierror.WriteJSON(w, http.StatusOK, toUserSummary(user))
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
