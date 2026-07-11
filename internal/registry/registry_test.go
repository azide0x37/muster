package registry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHostPathUsesStagedRoot(t *testing.T) {
	got, err := HostPath("/tmp/stage", "/opt/example/current/muster.yaml")
	if err != nil {
		t.Fatal(err)
	}
	want := "/tmp/stage/opt/example/current/muster.yaml"
	if got != want {
		t.Fatalf("HostPath() = %q, want %q", got, want)
	}
}

func TestLoadIsDeterministic(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "etc/muster/implementations.d")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"z.json": `{"schema":1,"id":"implementation:z","manifest":"/opt/z/current/muster.yaml"}`,
		"a.json": `{"schema":1,"id":"implementation:a","manifest":"/opt/a/current/muster.yaml"}`,
	} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].ID != "implementation:a" {
		t.Fatalf("unexpected entries %#v", entries)
	}
}
