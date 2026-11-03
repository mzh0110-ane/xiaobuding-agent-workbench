package reliable

import "gopkg.in/yaml.v3"

// Credential values are loaded only into the in-memory log filter.
func nativeCredentialSecrets(raw []byte) []string {
	var store struct {
		Refs    map[string]string `yaml:"refs"`
		Records map[string]struct {
			Key     string            `yaml:"key"`
			Env     map[string]string `yaml:"env"`
			Payload any               `yaml:"payload"`
		} `yaml:"records"`
	}
	if yaml.Unmarshal(raw, &store) != nil {
		return nil
	}
	var values []string
	for _, value := range store.Refs {
		values = append(values, value)
	}
	for _, record := range store.Records {
		values = append(values, record.Key)
		for _, value := range record.Env {
			values = append(values, value)
		}
		values = append(values, payloadStrings(record.Payload)...)
	}
	return values
}

func payloadStrings(value any) []string {
	var out []string
	switch typed := value.(type) {
	case string:
		out = append(out, typed)
	case map[string]any:
		for _, nested := range typed {
			out = append(out, payloadStrings(nested)...)
		}
	case map[any]any:
		for _, nested := range typed {
			out = append(out, payloadStrings(nested)...)
		}
	case []any:
		for _, nested := range typed {
			out = append(out, payloadStrings(nested)...)
		}
	}
	return out
}
