package service

import (
	"context"

	"MyDashBoard/internal/client"
	"MyDashBoard/internal/model"
)

type HealthService struct {
	checker *client.HealthChecker
	targets map[string]string // name -> base URL
}

func NewHealth(checker *client.HealthChecker, targets map[string]string) *HealthService {
	return &HealthService{
		checker: checker,
		targets: targets,
	}
}

func (s *HealthService) CheckAll(ctx context.Context) []model.ServiceStatus {
	return s.checker.CheckAll(ctx, s.targets)
}
