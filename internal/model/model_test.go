package model

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestNewGraphLookupAndCanonicalJSON(t *testing.T) {
	root := Component{
		ID:      "implementation:dvd-ingester",
		Kind:    "implementation",
		Health:  Health{Status: HealthHealthy},
		Summary: "Ingests and publishes optical media.",
		What:    "DVD Ingester",
		Why:     "Make archival ingest repeatable.",
		Metadata: Metadata{
			"z-last":  "last",
			"a-first": "first",
		},
		Children: []ID{"component:publisher"},
		Responsibilities: []string{
			"ingest media",
			"publish completed work",
		},
		FailureModes: []FailureMode{{ID: "nas-unavailable", Summary: "Cold storage cannot be reached."}},
	}
	publisher := Component{ID: "component:publisher", Kind: "systemd.service", Health: Health{Status: HealthHealthy}}

	graph, err := NewGraph(
		[]Implementation{{
			ID:         root.ID,
			Version:    "1.0.3",
			Components: []ID{root.ID, publisher.ID},
		}},
		[]Component{root, publisher},
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("NewGraph() error = %v", err)
	}
	if got, ok := graph.Lookup(publisher.ID); !ok || got.Kind != publisher.Kind {
		t.Fatalf("Lookup(%q) = %#v, %v", publisher.ID, got, ok)
	}
	if got, ok := graph.LookupImplementation(root.ID); !ok || got.Version != "1.0.3" {
		t.Fatalf("LookupImplementation(%q) = %#v, %v", root.ID, got, ok)
	}

	first, err := json.Marshal(graph)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	second, err := json.Marshal(graph)
	if err != nil {
		t.Fatalf("json.Marshal() second error = %v", err)
	}
	if string(first) != string(second) {
		t.Fatalf("JSON is not deterministic:\n%s\n%s", first, second)
	}
	if !strings.Contains(string(first), `"metadata":{"a-first":"first","z-last":"last"}`) {
		t.Fatalf("metadata did not marshal as a stable JSON object: %s", first)
	}
	if !strings.Contains(string(first), `"what":"DVD Ingester"`) || !strings.Contains(string(first), `"failure_modes"`) {
		t.Fatalf("literate fields are missing from JSON: %s", first)
	}
}

func TestRecursiveHealthDerivationAndExplanation(t *testing.T) {
	rootID := ID("implementation:dvd-ingester")
	publisherID := ID("component:publisher")
	nasID := ID("component:nas")
	graph := mustGraph(t,
		[]Implementation{{ID: rootID, Components: []ID{rootID, publisherID, nasID}}},
		[]Component{
			{ID: rootID, Kind: "implementation", Health: Health{Status: HealthHealthy}, Children: []ID{publisherID}},
			{ID: publisherID, Kind: "systemd.service", Health: Health{Status: HealthHealthy}},
			{ID: nasID, Kind: "capability.storage", Health: Health{Status: HealthDegraded, Summary: "mount is read-only"}},
		},
		[]Edge{{From: publisherID, Type: EdgeDependsOn, To: nasID}},
		nil,
	)

	health, err := graph.DerivedHealth(rootID)
	if err != nil {
		t.Fatalf("DerivedHealth() error = %v", err)
	}
	if health.Status != HealthDegraded {
		t.Fatalf("DerivedHealth().Status = %q, want %q", health.Status, HealthDegraded)
	}
	if !strings.Contains(health.Summary, string(nasID)) {
		t.Fatalf("DerivedHealth().Summary = %q, want propagated cause", health.Summary)
	}

	explanation, err := graph.ExplainHealth(rootID)
	if err != nil {
		t.Fatalf("ExplainHealth() error = %v", err)
	}
	if explanation.Declared.Status != HealthHealthy || explanation.Effective.Status != HealthDegraded {
		t.Fatalf("ExplainHealth() = %#v", explanation)
	}
	wantPath := []HealthStep{
		{From: rootID, Relationship: EdgeOwns, To: publisherID},
		{From: publisherID, Relationship: EdgeDependsOn, To: nasID},
	}
	if len(explanation.Causes) != 1 || explanation.Causes[0].ComponentID != nasID {
		t.Fatalf("ExplainHealth().Causes = %#v", explanation.Causes)
	}
	if !reflect.DeepEqual(explanation.Causes[0].Path, wantPath) {
		t.Fatalf("cause path = %#v, want %#v", explanation.Causes[0].Path, wantPath)
	}

	if err := graph.MaterializeDerivedHealth(); err != nil {
		t.Fatalf("MaterializeDerivedHealth() error = %v", err)
	}
	root, _ := graph.Lookup(rootID)
	if root.Health.Status != HealthDegraded || root.DeclaredHealth == nil || root.DeclaredHealth.Status != HealthHealthy {
		t.Fatalf("materialized root health = %#v, declared = %#v", root.Health, root.DeclaredHealth)
	}
	materialized, err := graph.ExplainHealth(rootID)
	if err != nil {
		t.Fatalf("ExplainHealth() after materialization error = %v", err)
	}
	if len(materialized.Causes) != 1 || materialized.Causes[0].ComponentID != nasID || !reflect.DeepEqual(materialized.Causes[0].Path, wantPath) {
		t.Fatalf("materialized causes = %#v, want NAS path %#v", materialized.Causes, wantPath)
	}
	if err := graph.MaterializeDerivedHealth(); err != nil {
		t.Fatalf("second MaterializeDerivedHealth() error = %v", err)
	}
	root, _ = graph.Lookup(rootID)
	if root.DeclaredHealth == nil || root.DeclaredHealth.Status != HealthHealthy {
		t.Fatalf("second materialization replaced declared health: %#v", root.DeclaredHealth)
	}
}

