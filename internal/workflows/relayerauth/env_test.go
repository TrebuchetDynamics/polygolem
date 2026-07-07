package relayerauth

import (
	"os"
	"path/filepath"
	"testing"
)

func TestV2KeyFromFilesReadsSimpleEnv(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("export RELAYER_API_KEY='key-1'\nRELAYER_API_KEY_ADDRESS=0xabc\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	key, source, ok := V2KeyFromFiles([]string{path})
	if !ok || source != path || key.Key != "key-1" || key.Address != "0xabc" {
		t.Fatalf("key=%+v source=%q ok=%v", key, source, ok)
	}
}

func TestEnvFileCandidatesHonorsOverride(t *testing.T) {
	got := EnvFileCandidates(" custom.env ", "default.env")
	if len(got) != 1 || got[0] != "custom.env" {
		t.Fatalf("candidates=%v", got)
	}
}
