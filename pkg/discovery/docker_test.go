package discovery

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeContainerLister struct {
	containers []dockerContainer
	err        error
}

func (f fakeContainerLister) ListContainers(_ context.Context) ([]dockerContainer, error) {
	return f.containers, f.err
}

func TestDocker_MatchesImageName(t *testing.T) {
	t.Parallel()
	rules := []Rule{{Port: 5432, ProcessMatch: "postgres", ReceiverConfig: "postgresql"}}
	container := dockerContainer{Image: "postgres:15", Names: []string{"/my-db"}}
	container.Ports = []struct {
		PrivatePort int    `json:"PrivatePort"`
		PublicPort  int    `json:"PublicPort"`
		Type        string `json:"Type"`
	}{{PrivatePort: 5432, PublicPort: 5432, Type: "tcp"}}

	d := &DockerDiscoverer{rules: rules, lister: fakeContainerLister{containers: []dockerContainer{container}}}

	targets, err := d.Discover(context.Background())
	require.NoError(t, err)
	require.Len(t, targets, 1)
	require.NotNil(t, targets[0].MatchedRule)
	require.Equal(t, "postgresql", targets[0].MatchedRule.ReceiverConfig)
	require.Equal(t, "postgres:15", targets[0].ProcessOrImage)
}

func TestDocker_UnmatchedImageSurfacedAsUnrecognized(t *testing.T) {
	t.Parallel()
	container := dockerContainer{Image: "my-custom-app:latest"}
	container.Ports = []struct {
		PrivatePort int    `json:"PrivatePort"`
		PublicPort  int    `json:"PublicPort"`
		Type        string `json:"Type"`
	}{{PrivatePort: 8080, PublicPort: 8080, Type: "tcp"}}

	d := &DockerDiscoverer{lister: fakeContainerLister{containers: []dockerContainer{container}}}

	targets, err := d.Discover(context.Background())
	require.NoError(t, err)
	require.Len(t, targets, 1)
	require.Nil(t, targets[0].MatchedRule)
}

func TestDocker_ContainerWithNoExposedPorts_Skipped(t *testing.T) {
	t.Parallel()
	d := &DockerDiscoverer{lister: fakeContainerLister{containers: []dockerContainer{{Image: "no-ports:latest"}}}}

	targets, err := d.Discover(context.Background())
	require.NoError(t, err)
	require.Empty(t, targets)
}

func TestDocker_ListerErrorPropagates(t *testing.T) {
	t.Parallel()
	d := &DockerDiscoverer{lister: fakeContainerLister{err: errors.New("docker socket unreachable")}}

	_, err := d.Discover(context.Background())
	require.Error(t, err)
}

func TestDocker_Name(t *testing.T) {
	t.Parallel()
	require.Equal(t, "docker", NewDockerDiscoverer(nil).Name())
}
