package server

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"time"
)

const FileIdLength = 12

// handleIndex handles requests to GET /
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) error {
	return PublicError{http.StatusNotFound, "Page not found."}
}

// handleNotFound is called when no other handlers match the request. In other words, this is called
// when the page is not found or the route doesn't exist.
func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) error {
	return PublicError{http.StatusNotFound, "Page not found."}
}

// handleFileUpload is called when someone tries to upload a file.
func (s *Server) handleFileUpload(w http.ResponseWriter, r *http.Request) error {
	userName, ok := r.Context().Value(AuthenticatedUserContextKey).(string)
	if !ok {
		panic("user in middleware but not in context key?")
	}

	err := r.ParseMultipartForm(1024 * 8)
	if err != nil {
		return err
	}

	uploadedFile, header, err := r.FormFile("upload")
	if err != nil {
		return err
	}

	fileId, err := s.getFreeFileId(FileIdLength)
	if err != nil {
		fmt.Println(err)
		return PublicError{http.StatusInternalServerError, "failed to generate id"}
	}

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream" // default for binary data
	}

	ext := path.Ext(header.Filename)
	diskName := fileId + ext

	// Handle storing the file
	fullPath := path.Join(s.cfg.FSPath, diskName)
	f, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer f.Close()

	n, err := io.Copy(f, uploadedFile)
	if err != nil {
		return err
	}

	log.Printf("User '%s' uploaded file '%s' (%s), n=%d\n", userName, header.Filename, diskName, n)

	// Handle storing the upload in the database
	deleteToken := s.generateDeleteToken(32)
	if _, err := s.db.Exec(`INSERT INTO "uploads" ("id", "mime", "user", "uploaded_at", "uploaded_as", "delete_token", "ext") VALUES ($1, $2, $3, $4, $5, $6, $7)`, fileId, mimeType, userName, time.Now().Unix(), header.Filename, deleteToken, ext); err != nil {
		// If we fail the database query we need to delete the file
		f.Close()
		_ = os.Remove(fullPath) // if we error here it's already too late

		return err
	}

	uploadUrl, err := url.JoinPath(s.cfg.BasePath, "/f/", diskName)
	if err != nil {
		return err
	}

	thumbUrl, err := url.JoinPath(s.cfg.BasePath, "/thumb/", diskName)
	if err != nil {
		return err
	}

	deleteUrl, err := url.JoinPath(s.cfg.BasePath, "/delete/", fileId, "/", deleteToken)

	writeJson(w, http.StatusCreated, jMap{ // it was a success!
		"file_url":      uploadUrl,
		"thumbnail_url": thumbUrl,
		"delete_url":    deleteUrl,
	})
	return nil
}

func getFileDetails(r *http.Request) (fileName string, fileId string) {
	fileName = chi.URLParam(r, "file")
	ext := path.Ext(fileName)
	fileId = fileName[:len(fileName)-len(ext)]

	return fileName, fileId
}

func (s *Server) handleFileView(w http.ResponseWriter, r *http.Request) error {
	fileName, fileId := getFileDetails(r)

	// make sure the file exists (this is quicker than the sql query???)
	cleanPath := filepath.Clean(fileName) // prevent path traversal -- which should be impossible anyway
	diskPath := path.Join(s.cfg.FSPath, cleanPath)

	f, err := os.Open(diskPath)
	if err != nil {
		return PublicError{http.StatusNotFound, "File not found."}
	}
	defer f.Close()

	var mimeType string
	if err := s.db.Get(&mimeType, `SELECT "mime" FROM "uploads" WHERE "id" = $1`, fileId); err != nil {
		// This means the file exists on disk but not in the database??
		return err
	}

	setCacheControlHeaders(w)
	w.Header().Set("Content-Type", mimeType)
	w.WriteHeader(200)

	if _, err := io.Copy(w, f); err != nil {
		return err
	}

	return nil
}

// handleThumbnailView handles people viewing the thumbnail images of files. The thumbnails are
// always 480x270, and pngs.
func (s *Server) handleThumbnailView(w http.ResponseWriter, r *http.Request) error {
	fileName, fileId := getFileDetails(r)
	diskPath := path.Join(s.cfg.FSPath, fileId+".thumbnail.png")
	originalDiskPath := path.Join(s.cfg.FSPath, filepath.Clean(fileName))

	var mimeType string
	if err := s.db.Get(&mimeType, `SELECT "mime" FROM "uploads" WHERE "id" = $1`, fileId); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PublicError{http.StatusNotFound, "File not Found."}
		}

		return err
	}

	// We already have the thumbnail image cached.
	if f, err := os.Open(diskPath); err == nil {
		defer f.Close()
		w.Header().Set("Content-Type", mimeType)
		w.WriteHeader(http.StatusOK)
		if _, err = io.Copy(w, f); err != nil {
			return err
		}

		return nil
	}

	// If the file isn't one of the allowed thumbnail types, we
	// return the default thumbnail.
	if !slices.Contains(AllowedThumbnailMimeTypes, mimeType) {
		defaultThumbnailPath, err := url.JoinPath(s.cfg.BasePath, "/assets/img/default_thumbnail.png")
		if err != nil {
			return err
		}

		http.Redirect(w, r, defaultThumbnailPath, http.StatusTemporaryRedirect)
		return nil
	}

	origFIle, err := os.Open(originalDiskPath)
	if err != nil {
		return err
	}
	defer origFIle.Close()

	thumb, err := s.MakeThumbnail(mimeType, origFIle)
	if err != nil {
		return err
	}

	outFile, err := os.Create(diskPath)
	if err != nil {
		return nil
	}
	defer outFile.Close()

	if _, err = io.Copy(outFile, thumb); err != nil {
		return err
	}
	outFile.Close()

	if f, err := os.Open(diskPath); err == nil {
		defer f.Close()
		setCacheControlHeaders(w)
		w.Header().Set("Content-Type", mimeType)
		w.WriteHeader(http.StatusOK)
		if _, err = io.Copy(w, f); err != nil {
			return err
		}

		return nil
	}

	return errors.New("thumbnail created, but not saved to disk")
}

func (s *Server) handleDeleteFile(w http.ResponseWriter, r *http.Request) error {
	fileId := chi.URLParam(r, "fileId")
	deleteToken := chi.URLParam(r, "deleteToken")
	// todo: delete main image and thumbnail

	var fileExt string
	if err := s.db.Get(&fileExt, `DELETE FROM "uploads" WHERE "id" = $1 AND "delete_token" = $2 RETURNING "ext"`, fileId, deleteToken); err != nil {
		return PublicError{http.StatusNotFound, "File upload not found or delete token is incorrect."}
	}

	if err := os.Remove(path.Join(s.cfg.FSPath, "/"+fileId+fileExt)); err != nil {
		return err
	}

	if err := os.Remove(path.Join(s.cfg.FSPath, "/"+fileId+".thumbnail.png")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	writeJson(w, http.StatusOK, jMap{"message": "File Deleted"})
	return nil
}

func setCacheControlHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "public, max-age=1800") // 30 min cache time
}
