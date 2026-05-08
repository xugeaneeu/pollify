package auth

import (
	"net/http"

	users "xugeaneeu/pollify/internal/users/core"
	"xugeaneeu/pollify/internal/pkg/apierror"
)

func RequireRole(role users.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actor, ok := AuthenticatedUserFromContext(r.Context())
			if !ok {
				apierror.Write(w, apierror.New(http.StatusUnauthorized, apierror.CodeUnauthorized, "authentication required"))
				return
			}
			if actor.Role != role {
				apierror.Write(w, apierror.New(http.StatusForbidden, apierror.CodeForbidden, "insufficient role"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
