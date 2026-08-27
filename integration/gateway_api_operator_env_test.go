//go:build gatewayAPIConformance

package integration

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/exec"
	"github.com/testcontainers/testcontainers-go/modules/k3s"
	"github.com/testcontainers/testcontainers-go/network"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	ktypes "k8s.io/apimachinery/pkg/types"
	kclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatev1 "sigs.k8s.io/gateway-api/apis/v1"
	gatev1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gatev1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
	"sigs.k8s.io/gateway-api/conformance"
	v1 "sigs.k8s.io/gateway-api/conformance/apis/v1"
	"sigs.k8s.io/gateway-api/conformance/tests"
	"sigs.k8s.io/gateway-api/conformance/utils/config"
	ksuite "sigs.k8s.io/gateway-api/conformance/utils/suite"
	"sigs.k8s.io/gateway-api/pkg/features"
	"sigs.k8s.io/yaml"
)

const (
	operatorImage      = "traefik/operator:latest"
	operatorNamespace  = "traefik-operator-system"
	operatorDeployment = "deployments/traefik-operator"
)

// operatorConformanceEnv is the cluster the operator conformance suites run
// against: a k3s node with the operator installed, and a load balancer
// assigning the addresses the operator asks for. It is shared by the suites
// covering the deployment topologies the operator provisions.
type operatorConformanceEnv struct {
	k3sContainer *k3s.K3sContainer
	kubeClient   client.Client
	restConfig   *rest.Config
	clientSet    *kclientset.Clientset
	loadBalancer *nodeLoadBalancer

	cancelLoadBalancer context.CancelFunc
}

// setup starts the cluster, installs the given manifests on top of the Gateway
// API CRDs and the operator, and side loads the Traefik and operator images.
func (e *operatorConformanceEnv) setup(ctx context.Context, t *testing.T, net *testcontainers.DockerNetwork, manifests ...string) {
	t.Helper()

	provider, err := testcontainers.ProviderDocker.GetProvider()
	require.NoError(t, err)

	// Ensure images are available locally.
	images, err := provider.ListImages(ctx)
	require.NoError(t, err)

	for _, image := range []string{traefikImage, operatorImage} {
		if !slices.ContainsFunc(images, func(img testcontainers.ImageInfo) bool {
			return img.Name == image
		}) {
			t.Fatalf("Image %s is not present", image)
		}
	}

	options := []testcontainers.ContainerCustomizer{
		// The k3s service load balancer exposes Services through host ports,
		// which a single node cannot do for the several port 80 Services the
		// operator provisions. nodeLoadBalancer assigns their addresses.
		testcontainers.WithCmdArgs("--disable=servicelb"),
		k3s.WithManifest("./fixtures/gateway-api-conformance/00-experimental-v1.6.1.yml"),
		k3s.WithManifest("./fixtures/gateway-api-conformance/01-operator.yml"),
		network.WithNetwork(nil, net),
	}
	for _, manifest := range manifests {
		options = append(options, k3s.WithManifest(manifest))
	}

	e.k3sContainer, err = k3s.Run(ctx, k3sImage, options...)
	require.NoError(t, err)

	require.NoError(t, e.k3sContainer.LoadImages(ctx, traefikImage, operatorImage))

	exitCode, _, err := e.k3sContainer.Exec(ctx, []string{"kubectl", "wait", "-n", operatorNamespace, operatorDeployment, "--for=condition=Available", "--timeout=120s"})
	if err != nil || exitCode > 0 {
		t.Fatalf("Operator pod is not ready: %v", err)
	}

	kubeConfigYaml, err := e.k3sContainer.GetKubeConfig(ctx)
	require.NoError(t, err)

	e.restConfig, err = clientcmd.RESTConfigFromKubeConfig(kubeConfigYaml)
	require.NoError(t, err)

	e.kubeClient, err = client.New(e.restConfig, client.Options{})
	require.NoError(t, err)

	e.clientSet, err = kclientset.NewForConfig(e.restConfig)
	require.NoError(t, err)

	require.NoError(t, gatev1alpha2.Install(e.kubeClient.Scheme()))
	require.NoError(t, gatev1beta1.Install(e.kubeClient.Scheme()))
	require.NoError(t, gatev1.Install(e.kubeClient.Scheme()))
	require.NoError(t, apiextensionsv1.AddToScheme(e.kubeClient.Scheme()))

	e.loadBalancer, err = newNodeLoadBalancer(ctx, e.k3sContainer, e.clientSet)
	require.NoError(t, err)

	// The suite context is done before the teardown runs, so the load balancer
	// gets its own.
	lbCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	e.cancelLoadBalancer = cancel
	e.loadBalancer.Start(lbCtx)
}

// teardown dumps the cluster state of a failed run and stops the cluster.
func (e *operatorConformanceEnv) teardown(ctx context.Context, t *testing.T) {
	t.Helper()

	if e.cancelLoadBalancer != nil {
		e.cancelLoadBalancer()
	}

	if t.Failed() || *showLog {
		k3sLogs, err := e.k3sContainer.Logs(ctx)
		if err == nil {
			if res, err := io.ReadAll(k3sLogs); err == nil {
				t.Log(string(res))
			}
		}

		e.logCommand(ctx, t, "kubectl", "logs", "-n", operatorNamespace, operatorDeployment)

		// The data planes are where a routing failure shows up.
		e.logCommand(ctx, t, "kubectl", "get", "gateways,pods,services", "--all-namespaces")
		e.logCommand(ctx, t, "kubectl", "logs", "--all-namespaces", "--selector", "app.kubernetes.io/managed-by=traefik-operator",
			"--prefix", "--tail=200", "--max-log-requests=30")
	}

	require.NoError(t, e.k3sContainer.Terminate(ctx))
}

