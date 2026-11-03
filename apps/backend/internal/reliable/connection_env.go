package reliable

import "strings"

func isolatedEnvironment(base []string, overrides map[string]string, removed []string) []string {
	blocked := make(map[string]bool, len(overrides)+len(removed))
	for key := range overrides {
		blocked[strings.ToUpper(key)] = true
	}
	for _, key := range removed {
		blocked[strings.ToUpper(key)] = true
	}
	out := make([]string, 0, len(base)+len(overrides))
	for _, entry := range base {
		key := strings.ToUpper(strings.SplitN(entry, "=", 2)[0])
		if !blocked[key] {
			out = append(out, entry)
		}
	}
	for key, value := range overrides {
		out = append(out, key+"="+value)
	}
	return out
}
