package healthcheck

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	gokitmetrics "github.com/go-kit/kit/metrics"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/config/runtime"
	"github.com/traefik/traefik/v2/pkg/log"
	"github.com/traefik/traefik/v2/pkg/metrics"
	"github.com/traefik/traefik/v2/pkg/safe"
	"github.com/vulcand/oxy/roundrobin"
)

const (
	serverUp    = "UP"
	serverDown  = "DOWN"
	serverDrain = "DRAIN"
)

var (
	singleton *HealthCheck
	once      sync.Once
)

// Balancer is the set of operations required to manage the list of servers in a load-balancer.
type Balancer interface {
	Servers() []*url.URL
	RemoveServer(u *url.URL) error
	UpsertServer(u *url.URL, options ...roundrobin.ServerOption) error
}

// BalancerHandler includes functionality for load-balancing management.
type BalancerHandler interface {
	ServeHTTP(w http.ResponseWriter, req *http.Request)
	Balancer
}

// BalancerStatusHandler is an http Handler that does load-balancing,
// and updates its parents of its status.
type BalancerStatusHandler interface {
	BalancerHandler
	StatusUpdater
}

type metricsHealthcheck struct {
	serverUpGauge gokitmetrics.Gauge
}

// Options are the public health check options.
type Options struct {
	Headers         map[string]string
	Hostname        string
	Scheme          string
	Path            string
	Method          string
	Port            int
	FollowRedirects bool
	Transport       http.RoundTripper
	Interval        time.Duration
	Timeout         time.Duration
	LB              Balancer
}

func (opt Options) String() string {
	return fmt.Sprintf("[Hostname: %s Headers: %v Path: %s Method: %s Port: %d Interval: %s Timeout: %s FollowRedirects: %v]", opt.Hostname, opt.Headers, opt.Path, opt.Method, opt.Port, opt.Interval, opt.Timeout, opt.FollowRedirects)
}

type backendURL struct {
	state  string
	url    *url.URL
	weight int
}

// BackendConfig HealthCheck configuration for a backend.
type BackendConfig struct {
	Options
	name string
	urls map[string]backendURL
}

func (b *BackendConfig) newRequest(serverURL *url.URL) (*http.Request, error) {
	u, err := serverURL.Parse(b.Path)
	if err != nil {
		return nil, err
	}

	if len(b.Scheme) > 0 {
		u.Scheme = b.Scheme
	}

	if b.Port != 0 {
		u.Host = net.JoinHostPort(u.Hostname(), strconv.Itoa(b.Port))
	}

	return http.NewRequest(http.MethodGet, u.String(), http.NoBody)
}

// setRequestOptions sets all request options present on the BackendConfig.
func (b *BackendConfig) setRequestOptions(req *http.Request) *http.Request {
	if b.Options.Hostname != "" {
		req.Host = b.Options.Hostname
	}

	for k, v := range b.Options.Headers {
		req.Header.Set(k, v)
	}

	if b.Options.Method != "" {
		req.Method = strings.ToUpper(b.Options.Method)
	}

	return req
}

// HealthCheck struct.
type HealthCheck struct {
	Backends map[string]*BackendConfig
	metrics  metricsHealthcheck
	cancel   context.CancelFunc
}

// SetBackendsConfiguration set backends configuration.
func (hc *HealthCheck) SetBackendsConfiguration(parentCtx context.Context, backends map[string]*BackendConfig) {
	hc.Backends = backends
	if hc.cancel != nil {
		hc.cancel()
	}
	ctx, cancel := context.WithCancel(parentCtx)
	hc.cancel = cancel

	for _, backend := range backends {
		currentBackend := backend
		safe.Go(func() {
			hc.execute(ctx, currentBackend)
		})
	}
}

func (hc *HealthCheck) execute(ctx context.Context, backend *BackendConfig) {
	logger := log.FromContext(ctx)

	logger.Debugf("Initial health check for backend: %q", backend.name)
	hc.checkServersLB(ctx, backend)

	ticker := time.NewTicker(backend.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			logger.Debugf("Stopping current health check goroutines of backend: %s", backend.name)
			return
		case <-ticker.C:
			logger.Debugf("Routine health check refresh for backend: %s", backend.name)
			hc.checkServersLB(ctx, backend)
		}
	}
}

