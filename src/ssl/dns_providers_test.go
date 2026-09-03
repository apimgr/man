package ssl

import (
	"reflect"
	"testing"
)

func TestProviders_IncludesManualAndCommonProviders(t *testing.T) {
	got := Providers()
	if len(got) < 5 {
		t.Fatalf("expected at least 5 providers, got %d", len(got))
	}
	have := map[string]bool{}
	for _, p := range got {
		have[p.ID] = true
	}
	for _, want := range []string{"manual", "cloudflare", "route53", "digitalocean", "godaddy", "namecheap", "rfc2136"} {
		if !have[want] {
			t.Errorf("provider %q missing from list", want)
		}
	}
}

func TestProviderByID(t *testing.T) {
	if _, ok := ProviderByID("cloudflare"); !ok {
		t.Error("cloudflare should resolve")
	}
	if _, ok := ProviderByID("nope"); ok {
		t.Error("unknown provider should not resolve")
	}
}

func TestValidateFields_RequiredFieldsEnforced(t *testing.T) {
	if _, err := ValidateFields("cloudflare", map[string]string{}); err == nil {
		t.Error("expected error for missing required field")
	}
	got, err := ValidateFields("cloudflare", map[string]string{
		"CLOUDFLARE_DNS_API_TOKEN": "abc",
		"IGNORED":                  "drop me",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]string{"CLOUDFLARE_DNS_API_TOKEN": "abc"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("clean fields = %v, want %v", got, want)
	}
}

func TestValidateFields_ManualNeedsNothing(t *testing.T) {
	got, err := ValidateFields(Manual, nil)
	if err != nil {
		t.Fatalf("manual provider should accept empty fields: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("manual fields should be empty, got %v", got)
	}
}

func TestValidateFields_UnknownProvider(t *testing.T) {
	if _, err := ValidateFields("nonesuch", nil); err == nil {
		t.Error("unknown provider should error")
	}
}