func TestHealthDerivationConvergesAcrossDependencyCycle(t *testing.T) {
	rootID := ID("implementation:cycle")
	aID := ID("component:a")
	bID := ID("component:b")
	graph := mustGraph(t,
		[]Implementation{{ID: rootID, Components: []ID{rootID, aID, bID}}},
		[]Component{
			{ID: rootID, Kind: "implementation", Health: Health{Status: HealthHealthy}, Children: []ID{aID}},
			{ID: aID, Kind: "service", Health: Health{Status: HealthHealthy}},
			{ID: bID, Kind: "service", Health: Health{Status: HealthUnhealthy}},
		},
		[]Edge{
			{From: aID, Type: EdgeDependsOn, To: bID},
			{From: bID, Type: EdgeDependsOn, To: aID},
		},
		nil,
	)

	for _, id := range []ID{rootID, aID, bID} {
		health, err := graph.DerivedHealth(id)
		if err != nil {
			t.Fatalf("DerivedHealth(%q) error = %v", id, err)
		}
		if health.Status != HealthUnhealthy {
			t.Errorf("DerivedHealth(%q).Status = %q, want %q", id, health.Status, HealthUnhealthy)
		}
	}
}

func TestExplainDependenciesInBothDirections(t *testing.T) {
	rootID := ID("implementation:dvd-ingester")
	publisherID := ID("component:publisher")
	queueID := ID("component:queue")
	nasID := ID("component:nas")
	graph := mustGraph(t,
		[]Implementation{{ID: rootID, Components: []ID{rootID, publisherID, queueID, nasID}}},
		[]Component{
			{ID: rootID, Kind: "implementation", Health: Health{Status: HealthHealthy}},
			{ID: publisherID, Kind: "service", Health: Health{Status: HealthHealthy}},
			{ID: queueID, Kind: "state", Health: Health{Status: HealthHealthy}},
			{ID: nasID, Kind: "storage", Health: Health{Status: HealthHealthy}},
		},
		[]Edge{
			{From: rootID, Type: EdgeDependsOn, To: publisherID},
			{From: publisherID, Type: EdgeDependsOn, To: queueID},
			{From: queueID, Type: EdgeDependsOn, To: nasID},
		},
		nil,
	)

	rootExplanation, err := graph.ExplainDependencies(rootID)
	if err != nil {
		t.Fatalf("ExplainDependencies(root) error = %v", err)
	}
	if got := pathFor(rootExplanation.DependsOn, nasID); !reflect.DeepEqual(got, []ID{rootID, publisherID, queueID, nasID}) {
		t.Fatalf("root -> NAS path = %#v", got)
	}

	nasExplanation, err := graph.ExplainDependencies(nasID)
	if err != nil {
		t.Fatalf("ExplainDependencies(NAS) error = %v", err)
	}
	if got := pathFor(nasExplanation.RequiredBy, rootID); !reflect.DeepEqual(got, []ID{nasID, queueID, publisherID, rootID}) {
		t.Fatalf("NAS required-by root path = %#v", got)
	}
}

