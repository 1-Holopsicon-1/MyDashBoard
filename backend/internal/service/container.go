package service

import (
	"context"

	"MyDashBoard/internal/client"
	"MyDashBoard/internal/model"
)

type ContainerService struct {
	client  *client.DockerClient
	filters []string // substrings to match container names
}

func NewContainer(c *client.DockerClient, filters []string) *ContainerService {
	return &ContainerService{
		client:  c,
		filters: filters,
	}
}

func (s *ContainerService) GetContainers(ctx context.Context) ([]model.ContainerStatus, error) {
	return s.client.GetContainers(ctx, s.filters)
}
