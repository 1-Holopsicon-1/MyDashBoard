package handler

import (
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
	links, err := h.svc.GetLinks(r.Context())
	if err != nil {
		respondError(w, http.StatusBadGateway, err.Error())
		return
	}
	if links == nil {
		links = make([]model.SimplexLink, 0)
	}
	respondJSON(w, http.StatusOK, links)
}
