package apierror

import (
	"encoding/json"
	"net/http"
)

const (
	CodeValidationError           = "validation_error"
	CodeUnauthorized              = "unauthorized"
	CodeForbidden                 = "forbidden"
	CodeNotFound                  = "not_found"
	CodeConflict                  = "conflict"
	CodePollNotActive             = "poll_not_active"
	CodePollHidden                = "poll_hidden"
	CodePollAlreadyVoted          = "poll_already_voted"
	CodePollVotePayloadInvalid    = "poll_vote_payload_invalid"
	CodePollUpdateNotAllowed      = "poll_update_not_allowed"
	CodeReportAlreadyExists       = "report_already_exists"
	CodeReportReviewSelfForbidden = "report_review_self_forbidden"
	CodeReportReviewAlreadyExists = "report_review_already_exists"
	CodeReportAlreadyResolved     = "report_already_resolved"
	CodeQuorumNotReached          = "quorum_not_reached"
	CodeInternalError             = "internal_error"
)

type Error struct {
	Status  int
	Code    string
	Message string
	Details map[string]any
}

func (e *Error) Error() string { return e.Message }

func New(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

func WithField(status int, code, field, message string) *Error {
	return &Error{
		Status:  status,
		Code:    code,
		Message: message,
		Details: map[string]any{"field": field},
	}
}

type envelope struct {
	Error body `json:"error"`
}

type body struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details"`
}

func Write(w http.ResponseWriter, e *Error) {
	if e == nil {
		e = New(http.StatusInternalServerError, CodeInternalError, "internal error")
	}
	details := e.Details
	if details == nil {
		details = map[string]any{}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(e.Status)
	_ = json.NewEncoder(w).Encode(envelope{Error: body{Code: e.Code, Message: e.Message, Details: details}})
}

func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(payload)
}
