package inspector

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/azide0x37/muster/internal/manifest"
)

func TestLoadProjectsStagedImplementationAndDoctorEvidence(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "etc/muster/implementations.d/example.json"), `{"schema":1,"id":"implementation:example","manifest":"/opt/example/current/muster.yaml"}`)
	mustWrite(t, filepath.Join(root, "opt/example/releases/1.2.3/VERSION"), "1.2.3\n")
	mustWrite(t, filepath.Join(root, "opt/example/releases/1.2.3/muster.yaml"), `
schema: 2
framework: Muster
project:
  name: example
  type: linux-service-appliance
  version_file: VERSION
  config_dir: /etc/example
  install_dir: /opt/example
  current_link: /opt/example/current
  release_dir: /opt/example/releases
inspection:
  id: implementation:example
  summary: Test implementation.
  root_components: [component:example:runtime, component:example:doctor]
  components:
    - id: component:example:runtime
      kind: component.group
      summary: Runtime
      children: [component:example:unit]
    - id: component:example:unit
      kind: systemd.service
      summary: Test unit
      source: {adapter: systemd.unit, unit: example.service}
    - id: component:example:doctor
      kind: doctor
      summary: Doctor
      source:
        adapter: observation.file
        state_file: /run/muster/example/observations/doctor.json
        max_age_seconds: 3600
  edges: []
`)
	mustSymlink(t, "releases/1.2.3", filepath.Join(root, "opt/example/current"))
	mustWrite(t, filepath.Join(root, "etc/systemd/system/example.service"), "[Service]\n")
	mustWrite(t, filepath.Join(root, "run/muster/example/observations/doctor.json"), `{
  "schema":"muster.observation/v1",
  "implementation":"example",
  "component":"doctor",
  "scope":"runtime",
  "health":"healthy",
  "status":"complete",
  "summary":"All checks passed",
  "observed_at":"2026-07-11T01:00:00Z",
  "duration_ms":12,
  "valid_for_seconds":3600,
  "checks":[{"id":"unit","health":"healthy","summary":"unit is active"}]
}`)

	host := New(root)
	host.Now = func() time.Time { return time.Date(2026, 7, 11, 1, 5, 0, 0, time.UTC) }
	snapshot, err := host.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	implementation, ok := snapshot.Graph.LookupImplementation("implementation:example")
	if !ok || implementation.Version != "1.2.3" {
		t.Fatalf("unexpected implementation %#v", implementation)
	}
	rootComponent, _ := snapshot.Graph.Lookup("implementation:example")
	if rootComponent.Health.Status != "healthy" {
		t.Fatalf("root health = %s, want healthy", rootComponent.Health.Status)
	}
	observation, ok := snapshot.Graph.LatestObservation("component:example:doctor", "doctor")
	if !ok || len(observation.Checks) != 1 {
		t.Fatalf("unexpected doctor observation %#v", observation)
	}
}

func TestStaleObservationBecomesUnknown(t *testing.T) {
	source := testObservationSource(t)
	host := New(t.TempDir())
	host.Now = func() time.Time { return time.Date(2026, 7, 12, 0, 0, 0, 0, time.UTC) }
	// Re-root the fixture's logical path underneath this inspector.
	path := filepath.Join(host.Root, "run/muster/example/doctor.json")
	mustWrite(t, path, source)
	health, observation, err := host.inspectObservation("example", "component:example:doctor", observationSource("/run/muster/example/doctor.json", 60))
	if err != nil {
		t.Fatal(err)
	}
	if health.Status != "unknown" {
		t.Fatalf("stale health = %s, want unknown", health.Status)
	}
	if observation == nil || !observation.Stale || observation.DerivedHealth().Status != "unknown" {
		t.Fatalf("stale observation = %#v, want stale unknown evidence", observation)
	}
}

func TestDeclaredInstalledLockIsRequired(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "etc/muster/implementations.d/example.json"), `{"schema":1,"id":"implementation:example","manifest":"/opt/example/current/muster.yaml","lock":"/opt/example/current/muster.lock.json"}`)
	mustWrite(t, filepath.Join(root, "opt/example/releases/1.0.0/muster.yaml"), "schema: 2\nframework: Muster\n")
	mustSymlink(t, "releases/1.0.0", filepath.Join(root, "opt/example/current"))

	_, err := New(root).Load(context.Background())
	if err == nil || !strings.Contains(err.Error(), "declared implementation lock is missing") {
		t.Fatalf("Load() error = %v, want missing declared lock", err)
	}
}

func TestStaticPatternCanDeclarePartialCoverageHealth(t *testing.T) {
	host := New(t.TempDir())
	health, observation, err := host.inspectSource(context.Background(), "example", "pattern:example:T2R1", manifest.SourceSpec{Adapter: "pattern", Status: "degraded"})
	if err != nil {
		t.Fatal(err)
	}
	if health.Status != "degraded" || observation != nil {
		t.Fatalf("pattern health = %#v, observation = %#v", health, observation)
	}
	if _, _, err := host.inspectSource(context.Background(), "example", "pattern:example:bad", manifest.SourceSpec{Adapter: "pattern", Status: "purple"}); err == nil {
		t.Fatal("invalid declared pattern status was accepted")
	}
}

func testObservationSource(t *testing.T) string {
	t.Helper()
	return `{"schema":"muster.observation/v1","implementation":"example","component":"doctor","health":"healthy","status":"complete","summary":"ok","observed_at":"2026-07-11T00:00:00Z","duration_ms":1,"checks":[]}`
}

func observationSource(path string, maxAge int64) manifest.SourceSpec {
	return manifest.SourceSpec{Adapter: "observation.file", StateFile: path, MaxAgeSeconds: maxAge}
}

func mustWrite(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustSymlink(t *testing.T, target, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
}