func (hc *HealthCheck) checkServersLB(ctx context.Context, backend *BackendConfig) {
	logger := log.FromContext(ctx)

	if backend.urls == nil {
		backend.urls = make(map[string]backendURL)
	}

	rr, isRR := backend.LB.(*roundrobin.RoundRobin)

	enabledURLs := backend.LB.Servers()
	for _, u := range enabledURLs {
		weight := 1
		if isRR {
			var gotWeight bool
			weight, gotWeight = rr.ServerWeight(u)
			if !gotWeight {
				weight = 1
			}
		}

		if _, found := backend.urls[u.String()]; !found {
			backend.urls[u.String()] = backendURL{state: serverUp, url: u, weight: weight}
		}
	}

	for _, bURL := range backend.urls {
		serverUpMetricValue := float64(1)

		newState, err := checkHealth(bURL.url, backend)
		if err != nil {
			logger.Warnf("Health check failed, Backend: %q URL: %q Weight: %d Reason: %v",
				backend.name, bURL.url.String(), bURL.weight, err)
		}
		if newState == bURL.state {
			if bURL.state == serverDown {
				serverUpMetricValue = 0
			}
			labelValues := []string{"service", backend.name, "url", bURL.url.String()}
			hc.metrics.serverUpGauge.With(labelValues...).Set(serverUpMetricValue)
			continue
		}

		switch newState {
		case serverUp:
			logger.Warnf("Health check up: Returning to server list. Backend: %q URL: %q Weight: %d",
				backend.name, bURL.url.String(), bURL.weight)
			if err = backend.LB.UpsertServer(bURL.url, roundrobin.Weight(bURL.weight)); err != nil {
				logger.Error(err)
			}
		case serverDrain:
			logger.Debugf("Health check in drain mode. Backend: %q URL: %q", backend.name, bURL.url.String())
			if err = backend.LB.UpsertServer(bURL.url, roundrobin.Weight(0)); err != nil {
				logger.Error(err)
			}
		case serverDown:
			logger.Warnf("Health check failed, removing from server list. Backend: %q URL: %q Reason: %v", backend.name, bURL.url.String(), err)
			if err := backend.LB.RemoveServer(bURL.url); err != nil {
				logger.Error(err)
			}
			serverUpMetricValue = 0
		}
		backend.urls[bURL.url.String()] = backendURL{state: newState, url: bURL.url, weight: bURL.weight}

		labelValues := []string{"service", backend.name, "url", bURL.url.String()}
		hc.metrics.serverUpGauge.With(labelValues...).Set(serverUpMetricValue)
	}
}

// GetHealthCheck returns the health check which is guaranteed to be a singleton.
func GetHealthCheck(registry metrics.Registry) *HealthCheck {
	once.Do(func() {
		singleton = newHealthCheck(registry)
	})
	return singleton
}

func newHealthCheck(registry metrics.Registry) *HealthCheck {
	return &HealthCheck{
		Backends: make(map[string]*BackendConfig),
		metrics: metricsHealthcheck{
			serverUpGauge: registry.ServiceServerUpGauge(),
		},
	}
}

// NewBackendConfig Instantiate a new BackendConfig.
func NewBackendConfig(options Options, backendName string) *BackendConfig {
	return &BackendConfig{
		Options: options,
		name:    backendName,
	}
}

// checkHealth returns a nil error in case it was successful and otherwise
// a non-nil error with a meaningful description why the health check failed.
func checkHealth(serverURL *url.URL, backend *BackendConfig) (string, error) {
	req, err := backend.newRequest(serverURL)
	if err != nil {
		return serverDown, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req = backend.setRequestOptions(req)

	client := http.Client{
		Timeout:   backend.Options.Timeout,
		Transport: backend.Options.Transport,
	}

	if !backend.FollowRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return serverDown, fmt.Errorf("HTTP request failed: %w", err)
	}

	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusGone:
		return serverDrain, fmt.Errorf("received draining status code: %d", resp.StatusCode)
	case resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest:
		return serverDown, fmt.Errorf("received error status code: %d", resp.StatusCode)
	}

	return serverUp, nil
}

// StatusUpdater should be implemented by a service that, when its status
// changes (e.g. all if its children are down), needs to propagate upwards (to
// their parent(s)) that change.
type StatusUpdater interface {
	RegisterStatusUpdater(fn func(up bool)) error
}

// NewLBStatusUpdater returns a new LbStatusUpdater.
func NewLBStatusUpdater(bh BalancerHandler, info *runtime.ServiceInfo, hc *dynamic.ServerHealthCheck) *LbStatusUpdater {
	return &LbStatusUpdater{
		BalancerHandler:  bh,
		serviceInfo:      info,
		wantsHealthCheck: hc != nil,
	}
}

// LbStatusUpdater wraps a BalancerHandler and a ServiceInfo,
// so it can keep track of the status of a server in the ServiceInfo.
type LbStatusUpdater struct {
	BalancerHandler
	serviceInfo      *runtime.ServiceInfo // can be nil
	updaters         []func(up bool)
	wantsHealthCheck bool
}

