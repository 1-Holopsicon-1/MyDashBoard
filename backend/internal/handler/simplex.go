package handler

import (
	"errors"
	"net/http"

	"MyDashBoard/internal/model"
	"MyDashBoard/internal/service"
)

type SimplexHandler struct {
	svc *service.SimplexService
}

func NewSimplex(svc *service.SimplexService) *SimplexHandler {
	return &SimplexHandler{svc: svc}
}

func (h *SimplexHandler) GetLinks(w http.ResponseWriter, r *http.Request) {
	if !IsAuthenticated(r) {
		respondError(w, http.StatusUnauthorized, errors.New("not authenticated"))
		return
	}

	links, err := h.svc.GetLinks(r.Context())
	if err != nil {
		respondError(w, http.StatusBadGateway, err)
		return
	}
	if links == nil {
		links = make([]model.SimplexLink, 0)
	}
	respondJSON(w, http.StatusOK, links)
}
