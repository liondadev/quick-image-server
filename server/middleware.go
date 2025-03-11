package server

import (
	"context"
	"errors"
	"net/http"
)

const (
	AuthenticatedUserAPIKeyContextKey = "qis::api_key"
	AuthenticatedUserContextKey       = "qis::authenticated_user"
)

// preHandleAuthentication sets the context with the key AuthenticatedUserContextKey to be either the
// name of the authenticated user, or an empty string if the user isn't authenticated.
func (s *Server) preHandleAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var apiKey string
		// check header
		h := r.Header.Get("X-Server-Api-Key")
		if h != "" {
			apiKey = h
		}

		// check cookie - for the frontend
		cook, err := r.Cookie("qis_api_key")
		if err == nil {
			apiKey = cook.Value
		}

		user, ok := s.cfg.Users[apiKey]
		if !ok {
			ctx := context.WithValue(r.Context(), AuthenticatedUserAPIKeyContextKey, "")
			ctx = context.WithValue(ctx, AuthenticatedUserContextKey, "")
			next.ServeHTTP(w, r.WithContext(ctx))

			return
		}

		ctx := context.WithValue(r.Context(), AuthenticatedUserAPIKeyContextKey, apiKey)
		ctx = context.WithValue(ctx, AuthenticatedUserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) preHandleRequireAuthentication(next http.Handler) http.Handler {
	return HandlerWithError(func(w http.ResponseWriter, r *http.Request) error {
		username := r.Context().Value(AuthenticatedUserContextKey)
		if username == nil {
			return errors.New("attempted to require authentication when the prehandleauthentication middleware isn't called")
		}

		if username == "" {
			return PublicError{http.StatusUnauthorized, "This page requires authentication."}
		}

		next.ServeHTTP(w, r)

		return nil
	})
}
