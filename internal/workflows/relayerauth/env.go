package relayerauth

import (
	"os"
	"strings"

	"github.com/TrebuchetDynamics/polygolem/internal/relayer"
)

func V2KeyFromProcessEnv() (relayer.V2APIKey, string, bool) {
	v2Key := strings.TrimSpace(os.Getenv("RELAYER_API_KEY"))
	v2Addr := strings.TrimSpace(os.Getenv("RELAYER_API_KEY_ADDRESS"))
	if v2Key != "" && v2Addr != "" {
		return relayer.V2APIKey{Key: v2Key, Address: v2Addr}, "env", true
	}
	return relayer.V2APIKey{}, "", false
}

func V2KeyFromFiles(paths []string) (relayer.V2APIKey, string, bool) {
	for _, path := range paths {
		values, ok := ReadSimpleEnvFile(path)
		if !ok {
			continue
		}
		v2Key := strings.TrimSpace(values["RELAYER_API_KEY"])
		v2Addr := strings.TrimSpace(values["RELAYER_API_KEY_ADDRESS"])
		if v2Key != "" && v2Addr != "" {
			return relayer.V2APIKey{Key: v2Key, Address: v2Addr}, path, true
		}
	}
	return relayer.V2APIKey{}, "", false
}

func EnvFileCandidates(override, defaultFile string) []string {
	if strings.TrimSpace(override) != "" {
		return []string{strings.TrimSpace(override)}
	}
	return []string{
		defaultFile,
		"../.env.relayer-v2",
		".env.relayer-v2",
		"../go-bot/.env",
		"../.env",
		".env",
	}
}

func ReadSimpleEnvFile(path string) (map[string]string, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	out := make(map[string]string)
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		if key != "" {
			out[key] = value
		}
	}
	return out, true
}
