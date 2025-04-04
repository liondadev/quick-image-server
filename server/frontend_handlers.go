package server

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/a-h/templ"
	"github.com/liondadev/quick-image-server/server/pages"
	"github.com/liondadev/quick-image-server/types"
)

// FrontendHandlerWithError is almost identical to HandlerWithError, but it handles
// erroneous responses by responding with an error page, not json
type FrontendHandlerWithError func(w http.ResponseWriter, r *http.Request) error

func (h FrontendHandlerWithError) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("Recovered from panic while handling frontend request for (%s) %s: %s", r.RemoteAddr, r.RequestURI, err)
			_ = writeHTML(w, http.StatusInternalServerError, pages.Error("PANIC", "500 - Internal Server Error", "Unrecoverable Server Panic"))
		}
	}()

	start := time.Now()
	err := h(w, r)
	dur := time.Since(start).String()
	if err != nil {
		var perr PublicError
		if errors.As(err, &perr) {
			log.Printf("Encountered public error when serving frontend request for (%s) %s: %s", r.RemoteAddr, r.RequestURI, err.Error())
			_ = writeHTML(w, perr.Code, pages.Error(dur, strconv.Itoa(perr.Code)+" - "+http.StatusText(perr.Code), perr.Message))

			return
		}

		log.Printf("Encountered error when serving frontend request for (%s) %s: %s", r.RemoteAddr, r.RequestURI, err.Error())
		_ = writeHTML(w, http.StatusInternalServerError, pages.Error(dur, "500 - Internal Server Error", "Internal Server Error"))

		return
	}
}

func writeHTML(w http.ResponseWriter, status int, html templ.Component) error {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(status)
	return html.Render(context.Background(), w)
}

func (s *Server) handleLoginPage(w http.ResponseWriter, r *http.Request) error {
	return writeHTML(w, http.StatusOK, pages.Login(""))
}

func (s *Server) handlePostLoginPage(w http.ResponseWriter, r *http.Request) error {
	err := r.ParseForm()
	if err != nil {
		return err
	}

	apiKey := r.FormValue("api_key")
	if apiKey == "" {
		return writeHTML(w, http.StatusBadRequest, pages.Login("Please enter an API key."))
	}

	_, ok := s.cfg.Users[apiKey]
	if !ok {
		return writeHTML(w, http.StatusBadRequest, pages.Login("Invalid API Key."))
	}

	exp := time.Now().Add(time.Hour * 24 * 30)
	http.SetCookie(w, &http.Cookie{
		Name:    "qis_api_key",
		Value:   apiKey,
		Path:    "/",
		Expires: exp,
	})

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	return nil
}

func (s *Server) handleDashboardPage(w http.ResponseWriter, r *http.Request) error {
	userName, ok := r.Context().Value(AuthenticatedUserContextKey).(string)
	if !ok {
		panic("user in middleware but not in context key?")
	}

	// Collect some statistics
	var totalUploads int = 0
	var lastUpload uint64 = 0
	if err := s.db.Get(&totalUploads, `SELECT COUNT(*) FROM "uploads" WHERE "user" = $1`, userName); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	// Collect the recent uploads
	uploads := make([]types.Upload, 16)
	if err := s.db.Select(&uploads, `SELECT "id", "mime", "user", "uploaded_at", "uploaded_as", "ext", "delete_token" FROM "uploads" WHERE "user" = $1 ORDER BY "uploaded_at" DESC LIMIT 16;`, userName); err != nil {
		return err
	}

	if len(uploads) >= 1 {
		lastUpload = uploads[0].Timestamp
	}

	return writeHTML(w, http.StatusOK, pages.Dashboard(userName, map[string]string{
		"Total Uploads": strconv.Itoa(totalUploads),
		"Last Upload":   time.Unix(int64(lastUpload), 0).Format(time.RFC1123),
	}, uploads))
}

func (s *Server) handleUploadsPage(w http.ResponseWriter, r *http.Request) error {
	userName, ok := r.Context().Value(AuthenticatedUserContextKey).(string)
	if !ok {
		panic("user in middleware but not in context key?")
	}

	pageNum := 1
	q := r.URL.Query()
	query := q.Get("search")
	pageStr := q.Get("page")

	if pn, err := strconv.Atoi(pageStr); err == nil && pn >= 1 {
		pageNum = pn
	}

	uploads := make([]types.Upload, 4*10)
	if query != "" {
		// This query is more expensive, even if we escape things properly.
		if err := s.db.Select(&uploads, `SELECT "id", "mime", "user", "uploaded_at", "uploaded_as", "ext", "delete_token" FROM "uploads" WHERE "user" = $1 AND ("id" LIKE '%' || $2 || '%' OR "uploaded_as" LIKE '%' || $2 || '%' OR "mime" = $2 OR "ext" LIKE '%' || $2 || '%') ORDER BY "uploaded_at" DESC LIMIT 40 OFFSET $3;`, userName, query, (pageNum-1)*40); err != nil {
			return err
		}
	} else {
		if err := s.db.Select(&uploads, `SELECT "id", "mime", "user", "uploaded_at", "uploaded_as", "ext", "delete_token" FROM "uploads" WHERE "user" = $1 ORDER BY "uploaded_at" DESC LIMIT 40 OFFSET $2;`, userName, (pageNum-1)*40); err != nil {
			return err
		}
	}

	return writeHTML(w, http.StatusOK, pages.Uploads(userName, uploads, query, pageNum))
}

func (s *Server) handleImportPage(w http.ResponseWriter, r *http.Request) error {
	userName, ok := r.Context().Value(AuthenticatedUserContextKey).(string)
	if !ok {
		panic("user in middleware but not in context key?")
	}

	return writeHTML(w, http.StatusOK, pages.Import(userName))
}
