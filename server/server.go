package server

import (
	"embed"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jmoiron/sqlx"
	"github.com/liondadev/quick-image-server/config"
)

//go:embed assets/*
var assetFs embed.FS

type PublicError struct {
	Code    int
	Message string
}

func (pe PublicError) Error() string {
	return fmt.Sprintf("(%d) %s", pe.Code, pe.Message)
}

// HandlerWithError is a wrapper around a http.Handler that allows you to return an error.
type HandlerWithError func(w http.ResponseWriter, r *http.Request) error

func (h HandlerWithError) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("Recovered from panic while handling request for (%s) %s: %s", r.RemoteAddr, r.RequestURI, err)

			writeJson(w, http.StatusInternalServerError, jMap{
				"error": "Unrecoverable Serverside Panic!",
			})
		}
	}()

	err := h(w, r)
	if err != nil {
		var perr PublicError
		if errors.As(err, &perr) {
			log.Printf("Encountered public error when serving request for (%s) %s: %s", r.RemoteAddr, r.RequestURI, err.Error())
			writeJson(w, perr.Code, jMap{
				"error": perr.Error(),
			})

			return
		}

		log.Printf("Encountered error when serving request for (%s) %s: %s", r.RemoteAddr, r.RequestURI, err.Error())
		writeJson(w, http.StatusInternalServerError, jMap{
			"error": "Internal Server Error!",
		})
	}
}

type Server struct {
	db  *sqlx.DB
	cfg *config.Config
	mux *chi.Mux
}

// New creates a new server instance from the config and database instance.
func New(cfg *config.Config, db *sqlx.DB) *Server {
	return &Server{
		cfg: cfg,
		db:  db,
	}
}

func (s *Server) SetupHTTP() error {
	mux := chi.NewMux()

	mux.Use(middleware.RealIP)
	mux.Use(middleware.Compress(5))
	mux.Use(middleware.Recoverer)
	mux.Use(middleware.CleanPath)

	// API Routes
	mux.With(s.preHandleAuthentication).With(s.preHandleRequireAuthentication).Handle("POST /upload", HandlerWithError(s.handleFileUpload))
	mux.Handle("GET /f/{file}", FrontendHandlerWithError(s.handleFileView))
	mux.Handle("GET /thumb/{file}", FrontendHandlerWithError(s.handleThumbnailView))
	mux.Handle("GET /delete/{fileId}/{deleteToken}", HandlerWithError(s.handleDeleteFile))

	// Frontend Routes
	mux.Handle("GET /app/login", FrontendHandlerWithError(s.handleLoginPage))
	mux.Handle("POST /app/login", FrontendHandlerWithError(s.handlePostLoginPage))
	mux.With(s.preHandleAuthentication).With(s.preHandleRequireAuthentication).Handle("GET /app", FrontendHandlerWithError(s.handleDashboardPage))
	mux.With(s.preHandleAuthentication).With(s.preHandleRequireAuthentication).Handle("GET /app/uploads", FrontendHandlerWithError(s.handleUploadsPage))

	// Redirects favicon to /assets/favicon.ico
	mux.Handle("GET /favicon.ico", HandlerWithError(func(w http.ResponseWriter, r *http.Request) error {
		path, err := url.JoinPath(s.cfg.BasePath, "/assets/img/favicon.ico")
		if err != nil {
			return err
		}
		http.Redirect(w, r, path, http.StatusPermanentRedirect)

		return nil
	}))

	// Static Assets
	httpFs := http.FileServerFS(assetFs)
	mux.Mount("/assets/", httpFs)

	// Not found handler
	mux.NotFound(FrontendHandlerWithError(s.handleNotFound).ServeHTTP)

	// Debug route logging
	//var routes []string
	//_ = chi.Walk(mux, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
	//	routes = append(routes, method+" - "+route
	//dle
	//	return nil
	//})
	//
	//fmt.Println(routes)

	s.mux = mux

	return nil
}

func (s *Server) Run(addr string) error {
	if s.mux == nil {
		return errors.New("the http mux hasn't been configured yet, call setuphttp()")
	}

	return http.ListenAndServe(addr, s.mux)
}

// ApplyMigrations creates all the SQL tables and stuff needed for the service to work.
func (s *Server) ApplyMigrations() error {
	// 001 - create database
	stmt := `CREATE TABLE IF NOT EXISTS "uploads" ("id" TEXT, "mime" TEXT, "user" TEXT, "uploaded_at" INTEGER, "uploaded_as" TEXT, "delete_token" TEXT, "ext" TEXT)`
	if _, err := s.db.Exec(stmt); err != nil {
		return fmt.Errorf("create initial schema: %w", err)
	}

	return nil
}
