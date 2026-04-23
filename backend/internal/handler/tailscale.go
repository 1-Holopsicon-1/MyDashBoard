package handler

import (
	"net/http"

	"MyDashBoard/internal/service"
)

type TailscaleHandler struct {
	svc *service.TailscaleService
}

func NewTailscale(svc *service.TailscaleService) *TailscaleHandler {
	return &TailscaleHandler{svc: svc}
}

func (h *TailscaleHandler) GetDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := h.svc.GetDevices(r.Context())
	if err != nil {
		respondError(w, http.StatusBadGateway, err.Error())
		return
	}

	showIP := IsAuthenticated(r)
	resp := make([]interface{}, len(devices))
	for i, d := range devices {
		resp[i] = d.ToResponse(showIP)
	}

	respondJSON(w, http.StatusOK, resp)
}
