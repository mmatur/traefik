package static

import (
	"time"

	"github.com/containous/traefik/v2/pkg/types"
)

// Plugin holds TraefikEE-specific Provider configuration.
type Plugin struct {
	Vault *Vault `description:"Enable Vault backend for TLS certificates with default settings." json:"vault" toml:"vault" yaml:"vault" export:"true"`
}

// Vault configures the Vault provider for TLS certificates.
type Vault struct {
	URL   string `description:"URL of the Vault API" json:"url" toml:"url" yaml:"url" export:"true"`
	Token string `description:"Token used to authenticate with the API" json:"token" toml:"token" yaml:"token" export:"true"`
	Path  string `description:"Path where TLS certificates are located" json:"path" toml:"path" yaml:"path" export:"true"`

	SyncInterval   types.Duration `description:"Interval to synchronize new and deleted certificates" json:"syncInterval" toml:"syncInterval" yaml:"syncInterval" export:"true"`
	RescanInterval types.Duration `description:"Interval to rescan all certificates for changes" json:"rescanInterval" toml:"rescanInterval" yaml:"rescanInterval" export:"true"`
}

// SetDefaults sets the default values on the Vault provider configuration.
func (p *Vault) SetDefaults() {
	p.Path = "tls"
	p.SyncInterval = types.Duration(5 * time.Second)
	p.RescanInterval = types.Duration(60 * time.Second)
}

// VaultPKI configures Vault as a certificate resolver.
type VaultPKI struct {
	URL   string `description:"URL of the Vault server" json:"url" toml:"url" yaml:"url" export:"true"`
	Token string `description:"Token used to authenticate with Vault" json:"token" toml:"token" yaml:"token" export:"true"`
	Path  string `description:"Path under which the Vault PKI secret engine is enabled" json:"path" toml:"path" yaml:"path" export:"true"`
	Role  string `description:"Role to be used to issue certificates" json:"role" toml:"role" yaml:"role" export:"true"`
}

// SetDefaults sets the default values on the Vault provider configuration.
func (p *VaultPKI) SetDefaults() {
	p.Path = "pki"
}
