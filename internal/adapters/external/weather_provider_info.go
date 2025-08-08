package external

import "weatherapi.app/internal/ports"

// NewProviderInfo creates provider info from a list of providers
func NewProviderInfo(providerNames []string) ports.ProviderInfo {
	return ports.ProviderInfo{
		TotalProviders:  len(providerNames),
		ProviderOrder:   providerNames,
		ChainEnabled:    true,
		FallbackEnabled: len(providerNames) > 1,
	}
}