// logCommand runs a command in the k3s container and logs its output.
func (e *operatorConformanceEnv) logCommand(ctx context.Context, t *testing.T, command ...string) {
	t.Helper()

	exitCode, result, err := e.k3sContainer.Exec(ctx, command, exec.Multiplexed())
	if err != nil {
		t.Logf("%v: %v", command, err)
		return
	}

	output, err := io.ReadAll(result)
	if err != nil {
		t.Logf("%v: %v", command, err)
		return
	}

	t.Logf("%v (exit code %d):\n%s", command, exitCode, output)
}

// supportedFeatures reads the feature set the operator publishes on the
// GatewayClass status. The operator owns that status here (the data plane runs
// with disableGatewayClassStatus), and it advertises the features it implements
// itself on top of the data plane ones, so the report cannot drift from what
// the implementation claims.
func (e *operatorConformanceEnv) supportedFeatures(ctx context.Context) ([]features.FeatureName, error) {
	var gatewayClass gatev1.GatewayClass
	if err := e.kubeClient.Get(ctx, ktypes.NamespacedName{Name: "traefik"}, &gatewayClass); err != nil {
		return nil, fmt.Errorf("getting the traefik GatewayClass: %w", err)
	}

	if len(gatewayClass.Status.SupportedFeatures) == 0 {
		return nil, errors.New("the traefik GatewayClass publishes no supported feature")
	}

	supportedFeatures := make([]features.FeatureName, 0, len(gatewayClass.Status.SupportedFeatures))
	for _, feature := range gatewayClass.Status.SupportedFeatures {
		supportedFeatures = append(supportedFeatures, features.FeatureName(feature.Name))
	}

	return supportedFeatures, nil
}

// runConformance runs the conformance suite against the environment and writes
// the report under the mode it names the topology by.
func (e *operatorConformanceEnv) runConformance(ctx context.Context, t *testing.T, mode string, skipTests []string) {
	t.Helper()

	supportedFeatures, err := e.supportedFeatures(ctx)
	require.NoError(t, err)

	// Provisioning a data plane adds a Deployment rollout to every Gateway
	// reconciliation, so the timeouts are longer than for the statically
	// deployed suite. They stay shortened for a status Traefik will never
	// report to fail before the test binary timeout, which would discard the
	// whole run and its report.
	timeoutConfig := config.DefaultTimeoutConfig()
	timeoutConfig.GatewayMustHaveAddress = 120 * time.Second
	timeoutConfig.GatewayMustHaveCondition = 120 * time.Second
	timeoutConfig.GWCMustBeAccepted = 60 * time.Second
	timeoutConfig.ListenerSetMustHaveCondition = 120 * time.Second
	timeoutConfig.NamespacesMustBeReady = 300 * time.Second

	cSuite, err := ksuite.NewConformanceTestSuite(ksuite.ConformanceOptions{
		Client:     e.kubeClient,
		Clientset:  e.clientSet,
		RestConfig: e.restConfig,
		ManifestFS: []fs.FS{&conformance.Manifests},
		ConfigurableOptions: ksuite.ConfigurableOptions{
			GatewayClassName:           "traefik",
			Debug:                      true,
			CleanupBaseResources:       true,
			CleanupTestResources:       true,
			TimeoutConfig:              timeoutConfig,
			EnableAllSupportedFeatures: false,
			RunTest:                    *gatewayAPIConformanceRunTest,
			Mode:                       mode,
			Implementation: v1.Implementation{
				Organization: "traefik",
				Project:      "traefik",
				URL:          "https://traefik.io/",
				Version:      *traefikVersion,
				Contact:      []string{"@traefik/maintainers"},
			},
			ConformanceProfiles: []ksuite.ConformanceProfileName{
				ksuite.GatewayHTTPConformanceProfileName,
				ksuite.GatewayGRPCConformanceProfileName,
				ksuite.GatewayTLSConformanceProfileName,
			},
			SupportedFeatures: supportedFeatures,
			SkipTests:         skipTests,
			UsableNetworkAddresses: []gatev1.GatewaySpecAddress{{
				Type:  new(gatev1.IPAddressType),
				Value: e.loadBalancer.StaticAddress(),
			}},
			UnusableNetworkAddresses: []gatev1.GatewaySpecAddress{{
				Type:  new(gatev1.IPAddressType),
				Value: nodeLBUnusableAddress,
			}},
		},
	})
	require.NoError(t, err)

	cSuite.Setup(t, tests.ConformanceTests)

	err = cSuite.Run(t, tests.ConformanceTests)
	require.NoError(t, err)

	report, err := cSuite.Report()
	require.NoError(t, err, "failed generating conformance report")

	// Ordering profile reports for the serialized report to be comparable.
	slices.SortFunc(report.ProfileReports, func(a, b v1.ProfileReport) int {
		return strings.Compare(a.Name, b.Name)
	})

	rawReport, err := yaml.Marshal(report)
	require.NoError(t, err)
	t.Logf("Conformance report:\n%s", string(rawReport))

	require.NoError(t, os.MkdirAll("./gateway-api-conformance-reports/"+report.GatewayAPIVersion, 0o755))
	outFile := filepath.Join("gateway-api-conformance-reports/"+report.GatewayAPIVersion, fmt.Sprintf("%s-%s-%s-report.yaml", report.GatewayAPIChannel, report.Version, report.Mode))
	require.NoError(t, os.WriteFile(outFile, rawReport, 0o600))
	t.Logf("Report written to: %s", outFile)
}
