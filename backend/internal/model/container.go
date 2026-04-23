package model

type ContainerStatus struct {
	Name   string `json:"name"`
	Image  string `json:"image"`
	State  string `json:"state"`  // running, exited, etc.
	Status string `json:"status"` // "Up 3 days", "Exited (0) 2h ago", etc.
	Online bool   `json:"online"`
}
