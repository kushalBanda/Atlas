package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"
)

// dockerSocketPath is the standard Docker Engine API Unix socket.
const dockerSocketPath = "/var/run/docker.sock"

// dockerContainer is the subset of the Docker Engine API's
// GET /containers/json response this discoverer needs.
type dockerContainer struct {
	Image string   `json:"Image"`
	Names []string `json:"Names"`
	Ports []struct {
		PrivatePort int    `json:"PrivatePort"`
		PublicPort  int    `json:"PublicPort"`
		Type        string `json:"Type"`
	} `json:"Ports"`
}

// containerLister lists running containers. Abstracted so
// DockerDiscoverer is testable without a real Docker socket.
type containerLister interface {
	ListContainers(ctx context.Context) ([]dockerContainer, error)
}

// DockerDiscoverer finds running Docker containers and matches their
// image name and exposed ports against curated rules.
type DockerDiscoverer struct {
	rules  []Rule
	lister containerLister
}

// NewDockerDiscoverer returns a Docker Discoverer matching against rules,
// talking to the local Docker socket at dockerSocketPath.
func NewDockerDiscoverer(rules []Rule) *DockerDiscoverer {
	return &DockerDiscoverer{rules: rules, lister: dockerSocketLister{}}
}

// Name implements Discoverer.
func (d *DockerDiscoverer) Name() string { return "docker" }

// Discover implements Discoverer: every running container's exposed ports
// are reported, matched or not. A container is reported once per exposed
// port so each port can independently match a curated rule.
func (d *DockerDiscoverer) Discover(ctx context.Context) ([]Target, error) {
	containers, err := d.lister.ListContainers(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing docker containers: %w", err)
	}

	var targets []Target
	for _, c := range containers {
		image := c.Image
		for _, p := range c.Ports {
			port := p.PublicPort
			if port == 0 {
				port = p.PrivatePort
			}
			if port == 0 {
				continue
			}
			targets = append(targets, Target{
				Host:           "localhost",
				Port:           port,
				ProcessOrImage: image,
				MatchedRule:    matchRule(d.rules, port, image),
			})
		}
	}
	return targets, nil
}

// dockerSocketLister lists containers via the local Docker Engine API
// Unix socket.
type dockerSocketLister struct{}

func (dockerSocketLister) ListContainers(ctx context.Context) ([]dockerContainer, error) {
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, "unix", dockerSocketPath)
			},
		},
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://docker/containers/json", nil)
	if err != nil {
		return nil, fmt.Errorf("building docker api request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling docker api at %s: %w", dockerSocketPath, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("docker api returned status %d", resp.StatusCode)
	}

	var containers []dockerContainer
	if err := json.NewDecoder(resp.Body).Decode(&containers); err != nil {
		return nil, fmt.Errorf("decoding docker api response: %w", err)
	}
	return containers, nil
}
