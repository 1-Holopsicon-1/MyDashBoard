package handler

import (
	"encoding/json"
	"net/http"

	"MyDashBoard/internal/model"
)

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, msg string) {
	respondJSON(w, status, model.ErrorResponse{Error: msg})
}
