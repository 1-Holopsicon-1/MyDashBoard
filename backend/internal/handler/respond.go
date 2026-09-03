package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"MyDashBoard/internal/model"
)

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, code int, err error) {
	log.Printf("request error [%d]: %v", code, err)
	respondJSON(w, code, model.ErrorResponse{Error: http.StatusText(code)})
}
