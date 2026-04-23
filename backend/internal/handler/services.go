package handler

import (
	"net/http"

	"MyDashBoard/internal/service"
)

type ServicesHandler struct {
	svc *service.HealthService
}

func NewServices(svc *service.HealthService) *ServicesHandler {
	return &ServicesHandler{svc: svc}
}

func (h *ServicesHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	statuses := h.svc.CheckAll(r.Context())
	respondJSON(w, http.StatusOK, statuses)
}
