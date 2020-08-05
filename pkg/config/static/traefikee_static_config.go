package static

// Vault configures the Vault provider for TLS certificates.
type Vault struct {
	URL   string `description:"URL of the Vault API" json:"url" toml:"url" yaml:"url" export:"true"`
	Token string `description:"Token used to authenticate with the API" json:"token" toml:"token" yaml:"token" export:"true"`
	Path  string `description:"Path where TLS certificates are located" json:"path" toml:"path" yaml:"path" export:"true"`

	SyncInterval   int `description:"Interval in seconds to synchronize new and deleted certificates" json:"syncInterval" toml:"syncInterval" yaml:"syncInterval" export:"true"`
	RescanInterval int `description:"Interval in seconds to rescan all certificates for changes" json:"rescanInterval" toml:"rescanInterval" yaml:"rescanInterval" export:"true"`
}

// SetDefaults sets the default values on the Vault provider configuration.
func (p *Vault) SetDefaults() {
	p.Path = "tls"
	p.SyncInterval = 5
	p.RescanInterval = 60
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
