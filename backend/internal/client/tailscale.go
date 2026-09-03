package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"MyDashBoard/internal/model"
)

type TailscaleClient struct {
	apiKey  string
	tailnet string
	http    *http.Client
}

func NewTailscale(httpClient *http.Client, apiKey, tailnet string) *TailscaleClient {
	return &TailscaleClient{
		apiKey:  apiKey,
		tailnet: tailnet,
		http:    httpClient,
	}
}

func (c *TailscaleClient) GetDevices(ctx context.Context) ([]model.TailscaleDevice, error) {
	url := fmt.Sprintf("https://api.tailscale.com/api/v2/tailnet/%s/devices?fields=all", c.tailnet)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tailscale API returned status %d", resp.StatusCode)
	}

	var result model.TailscaleDevicesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return result.Devices, nil
}
