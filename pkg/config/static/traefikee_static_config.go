package static

// Vault configures the Vault provider for TLS certificates.
type Vault struct {
	Address string `description:"HTTP address of the Vault API" json:"address" toml:"address" yaml:"address" export:"true"`
	Token   string `description:"Token used to authenticate with the API" json:"token" toml:"token" yaml:"token" export:"true"`
	Path    string `description:"Path where TLS certificates are located" json:"path" toml:"path" yaml:"path" export:"true"`

	SyncInterval   int `description:"Interval in seconds to synchronize new and deleted certificates" json:"syncInterval" toml:"syncInterval" yaml:"syncInterval" export:"true"`
	RescanInterval int `description:"Interval in seconds to rescan all certificates for changes" json:"rescanInterval" toml:"rescanInterval" yaml:"rescanInterval" export:"true"`
}

// SetDefaults sets the default values on the Vault provider configuration.
func (p *Vault) SetDefaults() {
	p.Path = "tls"
	p.SyncInterval = 5
	p.RescanInterval = 60
}
