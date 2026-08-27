package discovery

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakePodLister struct {
	pods []k8sPod
	err  error
}

func (f fakePodLister) ListPods(_ context.Context) ([]k8sPod, error) {
	return f.pods, f.err
}

func TestK8s_AttachesPodResourceAttributes(t *testing.T) {
	t.Parallel()
	rules := []Rule{{Port: 5432, ProcessMatch: "checkout-db", ReceiverConfig: "postgresql"}}
	d := &K8sDiscoverer{
		rules: rules,
		lister: fakePodLister{pods: []k8sPod{
			{Name: "checkout-db-0", Namespace: "prod", NodeName: "node-a", ContainerPorts: []int{5432}},
		}},
	}

	targets, err := d.Discover(context.Background())
	require.NoError(t, err)
	require.Len(t, targets, 1)

	target := targets[0]
	require.NotNil(t, target.MatchedRule)
	require.Equal(t, "prod", target.ResourceAttributes["k8s.namespace.name"])
	require.Equal(t, "checkout-db-0", target.ResourceAttributes["k8s.pod.name"])
	require.Equal(t, "node-a", target.ResourceAttributes["k8s.node.name"])
}

func TestK8s_UnmatchedPodSurfacedAsUnrecognized(t *testing.T) {
	t.Parallel()
	d := &K8sDiscoverer{
		lister: fakePodLister{pods: []k8sPod{
			{Name: "custom-app-0", Namespace: "default", NodeName: "node-b", ContainerPorts: []int{9999}},
		}},
	}

	targets, err := d.Discover(context.Background())
	require.NoError(t, err)
	require.Len(t, targets, 1)
	require.Nil(t, targets[0].MatchedRule)
	require.Equal(t, "default", targets[0].ResourceAttributes["k8s.namespace.name"])
}

func TestK8s_PodWithNoContainerPorts_Skipped(t *testing.T) {
	t.Parallel()
	d := &K8sDiscoverer{lister: fakePodLister{pods: []k8sPod{{Name: "no-ports-pod"}}}}

	targets, err := d.Discover(context.Background())
	require.NoError(t, err)
	require.Empty(t, targets)
}

func TestK8s_ListerErrorPropagates(t *testing.T) {
	t.Parallel()
	d := &K8sDiscoverer{lister: fakePodLister{err: errors.New("k8s api unreachable")}}

	_, err := d.Discover(context.Background())
	require.Error(t, err)
}

func TestK8s_Name(t *testing.T) {
	t.Parallel()
	require.Equal(t, "k8s", NewK8sDiscoverer(nil).Name())
}

func TestNewInClusterPodLister_NotInCluster_ReturnsUnavailable(t *testing.T) {
	// Not t.Parallel(): t.Setenv forbids it.
	t.Setenv("KUBERNETES_SERVICE_HOST", "")
	t.Setenv("KUBERNETES_SERVICE_PORT", "")

	lister := newInClusterPodLister()
	pods, err := lister.ListPods(context.Background())
	require.NoError(t, err)
	require.Empty(t, pods)
}
