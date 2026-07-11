package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/azide0x37/muster/internal/inspector"
	"github.com/azide0x37/muster/internal/manifest"
	"github.com/azide0x37/muster/internal/model"
)

func TestInspectAndExplainAddressEveryGraphObject(t *testing.T) {
	observedAt := time.Date(2026, time.July, 11, 2, 3, 4, 0, time.UTC)
	graph, err := model.NewGraph(
		[]model.Implementation{{
			ID: "implementation:example", Components: []model.ID{
				"implementation:example", "component:worker", "component:storage",
			},
		}},
		[]model.Component{
			{
				ID: "implementation:example", Kind: "implementation",
				Health: model.Health{Status: model.HealthHealthy}, Children: []model.ID{"component:worker"},
			},
			{
				ID: "component:worker", Kind: "systemd.service", Health: model.Health{Status: model.HealthHealthy},
				Actions: []model.Action{{
					ID: "action:example:doctor.run", Kind: "doctor.run", Label: "Run doctor",
					Target: "component:worker", RequiresRoot: true, RequiresConfirmation: true,
				}},
			},
			{ID: "component:storage", Kind: "storage", Health: model.Health{Status: model.HealthDegraded, Summary: "read-only"}},
		},
		[]model.Edge{{From: "component:worker", Type: model.EdgeDependsOn, To: "component:storage"}},
		[]model.Observation{{
			ID: "observation:example:doctor:1", ComponentID: "component:worker", Kind: model.ObservationDoctor,
			Status: model.HealthHealthy, ObservedAt: observedAt, Checks: []model.Check{{ID: "worker", Status: model.CheckPass}},
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := graph.MaterializeDerivedHealth(); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	app := App{Out: &output}
	if err := app.inspect(graph, []string{"action:example:doctor.run"}, false); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"action:example:doctor.run", "Owner: component:worker", "Requires root: yes"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("action inspection missing %q:\n%s", want, output.String())
		}
	}

	output.Reset()
	if err := app.inspect(graph, []string{"observation:example:doctor:1"}, true); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `"id": "observation:example:doctor:1"`) {
		t.Fatalf("observation JSON did not expose its global ID:\n%s", output.String())
	}

	output.Reset()
	if err := app.explain(graph, []string{"action:example:doctor.run"}, false); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"resolves through component:worker",
		"Declared: HEALTHY",
		"Effective: DEGRADED",
		"component:worker → component:storage",
	} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("action explanation missing %q:\n%s", want, output.String())
		}
	}
}

func TestDoctorTargetMustResolveToExactlyOneAction(t *testing.T) {
	rootID := model.ID("implementation:example")
	doctorID := model.ID("component:example:doctor")
	firstID := model.ID("action:example:doctor.quick")
	secondID := model.ID("action:example:doctor.deep")
	graph, err := model.NewGraph(
		[]model.Implementation{{ID: rootID, Components: []model.ID{rootID, doctorID}}},
		[]model.Component{
			{ID: rootID, Kind: "implementation", Health: model.Health{Status: model.HealthHealthy}, Children: []model.ID{doctorID}},
			{ID: doctorID, Kind: "doctor", Health: model.Health{Status: model.HealthUnknown}, Actions: []model.Action{
				{ID: firstID, Kind: "doctor.run", Label: "Quick", Target: doctorID},
				{ID: secondID, Kind: "doctor.run", Label: "Deep", Target: doctorID},
			}},
		},
		nil,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := &inspector.Snapshot{Graph: graph, Actions: map[model.ID]inspector.Action{
		firstID:  {ComponentID: doctorID, Spec: manifest.ActionSpec{ID: string(firstID), Kind: "doctor.run"}},
		secondID: {ComponentID: doctorID, Spec: manifest.ActionSpec{ID: string(secondID), Kind: "doctor.run"}},
	}}
	if _, err := findDoctorAction(snapshot, "example"); err == nil || !strings.Contains(err.Error(), "multiple doctor actions") {
		t.Fatalf("implementation doctor resolution error = %v", err)
	}
	if id, err := findDoctorAction(snapshot, string(firstID)); err != nil || id != firstID {
		t.Fatalf("exact action resolution = %s, %v", id, err)
	}
}
