package integration

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/traefik/traefik/v2/integration/try"
)

// ErrorPagesSuite test suites.
type ErrorPagesSuite struct {
	BaseSuite
	ErrorPageIP string
	BackendIP   string
}

func TestErrorPagesSuite(t *testing.T) {
	suite.Run(t, new(ErrorPagesSuite))
}

func (s *ErrorPagesSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()

	s.createComposeProject("error_pages")
	s.composeUp()

	s.ErrorPageIP = s.getComposeServiceIP("whoamierror")
	s.BackendIP = s.getComposeServiceIP("whoamiservice")
}

func (s *ErrorPagesSuite) TearDownSuite() {
	s.BaseSuite.TearDownSuite()
}

func (s *ErrorPagesSuite) TestSimpleConfiguration() {
	file := s.adaptFile("fixtures/error_pages/simple.toml", struct {
		Server1 string
		Server2 string
	}{"http://" + s.BackendIP + ":80", s.ErrorPageIP})

	s.traefikCmd(withConfigFile(file))

	frontendReq, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:8080", nil)
	require.NoError(s.T(), err)
	frontendReq.Host = "test.local"

	err = try.Request(frontendReq, 2*time.Second, try.BodyContains("service"))
	require.NoError(s.T(), err)
}

func (s *ErrorPagesSuite) TestIgnoreBackendErrors() {
	file := s.adaptFile("fixtures/error_pages/ignore_backend_errors.toml", struct {
		Server1 string
		Server2 string
	}{"http://" + s.BackendIP + ":80", s.ErrorPageIP})

	s.traefikCmd(withConfigFile(file))

	frontendReq, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:8080", nil)
	require.NoError(s.T(), err)
	frontendReq.Host = "test.local"

	// The error pages is configured to catch 200 Status Ok responses,
	// checking that the response is from the service shows that the error page middleware do not catch the service "errors".
	err = try.Request(frontendReq, 2*time.Second, try.BodyContains("service"))
	require.NoError(s.T(), err)
}

func (s *ErrorPagesSuite) TestIgnoreBackendErrors_butNotTraefikOnes_misconfigurationInService() {
	file := s.adaptFile("fixtures/error_pages/ignore_backend_errors.toml", struct {
		Server1 string
		Server2 string
	}{"http://" + s.BackendIP + ":80", s.ErrorPageIP})

	s.traefikCmd(withConfigFile(file))

	frontendReq, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:8080", nil)
	require.NoError(s.T(), err)
	frontendReq.Host = "test2.local"

	err = try.Request(frontendReq, 2*time.Second, try.BodyContains("errorpage"))
	require.NoError(s.T(), err)
}

func (s *ErrorPagesSuite) TestIgnoreBackendErrors_butNotTraefikOnes_emptyService() {
	file := s.adaptFile("fixtures/error_pages/ignore_backend_errors.toml", struct {
		Server1 string
		Server2 string
	}{"http://" + s.BackendIP + ":80", s.ErrorPageIP})

	s.traefikCmd(withConfigFile(file))

	frontendReq, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:8080", nil)
	require.NoError(s.T(), err)
	frontendReq.Host = "test3.local"

	err = try.Request(frontendReq, 2*time.Second, try.BodyContains("errorpage"))
	require.NoError(s.T(), err)
}

func (s *ErrorPagesSuite) TestErrorPage() {
	// error.toml contains a mis-configuration of the backend host
	file := s.adaptFile("fixtures/error_pages/error.toml", struct {
		Server1 string
		Server2 string
	}{s.BackendIP, s.ErrorPageIP})

	s.traefikCmd(withConfigFile(file))

	frontendReq, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:8080", nil)
	require.NoError(s.T(), err)
	frontendReq.Host = "test.local"

	err = try.Request(frontendReq, 2*time.Second, try.BodyContains("errorpage"))
	require.NoError(s.T(), err)
}

func (s *ErrorPagesSuite) TestErrorPageFlush() {
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Add("Transfer-Encoding", "chunked")
		rw.WriteHeader(http.StatusInternalServerError)
		_, _ = rw.Write([]byte("KO"))
	}))

	file := s.adaptFile("fixtures/error_pages/simple.toml", struct {
		Server1 string
		Server2 string
	}{srv.URL, s.ErrorPageIP})

	s.traefikCmd(withConfigFile(file))

	frontendReq, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:8080", nil)
	require.NoError(s.T(), err)
	frontendReq.Host = "test.local"

	err = try.Request(frontendReq, 2*time.Second,
		try.BodyContains("errorpage"),
		try.HasHeaderValue("Content-Type", "text/plain; charset=utf-8", true),
	)
	require.NoError(s.T(), err)
}
