package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jmoiron/sqlx"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
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

		http.Redirect(w, r, defaultThumbnailPath, http.StatusPermanentRedirect) // perma because it can never magically get a thumbnail.
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

func (s *Server) handleRunImport(w http.ResponseWriter, r *http.Request) {
	userName, ok := r.Context().Value(AuthenticatedUserContextKey).(string)
	if !ok {
		panic("user in middleware but not in context key?")
	}

	_ = userName

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)

	wf, ok := w.(http.Flusher)
	if !ok {
		panic("Handler is not a flusher.")
	}

	writeMessage := func(name string, content []byte) error {
		message := "event: " + name + "\n"
		message = message + "data: "
		messageBytes := append([]byte(message), content...)
		messageBytes = append(messageBytes, '\n', '\n')

		_, err := w.Write(messageBytes)
		if err != nil {
			return err
		}

		wf.Flush()
		return nil
	}

	typeSuccess := "success"
	typeInfo := "info"
	typeFail := "fail"
	_ = typeSuccess

	writeNormalMessage := func(t string, msg string) error {
		data := map[string]string{
			"type":    t,
			"content": msg,
		}

		jsb, err := json.Marshal(data)
		if err != nil {
			return err
		}

		return writeMessage("message", jsb)
	}

	fileName := r.URL.Query().Get("fileName")
	if fileName == "" {
		_ = writeNormalMessage(typeFail, "No file name provided.")
		return
	}

	_ = writeNormalMessage(typeInfo, "Checking for file "+fileName+".")

	fullPath := path.Join(s.cfg.BaseImportPath, fileName)
	_ = writeNormalMessage(typeInfo, "Full path to check: "+fullPath)

	if _, err := os.Stat(fullPath); err != nil {
		_ = writeNormalMessage(typeFail, "File does not exist on disk.")
		return
	}

	db, err := sqlx.Open("sqlite", fullPath)
	if err != nil {
		_ = writeNormalMessage(typeFail, "Failed to open SQLite database connection: "+err.Error())
		return
	}

	type LegacyUpload struct {
		Id          string `db:"id"`
		Extension   string `db:"ext"`
		DataBlob    []byte `db:"blob"`
		UploadedAs  string `db:"original_filename"`
		DeleteToken string `db:"delete_token"`
	}
	perPage := 25
	curPage := 1

	// go through all the pages
	for curPage <= 1000 {
		var uploads []LegacyUpload
		offset := (curPage - 1) * perPage
		if err := db.Select(&uploads, `SELECT "id", "ext", "blob", "original_filename", "delete_token" FROM "files" ORDER BY "id" LIMIT $1 OFFSET $2`, perPage, offset); err != nil {
			_ = writeNormalMessage(typeFail, "Failed to get files from pagination: "+err.Error())
			break
		}

		_ = writeNormalMessage(typeInfo, "Handling paginated page "+strconv.Itoa(curPage)+" ("+strconv.Itoa(len(uploads))+")")
		for _, up := range uploads {
			_ = writeNormalMessage(typeInfo, "Handling file with ID: "+up.Id)

			fullPath := path.Join(s.cfg.FSPath, up.Id+up.Extension)

			if _, err := os.Stat(fullPath); err == nil {
				_ = writeNormalMessage(typeInfo, "Skipping "+up.Id+" because a file already exists on our FS.")
				continue
			}

			var mimeType string = "text/plain"
			switch up.Extension {
			case ".png":
				mimeType = "image/png"
			case ".jpg", ".jpeg":
				mimeType = "image/jpeg"
			case ".gif":
				mimeType = "image/gif"
			case ".txt":
				mimeType = "text/plain"
			case ".bin":
				mimeType = "application/octet-stream"
			case ".mp4":
				mimeType = "video/mp4"
			case ".html":
				mimeType = "application/html"
			case ".md":
				mimeType = "text/markdown"
			case ".xlsx":
				mimeType = " application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
			}

			// insert into THE REAL db
			if _, err := s.db.Exec(`INSERT INTO "uploads" ("id", "mime", "user", "uploaded_at", "uploaded_as", "delete_token", "ext") VALUES ($1, $2, $3, $4, $5, $6, $7)`, up.Id, mimeType, userName, time.Now().Unix(), up.UploadedAs, up.DeleteToken, up.Extension); err != nil {
				_ = writeNormalMessage(typeFail, "Failed to insert file "+up.Id+" into the database: "+err.Error())
				continue
			}

			// store on the file system
			f, err := os.Create(fullPath)
			if err != nil {
				_ = writeNormalMessage(typeFail, "Failed to create disk file "+up.Id+": "+err.Error())
				continue
			}

			_, err = f.Write(up.DataBlob)
			if err != nil {
				_ = writeNormalMessage(typeFail, "Failed to write to file with id "+up.Id+": "+err.Error())
			}

			f.Close()
		}

		if len(uploads) < perPage {
			_ = writeNormalMessage(typeInfo, "Reached end of pagination.")
			break
		}

		curPage++
	}

	_ = writeNormalMessage(typeSuccess, "Done!")
}
