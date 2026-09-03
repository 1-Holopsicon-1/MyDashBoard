package client

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
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
					dialer := &net.Dialer{Timeout: 5 * time.Second}
					return dialer.DialContext(ctx, "unix", socketPath)
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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("docker API error: %d %s", resp.StatusCode, string(body))
	}

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

func (c *DockerClient) GetSimpleXLinks(ctx context.Context, filters []string) ([]model.SimplexLink, error) {
	containers, err := c.listAllContainers(ctx)
	if err != nil {
		return nil, err
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	var allLinks []model.SimplexLink
	for _, cont := range containers {
		name := containerName(cont.Names)
		if !matchFilters(name, filters) {
			continue
		}

		wg.Add(1)
		go func(contID, contName string) {
			defer wg.Done()
			callCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			links := c.parseServerAddresses(callCtx, contID, contName)
			mu.Lock()
			allLinks = append(allLinks, links...)
			mu.Unlock()
		}(cont.ID, name)
	}
	wg.Wait()

	return allLinks, nil
}

type dockerContainerFull struct {
	ID    string   `json:"Id"`
	Names []string `json:"Names"`
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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("docker API error: %d %s", resp.StatusCode, string(body))
	}

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

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	links, err := parseAddressesFromLogs(resp.Body, containerName)
	if err != nil {
		return nil
	}
	return links
}

func parseAddressesFromLogs(r io.Reader, containerName string) ([]model.SimplexLink, error) {
	var links []model.SimplexLink
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := scanner.Text()
		idx := strings.Index(line, "Server address:")
		if idx == -1 {
			continue
		}

		addr := strings.TrimSpace(line[idx+len("Server address:"):])
		if addr == "" {
			continue
		}

		addr = stripANSI(addr)

		links = append(links, model.SimplexLink{
			Container: containerName,
			Address:   addr,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read log stream: %w", err)
	}

	return links, nil
}

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\][^\x07]*(\x07|\x1b\\)|\x1b[()][AB0]`)

func stripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}
