package handler

import (
	"net/http"

	"MyDashBoard/internal/model"
	"MyDashBoard/internal/service"
)

type ContainersHandler struct {
	svc *service.ContainerService
}

func NewContainers(svc *service.ContainerService) *ContainersHandler {
	return &ContainersHandler{svc: svc}
}

func (h *ContainersHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	containers, err := h.svc.GetContainers(r.Context())
	if err != nil {
		respondError(w, http.StatusBadGateway, err.Error())
		return
	}
	if containers == nil {
		containers = make([]model.ContainerStatus, 0)
	}
	respondJSON(w, http.StatusOK, containers)
}
