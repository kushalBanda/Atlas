package discovery

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

const (
	inClusterTokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	inClusterCAPath    = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
)

// k8sPod is the subset of a Kubernetes Pod this discoverer needs.
type k8sPod struct {
	Name           string
	Namespace      string
	NodeName       string
	ContainerPorts []int
}

// podLister lists pods across the cluster. Abstracted so K8sDiscoverer is
// testable without a real cluster.
type podLister interface {
	ListPods(ctx context.Context) ([]k8sPod, error)
}

// K8sDiscoverer finds pods across the cluster and matches their exposed
// container ports against curated rules, attaching pod/namespace/node as
// resource attributes on each Target it reports.
type K8sDiscoverer struct {
	rules  []Rule
	lister podLister
}

// NewK8sDiscoverer returns a K8s Discoverer using in-cluster
// authentication (service account token + CA). Outside a cluster, Discover
// reports zero targets rather than erroring — see docs/design-considerations.md.
func NewK8sDiscoverer(rules []Rule) *K8sDiscoverer {
	return &K8sDiscoverer{rules: rules, lister: newInClusterPodLister()}
}

// Name implements Discoverer.
func (d *K8sDiscoverer) Name() string { return "k8s" }

// Discover implements Discoverer.
func (d *K8sDiscoverer) Discover(ctx context.Context) ([]Target, error) {
	pods, err := d.lister.ListPods(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing k8s pods: %w", err)
	}

	var targets []Target
	for _, p := range pods {
		for _, port := range p.ContainerPorts {
			targets = append(targets, Target{
				Host:           p.Name,
				Port:           port,
				ProcessOrImage: p.Name,
				MatchedRule:    matchRule(d.rules, port, p.Name),
				ResourceAttributes: map[string]string{
					"k8s.pod.name":       p.Name,
					"k8s.namespace.name": p.Namespace,
					"k8s.node.name":      p.NodeName,
				},
			})
		}
	}
	return targets, nil
}

// inClusterPodLister lists pods via the Kubernetes API server using
// in-cluster service account credentials. If those credentials aren't
// present (not running in-cluster), available is false and ListPods
// returns an empty, error-free result.
type inClusterPodLister struct {
	available  bool
	baseURL    string
	token      string
	httpClient *http.Client
}

func newInClusterPodLister() *inClusterPodLister {
	host := os.Getenv("KUBERNETES_SERVICE_HOST")
	port := os.Getenv("KUBERNETES_SERVICE_PORT")
	if host == "" || port == "" {
		return &inClusterPodLister{available: false}
	}

	token, err := os.ReadFile(inClusterTokenPath)
	if err != nil {
		return &inClusterPodLister{available: false}
	}

	caCert, err := os.ReadFile(inClusterCAPath)
	if err != nil {
		return &inClusterPodLister{available: false}
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caCert) {
		return &inClusterPodLister{available: false}
	}

	return &inClusterPodLister{
		available: true,
		baseURL:   fmt.Sprintf("https://%s:%s", host, port),
		token:     string(token),
		httpClient: &http.Client{
			Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool}},
			Timeout:   10 * time.Second,
		},
	}
}

// podListResponse is the subset of the Kubernetes PodList API response
// this discoverer needs.
type podListResponse struct {
	Items []struct {
		Metadata struct {
			Name      string `json:"name"`
			Namespace string `json:"namespace"`
		} `json:"metadata"`
		Spec struct {
			NodeName   string `json:"nodeName"`
			Containers []struct {
				Ports []struct {
					ContainerPort int `json:"containerPort"`
				} `json:"ports"`
			} `json:"containers"`
		} `json:"spec"`
	} `json:"items"`
}

func (l *inClusterPodLister) ListPods(ctx context.Context) ([]k8sPod, error) {
	if !l.available {
		return nil, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.baseURL+"/api/v1/pods", nil)
	if err != nil {
		return nil, fmt.Errorf("building k8s api request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+l.token)

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling k8s api at %s: %w", l.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("k8s api returned status %d", resp.StatusCode)
	}

	var list podListResponse
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, fmt.Errorf("decoding k8s api response: %w", err)
	}

	pods := make([]k8sPod, 0, len(list.Items))
	for _, item := range list.Items {
		var ports []int
		for _, c := range item.Spec.Containers {
			for _, p := range c.Ports {
				ports = append(ports, p.ContainerPort)
			}
		}
		pods = append(pods, k8sPod{
			Name:           item.Metadata.Name,
			Namespace:      item.Metadata.Namespace,
			NodeName:       item.Spec.NodeName,
			ContainerPorts: ports,
		})
	}
	return pods, nil
}
