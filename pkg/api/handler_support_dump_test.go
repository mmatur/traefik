package api

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	"github.com/traefik/traefik/v3/pkg/config/runtime"
	"github.com/traefik/traefik/v3/pkg/config/static"
)

func TestHandler_SupportDump(t *testing.T) {
	testCases := []struct {
		desc       string
		path       string
		confStatic static.Configuration
		confDyn    runtime.Configuration
		validate   func(t *testing.T, files map[string][]byte)
	}{
		{
			desc:       "empty configurations",
			path:       "/api/support-dump",
			confStatic: static.Configuration{API: &static.API{}, Global: &static.Global{}},
			confDyn:    runtime.Configuration{},
			validate: func(t *testing.T, files map[string][]byte) {
				t.Helper()

				require.Contains(t, files, "static-config.json")
				require.Contains(t, files, "runtime-config.json")
				require.Contains(t, files, "version.json")

				// Verify version.json contains version information
				assert.Contains(t, string(files["version.json"]), `"version":"dev"`)

				assert.JSONEq(t, `{"global":{},"api":{}}`, string(files["static-config.json"]))
				assert.Equal(t, `{}`, string(files["runtime-config.json"]))
			},
		},
		{
			desc: "with configuration data",
			path: "/api/support-dump",
			confStatic: static.Configuration{
				API:    &static.API{},
				Global: &static.Global{},
				EntryPoints: map[string]*static.EntryPoint{
					"web": {Address: ":80"},
				},
			},
			confDyn: runtime.Configuration{
				Services: map[string]*runtime.ServiceInfo{
					"test-service": {
						Service: &dynamic.Service{
							LoadBalancer: &dynamic.ServersLoadBalancer{
								Servers: []dynamic.Server{{URL: "http://127.0.0.1:8080"}},
							},
						},
						Status: runtime.StatusEnabled,
					},
				},
			},
			validate: func(t *testing.T, files map[string][]byte) {
				t.Helper()

				require.Contains(t, files, "static-config.json")
				require.Contains(t, files, "runtime-config.json")
				require.Contains(t, files, "version.json")

				// Verify version.json contains version information
				assert.Contains(t, string(files["version.json"]), `"version":"dev"`)

				// Verify static config contains entry points
				assert.Contains(t, string(files["static-config.json"]), `"entryPoints":{"web":{"address":"xxxx","http":{}`)

				// Verify runtime config contains services
				assert.Contains(t, string(files["runtime-config.json"]), `"services":`)
				assert.Contains(t, string(files["runtime-config.json"]), `"test-service"`)
			},
		},
		{
			desc: "with sensitive data redacted",
			path: "/api/support-dump",
			confStatic: static.Configuration{
				API:    &static.API{},
				Global: &static.Global{},
			},
			confDyn: runtime.Configuration{
				Services: map[string]*runtime.ServiceInfo{
					"svc@file": {
						Service: &dynamic.Service{
							LoadBalancer: &dynamic.ServersLoadBalancer{
								Servers: []dynamic.Server{
									{URL: "http://user:secretpassword@backend.local:8080"},
								},
							},
						},
						Status: runtime.StatusEnabled,
					},
				},
				Middlewares: map[string]*runtime.MiddlewareInfo{
					"auth@file": {
						Middleware: &dynamic.Middleware{
							BasicAuth: &dynamic.BasicAuth{
								Users: dynamic.Users{"admin:$apr1$examplehash"},
							},
						},
						Status: runtime.StatusEnabled,
					},
					"digest@file": {
						Middleware: &dynamic.Middleware{
							DigestAuth: &dynamic.DigestAuth{
								Users: dynamic.Users{"admin:traefik:digestedhash"},
							},
						},
						Status: runtime.StatusEnabled,
					},
				},
			},
			validate: func(t *testing.T, files map[string][]byte) {
				t.Helper()

				require.Contains(t, files, "runtime-config.json")
				runtimeConfig := string(files["runtime-config.json"])

				// Verify sensitive data is NOT present in the runtime config
				assert.NotContains(t, runtimeConfig, "secretpassword", "URL with embedded credentials should be redacted")
				assert.NotContains(t, runtimeConfig, "user:secretpassword", "URL with embedded credentials should be redacted")
				assert.NotContains(t, runtimeConfig, "backend.local", "URLs should be anonymized")
				assert.NotContains(t, runtimeConfig, "$apr1$examplehash", "BasicAuth password hashes should be redacted")
				assert.NotContains(t, runtimeConfig, "digestedhash", "DigestAuth password hashes should be redacted")

				// Verify the structure is still present (services and middlewares exist)
				assert.Contains(t, runtimeConfig, `"services"`)
				assert.Contains(t, runtimeConfig, `"middlewares"`)
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			handler := New(test.confStatic, &test.confDyn)
			server := httptest.NewServer(handler.createRouter())

			resp, err := http.DefaultClient.Get(server.URL + test.path)
			require.NoError(t, err)

			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, "application/gzip", resp.Header.Get("Content-Type"))
			assert.Equal(t, `attachment; filename=support-dump.tar.gz`, resp.Header.Get("Content-Disposition"))

			// Extract and validate the tar.gz contents.
			files, err := extractTarGz(resp.Body)
			require.NoError(t, err)

			test.validate(t, files)
		})
	}
}

// extractTarGz reads a tar.gz archive and returns a map of filename to contents
func extractTarGz(r io.Reader) (map[string][]byte, error) {
	files := make(map[string][]byte)

	gzr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}

		if header.Typeflag != tar.TypeReg {
			continue
		}

		contents, err := io.ReadAll(tr)
		if err != nil {
			return nil, err
		}

		files[header.Name] = contents
	}

	return files, nil
}
