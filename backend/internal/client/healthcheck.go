package client

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"MyDashBoard/internal/model"
)

type HealthChecker struct {
	http *http.Client
}

func NewHealthChecker(httpClient *http.Client) *HealthChecker {
	return &HealthChecker{http: httpClient}
}

func (c *HealthChecker) Check(ctx context.Context, name, url string) model.ServiceStatus {
	checkURL := url + "/alive"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checkURL, nil)
	if err != nil {
		return model.ServiceStatus{
			Name: name,
			URL:  checkURL,
		}
	}

	start := time.Now()
	resp, err := c.http.Do(req)
	latency := time.Since(start).Milliseconds()

	status := model.ServiceStatus{
		Name: name,
		URL:  checkURL,
	}

	if err != nil {
		return status
	}
	defer resp.Body.Close()

	status.Online = resp.StatusCode >= 200 && resp.StatusCode < 400
	status.LatencyMs = latency
	status.Code = resp.StatusCode

	return status
}

func (c *HealthChecker) CheckAll(ctx context.Context, targets map[string]string) []model.ServiceStatus {
	results := make([]model.ServiceStatus, 0, len(targets))
	for name, url := range targets {
		results = append(results, c.Check(ctx, name, url))
	}
	return results
}

// Validate проверяет что URL доступен для запросов.
func Validate(url string) error {
	if url == "" {
		return fmt.Errorf("URL is empty")
	}
	return nil
}
