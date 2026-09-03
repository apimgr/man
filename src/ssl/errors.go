package ssl

import "fmt"

func errUnknownProvider(id string) error {
	return fmt.Errorf("ssl: unknown provider %q", id)
}

func errMissingField(provider, key string) error {
	return fmt.Errorf("ssl: provider %q missing required field %q", provider, key)
}
