package handler

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"

	"MyDashBoard/internal/model"
)

func respondJSON(w http.ResponseWriter, status int, data any) {
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(data); err != nil {
		log.Printf("json encode error: %v", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(buf.Bytes())
}

func respondError(w http.ResponseWriter, code int, err error) {
	log.Printf("request error [%d]: %v", code, err)
	respondJSON(w, code, model.ErrorResponse{Error: http.StatusText(code)})
}
