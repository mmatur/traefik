package dynamic

// +k8s:deepcopy-gen=true

// Plugin holds TraefikEE-specific Middleware configuration.
type Plugin struct {
	LDAPAuth           *LDAPAuth               `json:"ldapAuth,omitempty" toml:"ldapAuth,omitempty" yaml:"ldapAuth,omitempty"`
	JWTAuth            *JWTAuth                `json:"jwtAuth,omitempty" toml:"jwtAuth,omitempty" yaml:"jwtAuth,omitempty"`
	OAuthIntrospection *OAuthIntrospection     `json:"oAuthIntrospection,omitempty" toml:"oAuthIntrospection,omitempty" yaml:"oAuthIntrospection,omitempty"`
	InFlightReq        *DistributedInFlightReq `json:"inFlightReq,omitempty" toml:"inFlightReq,omitempty" yaml:"inFlightReq,omitempty"`
	RateLimit          *DistributedRateLimit   `json:"rateLimit,omitempty" toml:"rateLimit,omitempty" yaml:"rateLimit,omitempty"`
	ForceCase          *ForceCase              `json:"forceCase,omitempty" toml:"forceCase,omitempty" yaml:"forceCase,omitempty"`
}

// +k8s:deepcopy-gen=true

// LDAPAuth holds the LDAP Middleware configuration.
type LDAPAuth struct {
	// Source is the name of the authentication source this middleware should use.
	Source string `json:"source,omitempty" toml:"source,omitempty" yaml:"source,omitempty"`

	// BaseDN is the base domain name that should be used for bind and search queries.
	BaseDN string `json:"baseDN,omitempty" toml:"baseDN,omitempty" yaml:"baseDN,omitempty"`
	// Attribute is the LDAP object attribute used to form a bind DN when sending bind queries:
	// <Attribute>=<Username>,<BaseDN>
	// where the Username is extracted from the Authorization header in the request.
	Attribute string `json:"attribute,omitempty" toml:"attribute,omitempty" yaml:"attribute,omitempty"`
	// SearchFilter can be set to enable search and bind mode. When set, this value will be used to filter the
	// results of a search query. Example of a search query: (&(objectClass=inetOrgPerson)(gidNumber=500)(uid=%s)).
	// "%s" can be used as a placeholder that will be replaced by the Username.
	SearchFilter string `json:"searchFilter,omitempty" toml:"searchFilter,omitempty" yaml:"searchFilter,omitempty"`

	// ForwardUsername determines whether a "Username" header should be added to the request, containing the value of the username used
	// to authenticate to the LDAP server.
	ForwardUsername bool `json:"forwardUsername,omitempty" toml:"forwardUsername,omitempty" yaml:"forwardUsername,omitempty"`
	// ForwardUsernameHeader sets the name of the header to use to forward the username.
	ForwardUsernameHeader string `json:"forwardUsernameHeader,omitempty" toml:"forwardUsernameHeader,omitempty" yaml:"forwardUsernameHeader,omitempty"`
	// ForwardAuthorization determines whether the "Authorization" header should be forwarded or stripped from the request.
	ForwardAuthorization bool `json:"forwardAuthorization,omitempty" toml:"forwardAuthorization,omitempty" yaml:"forwardAuthorization,omitempty"`
	// WWWAuthenticateHeader determines whether a "WWW-Authenticate" header should be added to the request if it fails with a 401 Unauthorized status code
	// in order to instruct the User-Agent he should try to authenticate. See https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/WWW-Authenticate
	// for more information.
	WWWAuthenticateHeader bool `json:"wwwAuthenticateHeader,omitempty" toml:"wwwAuthenticateHeader,omitempty" yaml:"wwwAuthenticateHeader,omitempty"`
	// WWWAuthenticateHeaderRealm sets a realm in the "WWW-Authenticate" header.
	WWWAuthenticateHeaderRealm string `json:"wwwAuthenticateHeaderRealm,omitempty" toml:"wwwAuthenticateHeaderRealm,omitempty" yaml:"wwwAuthenticateHeaderRealm,omitempty"`
}

// SetDefaults sets default values for an LDAP middleware.
func (l *LDAPAuth) SetDefaults() {
	l.Attribute = "cn"
	l.ForwardUsernameHeader = "Username"
}

// +k8s:deepcopy-gen=true

