package version

import "testing"

func TestInfoPublishesSafeBuildMetadata(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	info := Info()
	if info.Service != ServiceName || info.Environment != "test" || info.Version == "" || info.GitSHA == "" || info.BuildTime == "" || info.APIContractVersion == "" {
		t.Fatalf("unexpected version metadata: %#v", info)
	}
}
