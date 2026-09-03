package service

import (
	"context"
	"fmt"

	"MyDashBoard/internal/client"
	"MyDashBoard/internal/model"
)

type TailscaleService struct {
	client *client.TailscaleClient
}

func NewTailscale(c *client.TailscaleClient) *TailscaleService {
	return &TailscaleService{client: c}
}

func (s *TailscaleService) GetDevices(ctx context.Context) ([]model.TailscaleDevice, error) {
	devices, err := s.client.GetDevices(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching devices: %w", err)
	}
	return devices, nil
}
