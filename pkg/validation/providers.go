package validation

import (
	"fmt"
	"os"

	"github.com/oauth2-proxy/oauth2-proxy/v7/pkg/apis/options"
)

// validateProviders is the initial validation migration for multiple providrers
// It currently includes only logic that can verify the providers one by one and does not break the valdation pipe
func validateProviders(o *options.Options) []string {
	msgs := []string{}

	// validate general multiple provider configuration
	if len(o.Providers) == 0 {
		msgs = append(msgs, "at least one provider has to be defined")
	}
	if o.SkipProviderButton && len(o.Providers) > 1 {
		msgs = append(msgs, "SkipProviderButton and multiple providers are mutually exclusive")
	}

	providerIDs := make(map[string]struct{})

	for _, provider := range o.Providers {
		msgs = append(msgs, validateProvider(provider, providerIDs)...)
	}

	return msgs
}

func validateProvider(provider options.Provider, providerIDs map[string]struct{}) []string {
	msgs := []string{}

	if provider.ID == "" {
		msgs = append(msgs, "provider has empty id: ids are required for all providers")
	}

	// Ensure provider IDs are unique
	if _, ok := providerIDs[provider.ID]; ok {
		msgs = append(msgs, fmt.Sprintf("multiple providers found with id %s: provider ids must be unique", provider.ID))
	}
	providerIDs[provider.ID] = struct{}{}

	if provider.ClientID == "" {
		msgs = append(msgs, "provider missing setting: client-id")
	}

	// Add authentication method validations specific to the fork
	msgs = append(msgs, validateAuthenticationConfig(provider.AuthenticationConfig)...)

	// Add provider-specific validations
	if provider.Type == "google" {
		msgs = append(msgs, validateGoogleConfig(provider)...)
	}

	if provider.Type == "entra-id" {
		msgs = append(msgs, validateEntraConfig(provider)...)
	}

	if provider.Type == "login.gov" {
		msgs = append(msgs, validateGovLoginConfig(provider)...)
	}

	return msgs
}

func validateGoogleConfig(provider options.Provider) []string {
	msgs := []string{}

	hasGoogleGroups := len(provider.GoogleConfig.Groups) >= 1
	hasAdminEmail := provider.GoogleConfig.AdminEmail != ""
	hasSAJSON := provider.GoogleConfig.ServiceAccountJSON != ""
	useADC := provider.GoogleConfig.UseApplicationDefaultCredentials

	if !hasGoogleGroups && !hasAdminEmail && !hasSAJSON && !useADC {
		return msgs
	}

	if !hasGoogleGroups {
		msgs = append(msgs, "missing setting: google-group")
	}
	if !hasAdminEmail {
		msgs = append(msgs, "missing setting: google-admin-email")
	}

	_, err := os.Stat(provider.GoogleConfig.ServiceAccountJSON)
	if !useADC {
		if !hasSAJSON {
			msgs = append(msgs, "missing setting: google-service-account-json or google-use-application-default-credentials")
		} else if err != nil {
			msgs = append(msgs, fmt.Sprintf("Google credentials file not found: %s", provider.GoogleConfig.ServiceAccountJSON))
		}
	} else if hasSAJSON {
		msgs = append(msgs, "invalid setting: can't use both google-service-account-json and google-use-application-default-credentials")
	}

	return msgs
}

func validateEntraConfig(provider options.Provider) []string {
	msgs := []string{}

	if provider.MicrosoftEntraIDConfig.FederatedTokenAuth {
		federatedTokenPath := os.Getenv("AZURE_FEDERATED_TOKEN_FILE")

		if federatedTokenPath == "" {
			msgs = append(msgs, "entra federated token authentication is enabled, but AZURE_FEDERATED_TOKEN_FILE variable is not set, check your workload identity configuration.")
			return msgs
		}

		_, err := os.ReadFile(federatedTokenPath)
		if err != nil {
			msgs = append(msgs, "could not read entra federated token file")
		}
	}

	return msgs
}

func validateGovLoginConfig(provider options.Provider) []string {
	msgs := []string{}

	if provider.Type == "login.gov" && provider.AuthenticationConfig.Method != options.PrivateKeyJWT {
		msgs = append(msgs, "login.gov configuration not using private key jwt")
	}

	return msgs
}
