package types

// Upload represents an uploaded file in the database.
type Upload struct {
	Id         string `db:"id"`
	MimeType   string `db:"mime"`
	User       string `db:"user"`
	Timestamp  uint64 `db:"uploaded_at"`
	UploadedAs string `db:"uploaded_as"`
	Extension  string `db:"ext"`
}
