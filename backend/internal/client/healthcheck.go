package client

import (
	"context"
	"net/http"
	"sync"
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
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return model.ServiceStatus{
			Name: name,
			URL:  url,
		}
	}

	start := time.Now()
	resp, err := c.http.Do(req)
	latency := time.Since(start).Milliseconds()

	status := model.ServiceStatus{
		Name: name,
		URL:  url,
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
	var mu sync.Mutex
	var wg sync.WaitGroup
	results := make([]model.ServiceStatus, 0, len(targets))
	for name, url := range targets {
		wg.Add(1)
		go func(name, url string) {
			defer wg.Done()
			res := c.Check(ctx, name, url)
			mu.Lock()
			results = append(results, res)
			mu.Unlock()
		}(name, url)
	}
	wg.Wait()
	return results
}
