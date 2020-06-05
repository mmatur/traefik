package static

// Vault configures the Vault provider for TLS certificates.
type Vault struct {
	Address string `description:"HTTP address of the Vault API" json:"address" toml:"address" yaml:"address" export:"true"`
	Token   string `description:"Token used to authenticate with the API" json:"token" toml:"token" yaml:"token" export:"true"`
	Path    string `description:"Path where TLS certificates are located" json:"path" toml:"path" yaml:"path" export:"true"`
}

// SetDefaults sets the default values on the Vault provider configuration.
func (p *Vault) SetDefaults() {
	p.Path = "tls"
}
