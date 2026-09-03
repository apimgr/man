// DNS provider metadata used by the admin SSL form. Each provider declares
// the credential fields it requires; the same metadata drives validation in
// the POST handler and field rendering in the GET handler. Provider names
// match the lego dns/<name> package import path.

package ssl

import "sort"

// ProviderField describes one credential input for a DNS provider.
type ProviderField struct {
	// Key is both the form field name and the lego env var (uppercased).
	Key string
	// Label is the human-readable label shown in the form.
	Label string
	// Required marks fields that must be non-empty to save credentials.
	Required bool
	// Secret hides the value from echo and from List() output.
	Secret bool
}

// Provider describes a DNS provider that can solve dns-01 challenges.
type Provider struct {
	// ID matches lego's provider name (e.g. "cloudflare").
	ID string
	// Name is the display label.
	Name string
	// Fields are the credential inputs in form order.
	Fields []ProviderField
}

// Manual is the always-available fallback that prompts the operator to
// publish TXT records by hand. It needs no credentials.
const Manual = "manual"

var providers = map[string]Provider{
	Manual: {
		ID:     Manual,
		Name:   "Manual (publish TXT records yourself)",
		Fields: nil,
	},
	"cloudflare": {
		ID:   "cloudflare",
		Name: "Cloudflare",
		Fields: []ProviderField{
			{Key: "CLOUDFLARE_DNS_API_TOKEN", Label: "API token", Required: true, Secret: true},
		},
	},
	"route53": {
		ID:   "route53",
		Name: "AWS Route 53",
		Fields: []ProviderField{
			{Key: "AWS_ACCESS_KEY_ID", Label: "Access key ID", Required: true},
			{Key: "AWS_SECRET_ACCESS_KEY", Label: "Secret access key", Required: true, Secret: true},
			{Key: "AWS_REGION", Label: "Region (e.g. us-east-1)", Required: false},
			{Key: "AWS_HOSTED_ZONE_ID", Label: "Hosted zone ID (optional)", Required: false},
		},
	},
	"digitalocean": {
		ID:   "digitalocean",
		Name: "DigitalOcean",
		Fields: []ProviderField{
			{Key: "DO_AUTH_TOKEN", Label: "API token", Required: true, Secret: true},
		},
	},
	"godaddy": {
		ID:   "godaddy",
		Name: "GoDaddy",
		Fields: []ProviderField{
			{Key: "GODADDY_API_KEY", Label: "API key", Required: true},
			{Key: "GODADDY_API_SECRET", Label: "API secret", Required: true, Secret: true},
		},
	},
	"namecheap": {
		ID:   "namecheap",
		Name: "Namecheap",
		Fields: []ProviderField{
			{Key: "NAMECHEAP_API_USER", Label: "API user", Required: true},
			{Key: "NAMECHEAP_API_KEY", Label: "API key", Required: true, Secret: true},
		},
	},
	"rfc2136": {
		ID:   "rfc2136",
		Name: "RFC 2136 (BIND/Knot dynamic update)",
		Fields: []ProviderField{
			{Key: "RFC2136_NAMESERVER", Label: "Nameserver (host:port)", Required: true},
			{Key: "RFC2136_TSIG_KEY", Label: "TSIG key name", Required: true},
			{Key: "RFC2136_TSIG_ALGORITHM", Label: "TSIG algorithm (hmac-sha256)", Required: false},
			{Key: "RFC2136_TSIG_SECRET", Label: "TSIG secret", Required: true, Secret: true},
		},
	},
}

// Providers returns all known providers ordered by display name.
func Providers() []Provider {
	out := make([]Provider, 0, len(providers))
	for _, p := range providers {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// ProviderByID returns the provider with the given ID, or false if unknown.
func ProviderByID(id string) (Provider, bool) {
	p, ok := providers[id]
	return p, ok
}

// ValidateFields ensures every required field for the provider is present and
// non-empty. Unknown providers return an error. Extra fields not declared by
// the provider are dropped silently.
func ValidateFields(providerID string, fields map[string]string) (map[string]string, error) {
	p, ok := providers[providerID]
	if !ok {
		return nil, errUnknownProvider(providerID)
	}
	clean := map[string]string{}
	for _, f := range p.Fields {
		val, present := fields[f.Key]
		if f.Required && (!present || val == "") {
			return nil, errMissingField(providerID, f.Key)
		}
		if present {
			clean[f.Key] = val
		}
	}
	return clean, nil
}
