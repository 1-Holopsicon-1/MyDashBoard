package model

type ServiceStatus struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	Online    bool   `json:"online"`
	LatencyMs int64  `json:"latency_ms"`
	Code      int    `json:"code"`
}
