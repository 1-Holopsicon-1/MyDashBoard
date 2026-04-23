package client

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"MyDashBoard/internal/model"
)

type DockerClient struct {
	http       *http.Client
	socketPath string
}

func NewDocker(socketPath string) *DockerClient {
	if socketPath == "" {
		socketPath = "/var/run/docker.sock"
	}
	return &DockerClient{
		socketPath: socketPath,
		http: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					return net.DialTimeout("unix", socketPath, 5*time.Second)
				},
			},
		},
	}
}

type dockerContainer struct {
	Names  []string `json:"Names"`
	Image  string   `json:"Image"`
	State  string   `json:"State"`
	Status string   `json:"Status"`
}

func (c *DockerClient) GetContainers(ctx context.Context, filters []string) ([]model.ContainerStatus, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost/containers/json?all=true", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connecting to Docker socket %s: %w", c.socketPath, err)
	}
	defer resp.Body.Close()

	var containers []dockerContainer
	if err := json.NewDecoder(resp.Body).Decode(&containers); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return filterContainers(containers, filters), nil
}

func filterContainers(containers []dockerContainer, filters []string) []model.ContainerStatus {
	var result []model.ContainerStatus

	for _, c := range containers {
		name := containerName(c.Names)

		if !matchFilters(name, filters) {
			continue
		}

		result = append(result, model.ContainerStatus{
			Name:   name,
			Image:  c.Image,
			State:  c.State,
			Status: c.Status,
			Online: c.State == "running",
		})
	}

	return result
}

func containerName(names []string) string {
	if len(names) == 0 {
		return "unknown"
	}
	return strings.TrimPrefix(names[0], "/")
}

func matchFilters(name string, filters []string) bool {
	if len(filters) == 0 {
		return true
	}
	for _, f := range filters {
		if strings.Contains(strings.ToLower(name), strings.ToLower(f)) {
			return true
		}
	}
	return false
}

// GetSimpleXLinks получает адреса SimpleX серверов из логов контейнеров.
// Ищет строки "Server address:" в stdout/stderr логах.
func (c *DockerClient) GetSimpleXLinks(ctx context.Context, filters []string) ([]model.SimplexLink, error) {
	// Сначала получаем список контейнеров
	containers, err := c.listAllContainers(ctx)
	if err != nil {
		return nil, err
	}

	var links []model.SimplexLink
	for _, cont := range containers {
		name := containerName(cont.Names)
		if !matchFilters(name, filters) {
			continue
		}

		containerLinks := c.parseServerAddresses(ctx, cont.ID, name)
		links = append(links, containerLinks...)
	}

	return links, nil
}

type dockerContainerFull struct {
	ID     string   `json:"Id"`
	Names  []string `json:"Names"`
}

func (c *DockerClient) listAllContainers(ctx context.Context) ([]dockerContainerFull, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost/containers/json?all=true", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var containers []dockerContainerFull
	if err := json.NewDecoder(resp.Body).Decode(&containers); err != nil {
		return nil, err
	}
	return containers, nil
}

func (c *DockerClient) parseServerAddresses(ctx context.Context, containerID, containerName string) []model.SimplexLink {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("http://localhost/containers/%s/logs?stdout=true&stderr=true&tail=200", containerID), nil)
	if err != nil {
		return nil
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	return parseAddressesFromLogs(resp.Body, containerName)
}

func parseAddressesFromLogs(r io.Reader, containerName string) []model.SimplexLink {
	var links []model.SimplexLink
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := scanner.Text()
		// Docker log format: header (8 bytes) + payload
		// Ищем "Server address:" в строке
		idx := strings.Index(line, "Server address:")
		if idx == -1 {
			continue
		}

		addr := strings.TrimSpace(line[idx+len("Server address:"):])
		if addr == "" {
			continue
		}

		// Убираем ANSI escape коды если есть
		addr = stripANSI(addr)

		links = append(links, model.SimplexLink{
			Container: containerName,
			Address:   addr,
		})
	}

	return links
}

func stripANSI(s string) string {
	var result strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' {
			// Skip escape sequence
			i++
			if i < len(s) && s[i] == '[' {
				i++
				for i < len(s) && !((s[i] >= 'A' && s[i] <= 'Z') || (s[i] >= 'a' && s[i] <= 'z')) {
					i++
				}
				if i < len(s) {
					i++
				}
			}
		} else {
			result.WriteByte(s[i])
			i++
		}
	}
	return result.String()
}