// OAuthIntrospection holds the OAuth 2 token introspection Middleware configuration.
type OAuthIntrospection struct {
	// Source is the name of the authentication source this middleware should use.
	Source string `json:"source,omitempty" toml:"source,omitempty" yaml:"source,omitempty"`

	// TokenQueryKey defines where to find the token to introspect in the query parameters. Will look in the Authorization header first.
	TokenQueryKey string `json:"tokenQueryKey,omitempty" toml:"tokenQueryKey,omitempty" yaml:"tokenQueryKey,omitempty"`
	// TokenTypeHint is a hint to pass to the Authorization Server. See https://tools.ietf.org/html/rfc7662#section-2.1 for more information.
	TokenTypeHint string `json:"tokenTypeHint,omitempty" toml:"tokenTypeHint,omitempty" yaml:"tokenTypeHint,omitempty"`

	// ForwardAuthorization determines whether the "Authorization" header or query parameter containing the token should be
	// forwarded or stripped from the request.
	ForwardAuthorization bool `json:"forwardAuthorization,omitempty" toml:"forwardAuthorization,omitempty" yaml:"forwardAuthorization,omitempty"`
	// ForwardHeaders defines headers that should be added to the request and populated with values extracted from the response of the token introspection.
	ForwardHeaders map[string]string `json:"forwardHeaders,omitempty" toml:"forwardHeaders,omitempty" yaml:"forwardHeaders,omitempty"`
	// Claims defines an expression to perform validation on the token introspection's response. For example:
	//     Equals(`grp`, `admin`) && Equals(`scope`, `deploy`)
	Claims string `json:"claims,omitempty" toml:"claims,omitempty" yaml:"claims,omitempty"`
}

// +k8s:deepcopy-gen=true

// JWTAuth holds the JWT Middleware configuration.
type JWTAuth struct {
	// Source is the name of the authentication source this middleware should use.
	Source string `json:"source,omitempty" toml:"source,omitempty" yaml:"source,omitempty"`

	// ForwardAuthorization determines whether the "Authorization" header should be forwarded or stripped from the request.
	ForwardAuthorization bool `json:"forwardAuthorization,omitempty" toml:"forwardAuthorization,omitempty" yaml:"forwardAuthorization,omitempty"`
	// ForwardHeaders defines headers that should be added to the request and populated with values extracted from the JWT.
	ForwardHeaders map[string]string `json:"forwardHeaders,omitempty" toml:"forwardHeaders,omitempty" yaml:"forwardHeaders,omitempty"`
	// Claims defines an expression to perform validation on custom claims present in a JWT. For example:
	//     Equals(`grp`, `admin`) && Equals(`scope`, `deploy`)
	Claims string `json:"claims,omitempty" toml:"claims,omitempty" yaml:"claims,omitempty"`
}

// +k8s:deepcopy-gen=true

// ForceCase holds the ForceCase middleware configuration.
type ForceCase struct {
	// Headers is the list of headers on which to force case.
	Headers []string `json:"headers,omitempty" toml:"headers,omitempty" yaml:"headers,omitempty"`
}

// +k8s:deepcopy-gen=true

// DistributedInFlightReq limits the number of requests being processed and served concurrently, in a cluster.
type DistributedInFlightReq struct {
	Amount          int64            `json:"amount,omitempty" toml:"amount,omitempty" yaml:"amount,omitempty"`
	SourceCriterion *SourceCriterion `json:"sourceCriterion,omitempty" toml:"sourceCriterion,omitempty" yaml:"sourceCriterion,omitempty"`
}

// SetDefaults Default values for a DistributedInFlightReq.
func (i *DistributedInFlightReq) SetDefaults() {
	i.SourceCriterion = &SourceCriterion{
		RequestHost: true,
	}
}

// +k8s:deepcopy-gen=true

// DistributedRateLimit holds the rate limiting configuration for a given router.
type DistributedRateLimit struct {
	// Average is the maximum rate, in requests/s, allowed for the given source.
	// It defaults to 0, which means no rate limiting.
	Average int64 `json:"average,omitempty" toml:"average,omitempty" yaml:"average,omitempty"`
	// Burst is the maximum number of requests allowed to arrive in the same arbitrarily small period of time.
	// It defaults to 1.
	Burst           int64            `json:"burst,omitempty" toml:"burst,omitempty" yaml:"burst,omitempty"`
	SourceCriterion *SourceCriterion `json:"sourceCriterion,omitempty" toml:"sourceCriterion,omitempty" yaml:"sourceCriterion,omitempty"`
}

// SetDefaults sets the default values on a DistributedRateLimit.
func (r *DistributedRateLimit) SetDefaults() {
	r.Burst = 1
	r.SourceCriterion = &SourceCriterion{
		IPStrategy: &IPStrategy{},
	}
}