func TestDoctorObservationIsEvidence(t *testing.T) {
	when := time.Date(2026, time.July, 10, 15, 4, 5, 0, time.UTC)
	doctorID := ID("component:doctor")
	observation := Observation{
		ID:          "observation:doctor:20260710T150405Z",
		ComponentID: doctorID,
		Kind:        ObservationDoctor,
		ObservedAt:  when,
		DurationMS:  821,
		Checks: []Check{
			{ID: "mqtt", Status: CheckPass},
			{ID: "nas", Status: CheckWarn, Summary: "high latency"},
		},
		Artifacts: []Artifact{{URI: "/run/muster/dvd/doctor.log", MediaType: "text/plain"}},
	}
	graph := mustGraph(t,
		[]Implementation{{ID: "implementation:dvd", Components: []ID{"implementation:dvd", doctorID}}},
		[]Component{
			{ID: "implementation:dvd", Kind: "implementation", Health: Health{Status: HealthHealthy}},
			{ID: doctorID, Kind: "doctor", Health: Health{Status: HealthUnknown}},
		},
		nil,
		[]Observation{observation},
	)

	if got := observation.DerivedHealth(); got.Status != HealthDegraded || got.ObservedAt == nil || !got.ObservedAt.Equal(when) {
		t.Fatalf("Observation.DerivedHealth() = %#v", got)
	}
	passing := observation
	passing.Status = ""
	passing.Checks = []Check{{ID: "mqtt", Status: CheckPass}}
	if got := passing.DerivedHealth(); got.Status != HealthHealthy {
		t.Fatalf("passing Observation.DerivedHealth().Status = %q, want %q", got.Status, HealthHealthy)
	}
	latest, ok := graph.LatestObservation(doctorID, ObservationDoctor)
	if !ok || latest.ID != observation.ID {
		t.Fatalf("LatestObservation() = %#v, %v", latest, ok)
	}
	encoded, err := json.Marshal(observation)
	if err != nil {
		t.Fatalf("json.Marshal(observation) error = %v", err)
	}
	for _, fragment := range []string{`"kind":"doctor"`, `"duration_ms":821`, `"checks"`, `"artifacts"`} {
		if !strings.Contains(string(encoded), fragment) {
			t.Errorf("observation JSON %s does not contain %s", encoded, fragment)
		}
	}
}

func TestValidationRejectsBrokenGlobalReferences(t *testing.T) {
	_, err := NewGraph(
		[]Implementation{{ID: "implementation:broken", Components: []ID{"implementation:broken"}}},
		[]Component{{
			ID:       "implementation:broken",
			Kind:     "implementation",
			Health:   Health{Status: HealthHealthy},
			Children: []ID{"component:missing"},
		}},
		nil,
		nil,
	)
	if err == nil || !strings.Contains(err.Error(), "unknown child") {
		t.Fatalf("NewGraph() error = %v, want unknown child", err)
	}

	_, err = NewGraph(
		[]Implementation{{ID: "implementation:actions", Components: []ID{"implementation:actions", "component:worker"}}},
		[]Component{
			{ID: "implementation:actions", Kind: "implementation", Health: Health{Status: HealthHealthy}, Actions: []Action{{ID: "action:duplicate", Kind: "doctor.run"}}},
			{ID: "component:worker", Kind: "service", Health: Health{Status: HealthHealthy}, Actions: []Action{{ID: "action:duplicate", Kind: "doctor.run"}}},
		},
		nil,
		nil,
	)
	if err == nil || !strings.Contains(err.Error(), "duplicate global object id") {
		t.Fatalf("NewGraph() duplicate action error = %v, want global identity failure", err)
	}

	_, err = NewGraph(
		[]Implementation{
			{ID: "implementation:one", Components: []ID{"implementation:one", "component:shared"}},
			{ID: "implementation:two", Components: []ID{"implementation:two", "component:shared"}},
		},
		[]Component{
			{ID: "implementation:one", Kind: "implementation", Health: Health{Status: HealthHealthy}},
			{ID: "implementation:two", Kind: "implementation", Health: Health{Status: HealthHealthy}},
			{ID: "component:shared", Kind: "service", Health: Health{Status: HealthHealthy}},
		},
		nil,
		nil,
	)
	if err == nil || !strings.Contains(err.Error(), "belongs to both") {
		t.Fatalf("NewGraph() shared ownership error = %v, want one implementation owner", err)
	}

	graph := &Graph{SchemaVersion: CurrentSchemaVersion}
	if _, err := graph.Explain("component:missing"); !errors.Is(err, ErrComponentNotFound) {
		t.Fatalf("Explain() error = %v, want ErrComponentNotFound", err)
	}
}

func mustGraph(t *testing.T, implementations []Implementation, components []Component, edges []Edge, observations []Observation) *Graph {
	t.Helper()
	graph, err := NewGraph(implementations, components, edges, observations)
	if err != nil {
		t.Fatalf("NewGraph() error = %v", err)
	}
	return graph
}

func pathFor(paths []DependencyPath, id ID) []ID {
	for _, path := range paths {
		if path.ComponentID == id {
			return path.Path
		}
	}
	return nil
}
