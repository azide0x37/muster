package lockfile

import (
	"strings"
	"testing"

	"github.com/azide0x37/muster/internal/manifest"
	"github.com/azide0x37/muster/internal/model"
)

func TestValidateRequiresGraphAndExecutableActionsToAgree(t *testing.T) {
	actionID := model.ID("action:example:doctor.run")
	componentID := model.ID("component:example:doctor")
	graph, err := model.NewGraph(
		[]model.Implementation{{ID: "implementation:example", Components: []model.ID{"implementation:example", componentID}}},
		[]model.Component{
			{ID: "implementation:example", Kind: "implementation", Health: model.Health{Status: model.HealthHealthy}},
			{
				ID: componentID, Kind: "doctor", Health: model.Health{Status: model.HealthUnknown},
				Actions: []model.Action{{
					ID: actionID, Kind: "doctor.run", Label: "Run doctor", Target: componentID,
					RequiresRoot: true, RequiresConfirmation: true,
				}},
			},
		},
		nil,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	document := Document{
		Schema: CurrentSchema,
		Source: Source{ManifestSHA256: strings.Repeat("a", 64)},
		Graph:  *graph,
		Adapters: map[model.ID]manifest.SourceSpec{
			componentID: {Adapter: "observation.file", Path: "/run/muster/example/doctor.json"},
		},
		Actions: map[model.ID]Action{
			actionID: {
				ComponentID: componentID,
				Spec: manifest.ActionSpec{
					ID: string(actionID), Kind: "doctor.run", Label: "Run doctor",
					Command: []string{"/opt/example/current/bin/doctor.sh", "--runtime"}, RequiresRoot: true,
				},
			},
		},
	}
	if err := document.Validate(); err != nil {
		t.Fatalf("valid lock rejected: %v", err)
	}

	corrupt := document
	corrupt.Actions = cloneActions(document.Actions)
	action := corrupt.Actions[actionID]
	action.Spec.RequiresRoot = false
	corrupt.Actions[actionID] = action
	if err := corrupt.Validate(); err == nil || !strings.Contains(err.Error(), "requires_root") {
		t.Fatalf("requires_root mismatch error = %v", err)
	}

	corrupt = document
	corrupt.Actions = map[model.ID]Action{}
	if err := corrupt.Validate(); err == nil || !strings.Contains(err.Error(), "no executable declaration") {
		t.Fatalf("missing executable action error = %v", err)
	}

	corrupt = document
	corrupt.Adapters = map[model.ID]manifest.SourceSpec{"component:missing": {Adapter: "static"}}
	if err := corrupt.Validate(); err == nil || !strings.Contains(err.Error(), "unknown component") {
		t.Fatalf("unknown adapter component error = %v", err)
	}

	corrupt = document
	corrupt.Graph = document.Graph.Clone()
	declared := model.Health{Status: model.HealthHealthy}
	corrupt.Graph.Components[0].DeclaredHealth = &declared
	if err := corrupt.Validate(); err == nil || !strings.Contains(err.Error(), "runtime-only declared_health") {
		t.Fatalf("materialized lock health error = %v", err)
	}

	corrupt = document
	corrupt.Graph = document.Graph.Clone()
	for componentIndex := range corrupt.Graph.Components {
		for actionIndex := range corrupt.Graph.Components[componentIndex].Actions {
			corrupt.Graph.Components[componentIndex].Actions[actionIndex].RequiresConfirmation = false
		}
	}
	if err := corrupt.Validate(); err == nil || !strings.Contains(err.Error(), "must require confirmation") {
		t.Fatalf("doctor confirmation error = %v", err)
	}
}

func cloneActions(source map[model.ID]Action) map[model.ID]Action {
	result := make(map[model.ID]Action, len(source))
	for id, action := range source {
		action.Spec.Command = append([]string(nil), action.Spec.Command...)
		result[id] = action
	}
	return result
}
