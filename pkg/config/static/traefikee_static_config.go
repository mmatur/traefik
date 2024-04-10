package static

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/traefik/traefik/v2/pkg/tls"
)

// VaultPKI configures Vault as a certificate resolver.
type VaultPKI struct {
	URL string `description:"URL of the Vault server" json:"url" toml:"url" yaml:"url"`
	TLS *TLS   `description:"TLS configuration" json:"tls" toml:"tls" yaml:"tls" export:"true"`
	// Deprecated: please use Auth.Token instead.
	Token string    `description:"Token used to authenticate with Vault" json:"token" toml:"token" yaml:"token"`
	Auth  VaultAuth `json:"auth" toml:"auth" yaml:"auth" export:"true"`

	Namespace  string `description:"Namespace of the Vault PKI secret engine" json:"namespace" toml:"namespace" yaml:"namespace"`
	EnginePath string `description:"Path under which the Vault PKI secret engine is enabled" json:"enginePath" toml:"enginePath" yaml:"enginePath"`
	Role       string `description:"Role to be used to issue certificates" json:"role" toml:"role" yaml:"role"`
}

// SetDefaults sets the default values on the Vault provider configuration.
func (p *VaultPKI) SetDefaults() {
	p.EnginePath = "pki"
}

// TLS configures TLS communication.
type TLS struct {
	CABundle           tls.FileOrContent `description:"Certificate Authority bundle to use for TLS communication" json:"caBundle" toml:"caBundle" yaml:"caBundle"`
	Cert               string            `description:"TLS cert" json:"cert,omitempty" toml:"cert,omitempty" yaml:"cert,omitempty"`
	Key                string            `description:"TLS key" json:"key,omitempty" toml:"key,omitempty" yaml:"key,omitempty" loggable:"false"`
	InsecureSkipVerify bool              `description:"Whether the client should verify the TLS certificate" json:"insecureSkipVerify" toml:"insecureSkipVerify" yaml:"insecureSkipVerify" export:"true"`
}

// VaultAuth describes authentication methods for Vault providers.
type VaultAuth struct {
	Token      string      `description:"Token used to authenticate with Vault" json:"token" toml:"token" yaml:"token"`
	AppRole    *AppRole    `description:"Configures the Vault AppRole authentication" json:"appRole" toml:"appRole" yaml:"appRole" `
	Kubernetes *Kubernetes `description:"Configures the Vault Kubernetes authentication" json:"kubernetes" toml:"kubernetes" yaml:"kubernetes"`
}

// Validate validates that exactly one authentication method is present and that it is valid.
func (a VaultAuth) Validate() error {
	if err := ensureOneFieldSet(&a); err != nil {
		return fmt.Errorf("invalid authentication method: %w", err)
	}

	if a.AppRole != nil {
		if err := a.AppRole.Validate(); err != nil {
			return fmt.Errorf("appRole: %w", err)
		}
	}

	if a.Kubernetes != nil {
		if err := a.Kubernetes.Validate(); err != nil {
			return fmt.Errorf("kubernetes: %w", err)
		}
	}

	return nil
}

// AppRole configures the Vault AppRole authentication.
type AppRole struct {
	RoleID   string `description:"Role ID to use with AppRole authentication" json:"roleID" toml:"roleID" yaml:"roleID"`
	SecretID string `description:"Secret ID to use with AppRole authentication" json:"secretID" toml:"secretID" yaml:"secretID"`
	Path     string `description:"Custom path under which AppRole authentication is enabled in Vault" json:"path" toml:"path" yaml:"path"`
}

// SetDefaults sets the default values on the AppRole configuration.
func (p *AppRole) SetDefaults() {
	p.Path = "approle"
}

// Validate validates the AppRole configuration.
func (p *AppRole) Validate() error {
	if p.RoleID == "" {
		return errors.New("roleID must be set")
	}
	if p.SecretID == "" {
		return errors.New("secretID must be set")
	}

	return nil
}

// Kubernetes configures the Vault Kubernetes authentication.
type Kubernetes struct {
	Role string `description:"Role to use with Kubernetes authentication" json:"role" toml:"role" yaml:"role"`
	Path string `description:"Custom path under which Kubernetes authentication is enabled in Vault" json:"path" toml:"path" yaml:"path"`
}

// SetDefaults sets the default values on the Kubernetes configuration.
func (k *Kubernetes) SetDefaults() {
	k.Path = "kubernetes"
}

// Validate validates the Kubernetes configuration.
func (k *Kubernetes) Validate() error {
	if k.Role == "" {
		return errors.New("role must be set")
	}

	return nil
}

// DistributedACME configures the DistributedACME provider for TLS certificates.
type DistributedACME struct {
	URL string          `description:"URL of the ACME Agent" json:"url" toml:"url" yaml:"url"`
	TLS *DistributedTLS `description:"TLS certificates and keys used for mTLS" json:"tls" toml:"tls" yaml:"tls" export:"true"`
}

// DistributedTLS configures mTLS for the distributed ACME feature.
type DistributedTLS struct {
	Cert string `description:"Path to the client certificate" json:"cert" toml:"cert" yaml:"cert"`
	Key  string `description:"Path to the client key" json:"key" toml:"key" yaml:"key"`
	CA   string `description:"Path to the certificate authority" json:"ca" toml:"ca" yaml:"ca"`
}

// ensureOneFieldSet ensures exactly one field is set in the given structure.
func ensureOneFieldSet(s interface{}) error {
	var set, available []string
	v := reflect.ValueOf(s).Elem()
	for i := range v.NumField() {
		// Get the property name as specified in the configuration. Using the YAML struct tag
		// here, but they (JSON and TOML) are all set to the same value, so it doesn't matter:
		propertyName := strings.TrimSuffix(v.Type().Field(i).Tag.Get("yaml"), ",omitempty")
		available = append(available, propertyName)

		if !v.Field(i).IsZero() {
			set = append(set, propertyName)
		}
	}

	if len(set) == 0 {
		return fmt.Errorf("one of %q must be set", strings.Join(available, ", "))
	}
	if len(set) > 1 {
		return fmt.Errorf("only one of the following can be set: %q", strings.Join(set, ", "))
	}
	return nil
}
