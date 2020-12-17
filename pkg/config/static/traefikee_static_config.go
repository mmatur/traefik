package static

import (
	"errors"
	"fmt"
	"time"

	"github.com/traefik/paerser/types"
)

// Plugin holds TraefikEE-specific Provider configuration.
type Plugin struct {
	Vault *Vault `description:"Enable Vault backend for TLS certificates with default settings." json:"vault" toml:"vault" yaml:"vault" export:"true"`
}

// Vault configures the Vault provider for TLS certificates.
type Vault struct {
	URL string `description:"URL of the Vault API" json:"url" toml:"url" yaml:"url" export:"true"`
	// Deprecated: please use Auth.Token instead.
	Token string    `description:"Token used to authenticate with the API" json:"token" toml:"token" yaml:"token" export:"true"`
	Auth  VaultAuth `description:"Authentication method to use" json:"auth" toml:"auth" yaml:"auth" export:"true"`

	EnginePath     string         `description:"Path under which the KV secret engine is enabled" json:"enginePath" toml:"enginePath" yaml:"enginePath" export:"true"`
	SyncInterval   types.Duration `description:"Interval to synchronize new and deleted certificates" json:"syncInterval" toml:"syncInterval" yaml:"syncInterval" export:"true"`
	RescanInterval types.Duration `description:"Interval to rescan all certificates for changes" json:"rescanInterval" toml:"rescanInterval" yaml:"rescanInterval" export:"true"`
}

// SetDefaults sets the default values on the Vault provider configuration.
func (p *Vault) SetDefaults() {
	p.EnginePath = "secret"
	p.SyncInterval = types.Duration(5 * time.Second)
	p.RescanInterval = types.Duration(60 * time.Second)
}

// VaultPKI configures Vault as a certificate resolver.
type VaultPKI struct {
	URL string `description:"URL of the Vault server" json:"url" toml:"url" yaml:"url" export:"true"`
	// Deprecated: please use Auth.Token instead.
	Token string    `description:"Token used to authenticate with Vault" json:"token" toml:"token" yaml:"token" export:"true"`
	Auth  VaultAuth `json:"auth" toml:"auth" yaml:"auth" export:"true"`

	EnginePath string `description:"Path under which the Vault PKI secret engine is enabled" json:"enginePath" toml:"enginePath" yaml:"enginePath" export:"true"`
	Role       string `description:"Role to be used to issue certificates" json:"role" toml:"role" yaml:"role" export:"true"`
}

// SetDefaults sets the default values on the Vault provider configuration.
func (p *VaultPKI) SetDefaults() {
	p.EnginePath = "pki"
}

// VaultAuth describes authentication methods for Vault providers.
type VaultAuth struct {
	Token   string   `description:"Token used to authenticate with Vault" json:"token" yaml:"token" toml:"token"`
	AppRole *AppRole `json:"appRole" yaml:"appRole" toml:"appRole"`
}

// Validate validates that exactly one authentication method is present and that it is valid.
func (a VaultAuth) Validate() error {
	if a.Token == "" && a.AppRole == nil {
		return errors.New("at least token or appRole must be set")
	}

	if a.Token != "" && a.AppRole != nil {
		return errors.New("multiple authentication methods set")
	}

	if a.AppRole != nil {
		if err := a.AppRole.Validate(); err != nil {
			return fmt.Errorf("appRole: %w", err)
		}
	}

	return nil
}

// AppRole configures the Vault AppRole authentication.
type AppRole struct {
	RoleID   string `description:"Role ID to use with AppRole authentication" json:"roleID" yaml:"roleID" toml:"roleID"`
	SecretID string `description:"Secret ID to use with AppRole authentication" json:"secretID" yaml:"secretID" toml:"secretID"`
	Path     string `description:"Custom path under which AppRole authentication is enabled in Vault" json:"path" yaml:"path" toml:"path"`
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

// DistributedACME configures the DistributedACME provider for TLS certificates.
type DistributedACME struct {
	URL string `description:"URL of the ACME Agent" json:"url" toml:"url" yaml:"url" export:"true"`

	TLS *DistributedTLS `description:"TLS certificates and keys used for mTLS" json:"tls" toml:"tls" yaml:"tls" export:"true"`
}

// DistributedTLS configures mTLS for the distributed ACME feature.
type DistributedTLS struct {
	Cert string `description:"Path to the client certificate" json:"cert" toml:"cert" yaml:"cert" export:"true"`
	Key  string `description:"Path to the client key" json:"key" toml:"key" yaml:"key" export:"true"`
	CA   string `description:"Path to the certificate authority" json:"ca" toml:"ca" yaml:"ca" export:"true"`
}
