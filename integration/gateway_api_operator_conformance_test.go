//go:build gatewayAPIConformance

package integration

import (
	"testing"

	"github.com/stretchr/testify/suite"
	klog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// conformanceModeOperator names the deployment topology in the generated
// report: one Traefik data plane provisioned per Gateway, where the default
// mode is the statically deployed, cluster-wide Traefik of
// GatewayAPIConformanceSuite.
const conformanceModeOperator = "operator"

// GatewayAPIOperatorConformanceSuite runs the Gateway API conformance suite
// against a Traefik data plane provisioned per Gateway by the Traefik Gateway
// API operator, instead of a single statically deployed instance.
type GatewayAPIOperatorConformanceSuite struct {
	BaseSuite

	env operatorConformanceEnv
}

func TestGatewayAPIOperatorConformanceSuite(t *testing.T) {
	suite.Run(t, new(GatewayAPIOperatorConformanceSuite))
}

func (s *GatewayAPIOperatorConformanceSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()

	// Avoid panic.
	klog.SetLogger(zap.New())

	s.env.setup(s.T().Context(), s.T(), s.network, "./fixtures/gateway-api-conformance/02-gatewayclass.yml")
}

func (s *GatewayAPIOperatorConformanceSuite) TearDownSuite() {
	s.env.teardown(s.T().Context(), s.T())
	s.BaseSuite.TearDownSuite()
}

func (s *GatewayAPIOperatorConformanceSuite) TestK8sGatewayAPIOperatorConformance() {
	s.env.runConformance(s.T().Context(), s.T(), conformanceModeOperator, nil)
}
