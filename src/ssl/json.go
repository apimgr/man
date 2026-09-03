package ssl

import "encoding/json"

func encodeJSON(v any) ([]byte, error) { return json.MarshalIndent(v, "", "  ") }

func decodeJSON(data []byte, v any) error { return json.Unmarshal(data, v) }
