package server

import (
	"errors"
	"math/rand/v2"
)

var chars = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ_")

const maxGetFreeFileIdDepth = 4

func (s *Server) internalGetFreeFileId(n int, depth int) (string, error) {
	if depth >= maxGetFreeFileIdDepth {
		return "", errors.New("reached max depth/recursion limit")
	}

	// generate our id
	id := make([]byte, n)
	for idx := 0; idx < len(id); idx++ {
		id[idx] = chars[rand.IntN(len(chars))]
	}
	idStr := string(id)

	if err := s.db.Get(&(struct{}{}), `SELECT "id" FROM "uploads" WHERE "id" = $1 LIMIT 1`, idStr); err == nil {
		return s.internalGetFreeFileId(n, depth+1)
	}

	return idStr, nil
}

// getFreeFileId generates a random string of length n, that
// is not currently in use as a file id
func (s *Server) getFreeFileId(n int) (string, error) {
	return s.internalGetFreeFileId(n, 1)
}

func (s *Server) generateDeleteToken(n int) string {
	id := make([]byte, n)
	for idx := 0; idx < len(id); idx++ {
		id[idx] = chars[rand.IntN(len(chars))]
	}
	return string(id)
}
