//go:build gatewayAPIConformance

package integration

import (
	"testing"

	"github.com/stretchr/testify/suite"
	klog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// conformanceModeIsolated names the deployment topology in the generated
// report: a single Traefik data plane shared by every Gateway of the
// GatewayClass, each Gateway answering at the address of a Service of its own.
const conformanceModeIsolated = "gateway-isolation"

// GatewayAPIIsolatedConformanceSuite runs the Gateway API conformance suite
// against a single Traefik data plane serving every Gateway, isolated by the
// address each connection was accepted on. It is the topology Cilium uses: one
// data plane, one Service per Gateway.
type GatewayAPIIsolatedConformanceSuite struct {
	BaseSuite

	env operatorConformanceEnv
}

func TestGatewayAPIIsolatedConformanceSuite(t *testing.T) {
	suite.Run(t, new(GatewayAPIIsolatedConformanceSuite))
}

func (s *GatewayAPIIsolatedConformanceSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()

	// Avoid panic.
	klog.SetLogger(zap.New())

	s.env.setup(s.T().Context(), s.T(), s.network, "./fixtures/gateway-api-conformance/03-gatewayclass-isolated.yml")
}

func (s *GatewayAPIIsolatedConformanceSuite) TearDownSuite() {
	s.env.teardown(s.T().Context(), s.T())
	s.BaseSuite.TearDownSuite()
}

func (s *GatewayAPIIsolatedConformanceSuite) TestK8sGatewayAPIIsolatedConformance() {
	s.env.runConformance(s.T().Context(), s.T(), conformanceModeIsolated, nil)
}
