package server

import (
	"encoding/json"
	"net/http"
)

type jMap map[string]any

func writeJson(w http.ResponseWriter, status int, body jMap) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
