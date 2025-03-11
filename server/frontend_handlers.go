package server

import (
	"context"
	"errors"
	"github.com/a-h/templ"
	"github.com/liondadev/quick-image-server/server/pages"
	"log"
	"net/http"
	"strconv"
	"time"
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
		err = writeHTML(w, http.StatusInternalServerError, pages.Error(dur, "500 - Internal Server Error", "Internal Server Error"))
		return
	}
}

func writeHTML(w http.ResponseWriter, status int, html templ.Component) error {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(status)
	return html.Render(context.Background(), w)
}

func (s *Server) handleLoginPage(w http.ResponseWriter, r *http.Request) error {
	return writeHTML(w, http.StatusOK, pages.Login())
}

func (s *Server) handlePostLoginPage(w http.ResponseWriter, r *http.Request) error {
	err := r.ParseForm()
	if err != nil {
		return err
	}

	return nil
}
