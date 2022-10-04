package acme

import "fmt"

// Refresh refreshes the internal cache from the store.
// This function causes a dynamic configuration to be sent.
func (p *Provider) Refresh() error {
	certificates, err := p.Store.GetCertificates(p.ResolverName)
	if err != nil {
		return fmt.Errorf("unable to get ACME certificates : %w", err)
	}
	p.certificates = certificates

	p.configurationChan <- p.buildMessage()

	return nil
}