// RegisterStatusUpdater adds fn to the list of hooks that are run when the
// status of the Balancer changes.
// Not thread safe.
func (lb *LbStatusUpdater) RegisterStatusUpdater(fn func(up bool)) error {
	if !lb.wantsHealthCheck {
		return errors.New("healthCheck not enabled in config for this loadbalancer service")
	}

	lb.updaters = append(lb.updaters, fn)
	return nil
}

// RemoveServer removes the given server from the BalancerHandler,
// and updates the status of the server to "DOWN".
func (lb *LbStatusUpdater) RemoveServer(u *url.URL) error {
	// TODO(mpl): when we have the freedom to change the signature of RemoveServer
	// (kinda stuck because of oxy for now), let's pass around a context to improve
	// logging.
	ctx := context.TODO()
	upBefore := len(lb.BalancerHandler.Servers()) > 0
	err := lb.BalancerHandler.RemoveServer(u)
	if err != nil {
		return err
	}
	if lb.serviceInfo != nil {
		lb.serviceInfo.UpdateServerStatus(u.String(), serverDown)
	}
	log.FromContext(ctx).Debugf("child %s now %s", u.String(), serverDown)

	if !upBefore {
		// we were already down, and we still are, no need to propagate.
		log.FromContext(ctx).Debugf("Still %s, no need to propagate", serverDown)
		return nil
	}
	if len(lb.BalancerHandler.Servers()) > 0 {
		// we were up, and we still are, no need to propagate
		log.FromContext(ctx).Debugf("Still %s, no need to propagate", serverUp)
		return nil
	}

	log.FromContext(ctx).Debugf("Propagating new %s status", serverDown)
	for _, fn := range lb.updaters {
		fn(false)
	}
	return nil
}

// UpsertServer adds the given server to the BalancerHandler,
// and updates the status of the server to "UP".
func (lb *LbStatusUpdater) UpsertServer(u *url.URL, options ...roundrobin.ServerOption) error {
	ctx := context.TODO()
	upBefore := len(lb.BalancerHandler.Servers()) > 0
	err := lb.BalancerHandler.UpsertServer(u, options...)
	if err != nil {
		return err
	}

	state := serverUp
	if rr, ok := lb.BalancerHandler.(*roundrobin.RoundRobin); ok {
		if weight, _ := rr.ServerWeight(u); weight == 0 {
			state = serverDrain
		}
	}

	if lb.serviceInfo != nil {
		lb.serviceInfo.UpdateServerStatus(u.String(), state)
	}

	log.FromContext(ctx).Debugf("child %s now %s", u.String(), state)

	if upBefore {
		// we were up, and we still are, no need to propagate
		log.FromContext(ctx).Debugf("Still %s, no need to propagate", state)
		return nil
	}

	log.FromContext(ctx).Debugf("Propagating new %s status", state)
	for _, fn := range lb.updaters {
		fn(true)
	}
	return nil
}

// Balancers is a list of Balancers(s) that implements the Balancer interface.
type Balancers []Balancer

// Servers returns the deduplicated server URLs from all the Balancer.
// Note that the deduplication is only possible because all the underlying
// balancers are of the same kind (the oxy implementation).
// The comparison property is the same as the one found at:
// https://github.com/vulcand/oxy/blob/fb2728c857b7973a27f8de2f2190729c0f22cf49/roundrobin/rr.go#L347.
func (b Balancers) Servers() []*url.URL {
	seen := make(map[string]struct{})

	var servers []*url.URL
	for _, lb := range b {
		for _, server := range lb.Servers() {
			key := serverKey(server)
			if _, ok := seen[key]; ok {
				continue
			}

			servers = append(servers, server)
			seen[key] = struct{}{}
		}
	}

	return servers
}

// RemoveServer removes the given server from all the Balancer,
// and updates the status of the server to "DOWN".
func (b Balancers) RemoveServer(u *url.URL) error {
	for _, lb := range b {
		if err := lb.RemoveServer(u); err != nil {
			return err
		}
	}
	return nil
}

// UpsertServer adds the given server to all the Balancer,
// and updates the status of the server to "UP".
func (b Balancers) UpsertServer(u *url.URL, options ...roundrobin.ServerOption) error {
	for _, lb := range b {
		if err := lb.UpsertServer(u, options...); err != nil {
			return err
		}
	}
	return nil
}

func serverKey(u *url.URL) string {
	return u.Path + u.Host + u.Scheme
}
