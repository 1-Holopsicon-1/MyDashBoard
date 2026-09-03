package service

import (
	"context"

	"MyDashBoard/internal/client"
	"MyDashBoard/internal/model"
)

type SimplexService struct {
	client  *client.DockerClient
	filters []string
}

func NewSimplex(c *client.DockerClient, filters []string) *SimplexService {
	return &SimplexService{client: c, filters: filters}
}

func (s *SimplexService) GetLinks(ctx context.Context) ([]model.SimplexLink, error) {
	return s.client.GetSimpleXLinks(ctx, s.filters)
}
