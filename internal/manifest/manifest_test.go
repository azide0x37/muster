package manifest

import (
	"strings"
	"testing"
)

func TestDecodeInspectionGraph(t *testing.T) {
	document, err := Decode(strings.NewReader(`
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
  summary: An example implementation.
  root_components: [component:example:service]
  components:
    - id: component:example:service
      kind: systemd.service
      summary: Runs the example.
      source:
        adapter: systemd
        unit: example.service
      actions:
        - id: action:example:restart
          label: Restart
          kind: systemd.restart
          command: [systemctl, restart, example.service]
  edges: []
`))
	if err != nil {
		t.Fatal(err)
	}
	if document.Inspection.ID != "implementation:example" {
		t.Fatalf("unexpected implementation ID %q", document.Inspection.ID)
	}
	if got := document.Inspection.Components[0].Source.Unit; got != "example.service" {
		t.Fatalf("unexpected unit %q", got)
	}
}

func TestDecodeRejectsUnknownInspectionField(t *testing.T) {
	_, err := Decode(strings.NewReader(`
schema: 2
framework: Muster
project: {name: example}
inspection:
  id: implementation:example
  mystery: true
`))
	if err == nil {
		t.Fatal("expected unknown field to fail")
	}
}
